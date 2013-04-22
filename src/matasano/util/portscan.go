package util

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"strings"
	"time"
	"strconv"
	"matasano/ewma"
	"syscall"
	"sync"
	"sync/atomic"
)

type PortRange struct {
	start, stop, count uint16
}

type PortRangeSet []PortRange

const (
	ProbeSuccess = iota
	ProbeRefused
	ProbeFailed
	ProbeTimeout
	ProbeSquelch
)

func ParsePortRanges(input string) PortRangeSet {
	res := make([]PortRange, 0, 10)

	for _, r := range(strings.Split(input, ",")) {
		r = strings.Trim(r, " \t\r\n")

		if sep := strings.Index(r, "-"); sep != -1 {
			l := strings.Trim(r[0:sep], " \t\r\n")
			r = strings.Trim(r[sep+1:], " \t\r\n")
			
			lp, err := strconv.ParseUint(l, 10, 16)
			if err != nil { 
				log.Printf("invalid port range \"%s\"", l)
				continue
			}

			rp, err := strconv.ParseUint(r, 10, 16)
			if err != nil { 
				log.Printf("invalid port range \"%s\"", r)
				continue
			}

			res = append(res, PortRange{
				start: uint16(lp),
				stop: uint16(rp),
				count: uint16(rp - lp) + 1,
			})
		} else {
			port, err := strconv.ParseInt(r, 10, 16); 
			if err != nil {
				log.Printf("invalid port range \"%s\"", r)
				continue
			}

			res = append(res, PortRange{
				start: uint16(port),
				stop: uint16(port),
				count: 1,
			})
		}
	}

	return res
}

type HostPort struct {
	Addr *net.IPAddr
	Port uint16
}

func ProbePortTimeout(addr HostPort, timeout time.Duration) (code int, err error) { 
	code = ProbeFailed
	err = nil

	s, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0) 
	if err != nil { 
		return
	} 

	defer syscall.Close(s)

	// #go-nuts, this isn't ever going to fail
	_ = syscall.SetNonblock(s, true)

	sa := syscall.SockaddrInet4{}
	sa.Port = int(addr.Port)
	b := addr.Addr.IP.To4()
	for i := 0; i < 4; i++ {
		sa.Addr[i] = b[i]
	}

	err = syscall.Connect(s, &sa)
	if err != nil && err != syscall.EINPROGRESS { 
		return
	}

	tv := syscall.NsecToTimeval(int64(timeout))
	
	off := int(s) / 32
	bit := int(s) % 32 
	
	fds := syscall.FdSet{}
	fds.Bits[off] |= 1 << uint(bit)	

	err = syscall.Select(s + 1, nil, &fds, nil, &tv)
	if err != nil { 
		return
	}

	if fds.Bits[off] & (1 << uint(bit)) == 0 {
		code = ProbeTimeout
		return
	}		

	if _, e := syscall.Getpeername(s); e != nil {
		// don't actually care about the error, though this 
		// is a cheat
		code = ProbeRefused
	} else {
		code = ProbeSuccess
	}

	return
}

func (self PortRangeSet) Sum() int {
	ret := 0
	for _, r := range(self) {
		ret += int(r.count)
	}
	return ret
}

func (self PortRangeSet) Randomizer() []uint16 {
	out := make([]uint16, 0, len(self) * 10)

	for _, r := range(self) {
		stop := uint32(r.stop)
		for s := uint32(r.start); s <= stop; s++ {
			out = append(out, uint16(s))
		}	
	}

	for i := len(out) - 1; i > 0; i-- { 
		o := rand.Int() % len(out)
		x := out[o]
		out[o] = out[i]
		out[i] = x		
	}

	return out
}

type scanningHost struct {
	addr   net.IPAddr
	offset int
}

func (self HostPort) String() string {
	return fmt.Sprintf("%s:%d", self.Addr, self.Port)
}

func to4(a net.IP) uint32 {
	v := a.To4()
	return uint32(v[0]) << 24 | uint32(v[1]) << 16 | uint32(v[2]) << 8 | uint32(v[3])
}

type ProbePositive struct {
	SockAddr HostPort
	Elapsed	 time.Duration
	Complete bool
}

type PortScanPolicy interface { 
	ComputeFirstVolley() int
	ComputeTimeout(a net.IP) time.Duration
	Elapsed(a net.IP, d time.Duration, code int)
	ResultChannel() chan ProbePositive
}

type DefaultPolicy struct {
	MaxInFlight int
	LiveResults chan ProbePositive
	
	alltimeout ewma.EWMA
	timeouts map[uint32]ewma.EWMA
	mux sync.Mutex
}

func (self *DefaultPolicy) ResultChannel() chan ProbePositive {
	return self.LiveResults
}

func (self *DefaultPolicy) ComputeFirstVolley() int {
	return self.MaxInFlight
}

func (self *DefaultPolicy) ComputeTimeout(a net.IP) time.Duration {	
	var timeout time.Duration = 5 * time.Second

	self.mux.Lock()
	defer self.mux.Unlock()

	v := self.alltimeout.Read()
	if v != 0 { 
		timeout = (time.Duration(v) * time.Millisecond) * 8
	}

	e, ok := self.timeouts[to4(a)]
	if ok {
		timeout = (time.Duration(e.Read()) * time.Millisecond) * 4
	}

	if timeout < (20 * time.Millisecond) {
		timeout = 20 * time.Millisecond
	}
		
	return timeout
}

func (self *DefaultPolicy) Elapsed(a net.IP, d time.Duration, code int) { 
	var e ewma.EWMA
	var ok bool
	var delt = uint64(d / time.Millisecond) 

	self.mux.Lock()
	defer self.mux.Unlock()	

	if code != ProbeTimeout && delt != 0 && (code == ProbeSuccess || code == ProbeRefused) { 
		self.alltimeout.Add(delt)
	}
	
	if self.timeouts == nil { 
		if code == ProbeTimeout {
			// don't try to get smart if we have no real samples
			return 
		}

		self.timeouts = make(map[uint32]ewma.EWMA)
	}

	if e, ok = self.timeouts[to4(a)]; !ok {
		e = ewma.EWMA{}
	}

	if code == ProbeSuccess || code == ProbeRefused {
		e.Add(delt + 1)
		self.timeouts[to4(a)] = e
	}
}

func PortScan(inaddrs []*net.IPAddr, ports PortRangeSet, policy PortScanPolicy) map[HostPort]int {

	ret := make(map[HostPort]int)
	random_port_mask := ports.Randomizer()

	type result struct {
		sockAddr HostPort
		result   int
		elapsed	 time.Duration
	}

	var addrs []scanningHost

	for _, e := range(inaddrs) { 
		addrs = append(addrs, scanningHost{
			addr: *e,
		})
	}

	results := make(chan result)
	requests := make(chan HostPort)

	allc := uint32(0)

	probe := func(which int) {
		for { 
			sockAddr := <- requests

			if sockAddr.Addr == nil { 
				return
			}

			atomic.AddUint32(&allc, 1)
	
			timeout := policy.ComputeTimeout(sockAddr.Addr.IP)

			t := time.Now()
			code, _ := ProbePortTimeout(sockAddr, timeout)

			policy.Elapsed(sockAddr.Addr.IP, time.Since(t), code)

			// fmt.Printf("(%d) %v %dms (of %dms) %dgr\n", allc, sockAddr, time.Since(t) / time.Millisecond, timeout / time.Millisecond, runtime.NumGoroutine())

			results <- result{
				sockAddr: sockAddr,
				result:   code,
				elapsed:  time.Since(t),
			}
		}
	}

	port_sums := ports.Sum()	
	total_count := port_sums * len(addrs)

	hostoff := 0
	hostperm := rand.Perm(len(addrs))

	go func() { 
	 	for taps := 0; taps < total_count; taps++ { 
	 		for { 
	 			hostoff = (hostoff + 1) % len(addrs)
	 			idx := hostperm[hostoff]
	 			if addrs[idx].offset < port_sums { 
	 				port := random_port_mask[addrs[idx].offset]
	 				addrs[idx].offset += 1
	 				requests <- HostPort{
	 					Addr: &addrs[idx].addr,
	 					Port: port,
	 				}
	 			}
	 		}
	 
	 		panic("not reached")
		}

		for i := 0; i < policy.ComputeFirstVolley(); i++ {
			requests <- HostPort{ nil, 0 }
		}
	}()

	for i := 0; i < policy.ComputeFirstVolley(); i++ { 
		go probe(i)
	}

	for i := 0; i < total_count; i++ { 
		res := <- results
			
		if res.result == ProbeSuccess { 
			ret[res.sockAddr] = res.result

			ch := policy.ResultChannel()

			if ch != nil { 
				ch <- ProbePositive{
					SockAddr: res.sockAddr,
					Elapsed: res.elapsed,
				}
			}
		}		

		if res.result == ProbeSquelch {
			total_count -= 1
			time.Sleep(50 * time.Millisecond)
			requests <- res.sockAddr
		} 
	}

	ch := policy.ResultChannel()
	if ch != nil { 
		close(ch)
	}

	return ret
}
