/*
 Copyright (c) 2015, Northeastern University
 All rights reserved.

 Redistribution and use in source and binary forms, with or without
 modification, are permitted provided that the following conditions are met:
     * Redistributions of source code must retain the above copyright
       notice, this list of conditions and the following disclaimer.
     * Redistributions in binary form must reproduce the above copyright
       notice, this list of conditions and the following disclaimer in the
       documentation and/or other materials provided with the distribution.
     * Neither the name of the Northeastern University nor the
       names of its contributors may be used to endorse or promote products
       derived from this software without specific prior written permission.

 THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
 ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
 WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
 DISCLAIMED. IN NO EVENT SHALL Northeastern University BE LIABLE FOR ANY
 DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
 (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
 LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
 ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
 SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

// Package util contains utilities uses throughout the packages
package util

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/NEU-SNS/ReverseTraceroute/log"
	guuid "github.com/google/uuid"
)

const (
	// IP is the index for the ip in a split of an address string
	IP = 0
	// PORT is the index for the port in a split of an address string
	PORT = 1
)

var (
	// ErrorInvalidIP is the error if the IP is invalid
	ErrorInvalidIP = errors.New("invalid IP address")
	// ErrorInvalidPort is the error if the Port is invalid
	ErrorInvalidPort = errors.New("invalid port")
)

// IsDir checks if what is at path dir is a directory
func IsDir(dir string) (bool, error) {
	fi, err := os.Stat(dir)
	if err != nil {
		return false, err
	}
	return fi.IsDir(), nil
}

// MakeDir create a directory at path path with permissions mode
func MakeDir(path string, mode os.FileMode) error {
	return os.Mkdir(path, mode)
}

// ParseAddrArg checks if the address in ip:port notation is valid
func ParseAddrArg(addr string) (int, net.IP, error) {
	ip, port, err := net.SplitHostPort(addr)
	if err != nil {
		return 0, nil, err
	}
	//shortcut, maybe resolve?
	if ip == "localhost" {
		ip = "127.0.0.1"
	}
	pport, err := strconv.Atoi(port)
	if err != nil {
		log.Errorf("Failed to parse port")
		return 0, nil, err
	}
	if pport < 1 || pport > 65535 {
		log.Errorf("Invalid port passed to Start: %d", pport)
		return 0, nil, ErrorInvalidPort
	}
	var pip net.IP
	var cont bool
	if ip == "" {
		pip = nil
		cont = true
	} else {
		pip = net.ParseIP(ip)
	}
	if pip == nil && !cont {
		log.Errorf("Invalid IP passed to Start: %s", ip)
		return 0, nil, ErrorInvalidIP
	}
	return pport, pip, nil
}

// CloseStdFiles closes the standard file descriptors
func CloseStdFiles(c bool) {
	if !c {
		return
	}
	log.Info("Closing standard file descripters")
	err := os.Stdin.Close()

	if err != nil {
		log.Error("Failed to close Stdin")
		os.Exit(1)
	}
	err = os.Stderr.Close()
	if err != nil {
		log.Error("Failed to close Stderr")
		os.Exit(1)
	}
	err = os.Stdout.Close()
	if err != nil {
		log.Error("Failed to close Stdout")
		os.Exit(1)
	}
}

// ConnToRW converts a net.Conn to a bufio.ReadWriter
func ConnToRW(c net.Conn) *bufio.ReadWriter {
	w := bufio.NewWriter(c)
	r := bufio.NewReader(c)
	rw := bufio.NewReadWriter(r, w)
	return rw
}

// ConvertBytes converts an input byte slice to an output byte slice using the
// executable at path
func ConvertBytes(path string, b []byte) ([]byte, error) {
	cmd := exec.Command(path)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	err = cmd.Start()
	if err != nil {
		return nil, err
	}
	_, err = stdin.Write(b)
	if err != nil {
		return nil, err
	}
	err = stdin.Close()
	if err != nil {
		return nil, err
	}
	res := make([]byte, 100*1024)
	n, err := stdout.Read(res)
	if err != nil {
		return res, err
	}
	err = cmd.Wait()
	if err != nil {
		return nil, err
	}
	return res[:n], err
}

// Int32ToIPString converts an ip as an uint32 to a string
func Int32ToIPString(ip uint32) (string, error) {
	var a, b, c, d byte
	if ip < 0 || ip > 4294967295 {
		return "", fmt.Errorf("Ip out of range")
	}
	d = byte(ip & 0x000000ff)
	c = byte(ip & 0x0000ff00 >> 8)
	b = byte(ip & 0x00ff0000 >> 16)
	a = byte(ip & 0xff000000 >> 24)
	nip := net.IPv4(a, b, c, d)
	if nip == nil {
		return "", fmt.Errorf("Invalid IP")
	}
	return nip.String(), nil
}

// IPStringToInt32 converts an IP string to a uint32
func IPStringToInt32(ips string) (uint32, error) {
	ip := net.ParseIP(ips)
	if ip == nil {
		return 0, fmt.Errorf("Nil ip in IpToInt32 %s", ip)
	}
	ip = ip.To4()
	if ip == nil {
		return 0, fmt.Errorf("Nil ip.To4 in IpToInt32 %s", ip)
	}
	var res uint32
	res |= uint32(ip[0]) << 24
	res |= uint32(ip[1]) << 16
	res |= uint32(ip[2]) << 8
	res |= uint32(ip[3])
	return res, nil
}

// IPtoInt32 converts a net.IP to an uint32
func IPtoInt32(ip net.IP) (uint32, error) {
	ip = ip.To4()
	var res uint32
	res |= uint32(ip[0]) << 24
	res |= uint32(ip[1]) << 16
	res |= uint32(ip[2]) << 8
	res |= uint32(ip[3])
	return res, nil
}

// Int32ToIP converts an uint32 into a net.IP
func Int32ToIP(ip uint32) net.IP {
	var a, b, c, d byte
	a = byte(ip)
	b = byte(ip >> 8)
	c = byte(ip >> 16)
	d = byte(ip >> 24)
	return net.IPv4(a, b, c, d)
}

// MicroToNanoSec convers microseconds to nanoseconds
func MicroToNanoSec(usec int64) int64 {
	return usec * 1000
}

// GetBindAddr gets the IP of the eth0 like address
func GetBindAddr() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if strings.Contains(iface.Name, "em1") ||
			strings.Contains(iface.Name, "eth0") &&
				uint(iface.Flags)&uint(net.FlagUp) > 0 {
			addrs, err := iface.Addrs()
			if err != nil {
				return "", err
			}
			addr := addrs[0]
			ip, _, err := net.ParseCIDR(addr.String())
			if err != nil {
				return "", err
			}
			return ip.String(), nil
		}
	}
	return "", fmt.Errorf("Didn't find eth0 interface")
}

// ReverseAny reverse a slice
func ReverseAny(s interface{}) {
    n := reflect.ValueOf(s).Len()
    swap := reflect.Swapper(s)
    for i, j := 0, n-1; i < j; i, j = i+1, j-1 {
        swap(i, j)
    }
}

// Find takes a slice and looks for an element in it. If found it will
// return it's key, otherwise it will return -1 and a bool of false.
func Find(slice []string, val string) (int, bool) {
    for i, item := range slice {
        if item == val {
            return i, true
        }
    }
    return -1, false
}

func FindFirstAfterNotEqual(slice [] string, val string) (int, bool) {
	found := false
	for i, item := range slice {
		if item == val {
			found = true
		} else {
			if found {
				return i, true
			}
		}
	}
	return -1, false
}

type PairAddress struct {
	Src string
	Dst string
}

type PairAddressUint32 struct {
	Src uint32
	Dst uint32
}

func SrcTargetsFromFile(path string, separator string, limit int)[] PairAddressUint32 {
	// os.Open() opens specific file in  
    // read-only mode and this return  
    // a pointer of type os. 
    file, err := os.Open(path) 
  
    if err != nil { 
        log.Fatalf("failed to open") 
    } 
		

	defer file.Close()

	reader := bufio.NewReader(file)
	i := 0
	targets := []PairAddressUint32{} 
	for {
		i += 1
		if i > limit && i != -1{
			break
		}
		line, _, err := reader.ReadLine()

		if err == io.EOF {
				break
		}
		tokens := strings.Split(string(line), separator)
		src := tokens[0]
		dst := tokens[1]
		srcI, _ := IPStringToInt32(src)
		dstI, _ := IPStringToInt32(dst)
        targets = append(targets, PairAddressUint32{
			Src: srcI,
			Dst: dstI,
		}) 
	}

    return targets
}

func IPsFromFile(path string) [] string{
	// os.Open() opens specific file in  
    // read-only mode and this return  
    // a pointer of type os. 
    file, err := os.Open(path) 
  
    if err != nil { 
        log.Fatalf("failed to open") 
    } 
  
    // The bufio.NewScanner() function is called in which the 
    // object os.File passed as its parameter and this returns a 
    // object bufio.Scanner which is further used on the 
    // bufio.Scanner.Split() method. 
    scanner := bufio.NewScanner(file) 
  
    // The bufio.ScanLines is used as an  
    // input to the method bufio.Scanner.Split() 
    // and then the scanning forwards to each 
    // new line using the bufio.Scanner.Scan() 
    // method. 
    scanner.Split(bufio.ScanLines) 
    targets := []string{} 
		
    for scanner.Scan() { 
        targets = append(targets, scanner.Text()) 
    } 
  
    // The method os.File.Close() is called 
    // on the os.File object to close the file 
    file.Close() 

    return targets
}

func RunCMD(cmd string) (string, error) {
	CMD := exec.Command("bash", "-c", cmd)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	CMD.Stdout = &stdout
	CMD.Stderr = &stderr
	err := CMD.Run()
	stderrString := stderr.String()
	if err != nil {
		log.Errorf(fmt.Sprintf("Failed to execute command: %s, %s", cmd, fmt.Sprint(err) + ": " + stderrString))
		return stderrString, err
	} else {
		log.Infof("Successfully  executed %s, %s", cmd, stdout.String())
	}
	return stderrString, nil 	
}

func Remove(s []string, r string) []string {
    for i, v := range s {
        if v == r {
            return append(s[:i], s[i+1:]...)
        }
    }
    return s
}

func Shuffle(slice []string) {
	rand.Seed(time.Now().UnixNano())
	for i := len(slice) - 1; i > 0; i-- { // FisherâYates shuffle
		j := rand.Intn(i + 1)
		slice[i], slice[j] = slice[j], slice[i]
	}
}

func GenUUID() string {
	id := guuid.New()
	// fmt.Printf("github.com/google/uuid:         %s\n", id.String())
	return id.String()
}

func (u *UniqueRand) Int(n int) int {
    for {
        i := rand.Intn(n)
        if !u.Generated[i] {
            u.Generated[i] = true
            return i
        }
    }
}

func Min(s []uint32 ) uint32 {
	min := s[0]
	for _, v := range s {
			if (v < min) {
				min = v
			}
	}
	return min
}


func TheoreticalSpoofTimeout(probingRate uint32, pingsByVp map[uint32]uint32) uint32 {
	// Compute the maximum number of packets per vp
	maxPings := uint32(0)
	for _, nPings :=  range (pingsByVp) {
		if nPings > maxPings  {
			maxPings = nPings
		}
	}

	// Rate is 100 pps, so compute the theoretical maximum number of seconds to send the packets 
	theoreticalSeconds := uint32(maxPings / probingRate)

	return theoreticalSeconds

}

func BoolToInt(b bool) int {
	if b {
		return 1
	} else {
		return 0
	}
}

func Exit(status int) {
	os.Exit(status)
}