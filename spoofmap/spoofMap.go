package spoofmap

import (
	"fmt"
	"sync"
	"time"

	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	env "github.com/NEU-SNS/ReverseTraceroute/environment"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/util"
)

var (
	// ErrorIDInUse is returned when the id of a spoofed request is already in use.
	ErrorIDInUse = fmt.Errorf("The is received is already in use.")
	// ErrorSpoofNotFound is returned when a spoof is received that doesn't have
	// have a matching id
	ErrorSpoofNotFound = fmt.Errorf("Received a spoof with no matching Id")
)

// Sender is the interface for something that can sent a slice of SpoofedProbes
// to an address
type Sender interface {
	Send([]*dm.Probe, uint32) error
}

type spoof struct {
	probe *dm.Probe
	t     time.Time
	spoof dm.Spoof
}

type SpoofMap interface {
	Quit()
	Register(dm.Spoof) error
	Receive(*dm.Probe) error
}

// SpoofMap is used to track spoofed measurement requests
type spoofMap struct {
	sync.Mutex
	recvSpoofsChan  chan *dm.Probe
	spoofs     	    map[uint32]*spoof
	quit            chan struct{}
	transport       Sender
	// A bit hacky here, put the IP address of the controller here...
	ccontrollerAddr uint32         
}

// New creates a SpoofMap
func New(s Sender, ccontrollerAddr string) SpoofMap {
	ccontrollerAddrI, err := util.IPStringToInt32(ccontrollerAddr)
	if err != nil {
		panic(err)
	}
	sm := &spoofMap{
		recvSpoofsChan: make(chan *dm.Probe, env.MaximumFlyingSpoofed),
		spoofs:     make(map[uint32]*spoof),
		transport:  s,
		quit:       make(chan struct{}),
		ccontrollerAddr : ccontrollerAddrI,
	}
	go sm.sendSpoofs()
	go sm.cleanOld()
	return sm
}

// Quit ends the sending loop of the spoofMap
func (s *spoofMap) Quit() {
	close(s.quit)
}

// Register is called when a spoof is desired, corresponds to one packet
func (s *spoofMap) Register(sp dm.Spoof) error {
	s.Lock()
	defer s.Unlock()
	// Do not check if the ID is in use, this should never happen if always incrementing
	if spf, ok := s.spoofs[sp.Id]; ok {
		if time.Since(spf.t) > time.Second*60 {
			s.spoofs[sp.Id] = &spoof{
				t:     time.Now(),
				spoof: sp,
			}
			return nil
		}
		return ErrorIDInUse
	}
	s.spoofs[sp.Id] = &spoof{
		t:     time.Now(),
		spoof: sp,
	}
	return nil
}

// Receive is used when a probe for a spoof is gotten
func (s *spoofMap) Receive(p *dm.Probe) error {
	s.Lock()
	defer s.Unlock()
	log.Debugf("Received probe id %d : measurement from %d to %d with spoofer %d", p.ProbeId, p.Dst, p.Src, p.SpooferIp)
	if sp, ok := s.spoofs[p.ProbeId]; ok {
		pr := *p
		sp.probe = &pr
		return nil
	}
	return ErrorSpoofNotFound

	// The check if the spoof probe is actually a spoofed probe will be done later
	// s.recvSpoofsChan <- p
	// return nil
}

// call in a goroutine
func (s *spoofMap) sendSpoofs() {
	t := time.NewTicker(time.Millisecond * 500)
	defer t.Stop()
	var dests map[uint32][]*dm.Probe
	// batchSpoofedProbes := []*dm.Probe {}
	for {
		select {
		case <-s.quit:
			return
		case <-t.C:
			// if len(batchSpoofedProbes) > 0{
			// 	log.Infof("Sending %d spoofed packets", len(batchSpoofedProbes))
			// 	if err := s.transport.Send(batchSpoofedProbes, s.ccontrollerAddr); err != nil {
			// 		log.Error(err)
			// 		continue
			// 	}
			// 	batchSpoofedProbes = nil
			// }
			
			s.Lock()
			dests = make(map[uint32][]*dm.Probe)
			for id, spoof := range s.spoofs {
				if spoof.probe != nil {
					dests[spoof.probe.SenderIp] = append(dests[spoof.probe.SenderIp], spoof.probe)
					spoof.probe = nil
					delete(s.spoofs, id)
				}
			}
			for ip, probes := range dests {
				if err := s.transport.Send(probes, ip); err != nil {
					log.Error(err)
					continue
				}
				log.Infof("Sent %d spoofed responses", len(probes))
				delete(dests, ip)
			}
			s.Unlock()
		// case p := <-s.recvSpoofsChan:
		// 	batchSpoofedProbes = append(batchSpoofedProbes, p)
		}
	}
}

// run this in a goroutine
// spoofed probes may never get responses
// clean out old ones so the memory doesn't grow quite as much
func (s *spoofMap) cleanOld() {
	
	t := time.NewTicker(time.Minute)
	// var count int
	for {
		select {
		case <-s.quit:
			return
		case <-t.C:
			startCleaningTime := time.Now().Unix()
			log.Infof("Cleaning old spoofed IDs %d", startCleaningTime)
			s.Lock()
			// count = 0
			for id, spoof := range s.spoofs {
				if time.Since(spoof.t) > time.Second*120 {
					spoof.probe = nil
					delete(s.spoofs, id)
				}
				// count++
				// if count == 10000 {
				// 	break
				// }
			}
			s.Unlock()
			endCleaningTime := time.Now().Unix()
			log.Infof("Cleaning old spoofed IDs %d", endCleaningTime - startCleaningTime)
		}
	}
}
