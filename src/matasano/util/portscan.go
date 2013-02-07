package util

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"strings"
	"time"
	"strconv"
)

type PortRange struct {
	start, stop, count uint16
}

type PortRangeSet []PortRange

func ParsePortRanges(input string) PortRangeSet {
	res := make([]PortRange, 0, 10)

	for _, r := range(strings.Split(input, ",")) {
		r = strings.Trim(r, " \t\r\n")

		if sep := strings.Index(r, "-"); sep != -1 {
			l := strings.Trim(r[0:sep], " \t\r\n")
			r = strings.Trim(r[sep+1:], " \t\r\n")
			
			lp, err := strconv.ParseInt(l, 10, 16)
			if err != nil { 
				log.Printf("invalid port range \"%s\"", l)
				continue
			}

			rp, err := strconv.ParseInt(r, 10, 16)
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
		for s := r.start; s <= r.stop; s++ { 
			out = append(out, s)
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

type HostPort struct {
	Addr *net.IPAddr
	Port uint16
}

func (self HostPort) String() string {
	return fmt.Sprintf("%s:%d", self.Addr, self.Port)
}

type PortScanPolicy struct{ 
	MaxInFlight int
}

func PortScan(inaddrs []*net.IPAddr, ports PortRangeSet, policy PortScanPolicy) map[HostPort]int {
	ret := make(map[HostPort]int)
	random_port_mask := ports.Randomizer()

	type result struct {
		sockAddr HostPort
		result   int
	}

	var addrs []scanningHost

	for _, e := range(inaddrs) { 
		addrs = append(addrs, scanningHost{
			addr: *e,
		})
	}

	results := make(chan result)

	probe := func(sockAddr HostPort) {
		if sockAddr.Addr == nil { 
			return
		}

		fmt.Printf("%v\n", sockAddr)

		conn, err := net.DialTimeout("tcp",
			sockAddr.String(),
			5*time.Second)

		if conn != nil {			
			defer conn.Close()
		}

		if err == nil {
			results <- result{
				sockAddr: sockAddr,
				result:   1,
			}
		} else {
			results <- result{
				sockAddr: sockAddr,
				result:   0,
			}
		}
	}

	port_sum := ports.Sum()
	total_count := port_sum * len(addrs)

	hostoff := 0
	hostperm := rand.Perm(len(addrs))

	taps := 0
	selector := func() HostPort {
		if taps == total_count { 
			return HostPort{ nil, 0 }
		} 

		taps += 1

		for { 
			hostoff = (hostoff + 1) % len(addrs)
			idx := hostperm[hostoff]
			if addrs[idx].offset < port_sum { 
				port := random_port_mask[addrs[idx].offset]
				addrs[idx].offset += 1
				return HostPort{
					Addr: &addrs[idx].addr,
					Port: port,
				}
			}
		}

		panic("not reached")
		return HostPort{ nil, 0 }
	}

	for i := 0; i < policy.MaxInFlight; i++ { 
		go probe(selector())
	}

	for i := 0; i < total_count; i++ { 
		res := <- results
		if res.result > 0 { 
			ret[res.sockAddr] = res.result
		}		

		if i < total_count {
			go probe(selector())
		}
	}

	return ret
}
