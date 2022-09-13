package controller

import (
	"sync"
	"time"

	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/prometheus/log"
)

type spoofMap struct {
	sync.Mutex
	sm map[uint32]*channel
	tm map[uint32]int64 // Map for debugging timeouts
}

type channel struct {
	mu     sync.Mutex
	count  int
	ch     chan *dm.Probe
	kill   chan struct{}
	Opened int64
}

func newChannel(ch chan *dm.Probe, kill chan struct{}, count int) *channel {
	return &channel{
		ch:     ch,
		count:  count,
		kill:   kill,
		Opened: time.Now().Unix(),
	}
}

func (c *channel) Send(p *dm.Probe) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.count--
	select {
	case c.ch <- p:
	case <-c.kill:
		return
	}

	if c.count == 0 {
		close(c.ch)
	}
}

// kill must always be closed when this is done
func (sm *spoofMap) Add(notify chan *dm.Probe, kill chan struct{}, ids []uint32) {
	sm.Lock()
	defer sm.Unlock()
	log.Debugf("Adding spoof IDs: %v", ids)
	ch := newChannel(notify, kill, len(ids))
	opened := time.Now().Unix()
	for _, id := range ids {
		sm.sm[id] = ch
		sm.tm[id] = opened
	}
	// When the group is killed, remove any ids that were left over
	go func() {
		select {
		case <-kill:
			sm.Lock()
			defer sm.Unlock()
			for _, id := range ids {
				delete(sm.sm, id)
				// Do not delete the corresponding sent time.
			}
		}
	}()
}

func (sm *spoofMap) Notify(probe *dm.Probe) {
	sm.Lock()
	defer sm.Unlock()
	log.Debug("Notify: ", probe)
	receivedInController := time.Now().Unix()
	
	if c, ok := sm.sm[probe.ProbeId]; ok {
		// log.Infof(`Elapsed for probe: %v, src %d, dst %d, spoof %d, 
		// rx vp %d , rx ctrl %d, 
		// rx ctrl - rx vp %d, rx vp-tx %d , rx ctrl -tx %d`, 
		// probe.ProbeId, probe.Dst, probe.Src, probe.SpooferIp, 
		// probe.ReceivedTimestamp, 
		// receivedInController, receivedInController - probe.ReceivedTimestamp, 
		// probe.ReceivedTimestamp - c.Opened,
		// receivedInController - c.Opened,
		// )
		c.Send(probe)
		delete(sm.sm, probe.ProbeId)
		return
	}
	log.Errorf(`No channel found for probe: %v, src %d, dst %d, spoof %d, sent %d, 
	rx vp %d , rx ctrl %d, 
	rx ctrl - rx vp %d,
	rx vp-tx %d, 
	rx ctrl -tx %d`, 
	probe.ProbeId, probe.Dst, probe.Src, probe.SpooferIp, sm.tm[probe.ProbeId], 
	probe.ReceivedTimestamp, receivedInController, 
	receivedInController - probe.ReceivedTimestamp, 
	probe.ReceivedTimestamp - sm.tm[probe.ProbeId],
	receivedInController - sm.tm[probe.ProbeId],
	)
	
}
