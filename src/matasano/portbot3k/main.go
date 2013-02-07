package main

import (
	"flag"
	"io/ioutil"
	"log"
	"matasano/util"
	"net"
	"os"
	"strings"
)


func main() {
	ip_in_f := os.Stdin
	infile := flag.String("hosts", "", "file containing IP addresses (default: stdin)")
	max_inflight := flag.Int("max", 10, "maximum in flight requests")

	flag.Parse()

	if *infile != "" {
		var err error
		ip_in_f, err = os.Open(*infile)
		if err != nil { 
			log.Fatalf("Can't open \"%s\": %v", infile, err)
		}
	}

	rset := util.ParsePortRanges(flag.Arg(0))

	buf, err := ioutil.ReadAll(ip_in_f)
	if err != nil {
		log.Fatal("can't read IP input file")
	}

	var addrs []*net.IPAddr

	for _, line := range strings.Split(string(buf), "\n") {
		line = strings.Trim(line, " \t")
		if addr, err := net.ResolveIPAddr("ip4", line); err != nil {
			log.Printf("invalid IP address \"%s\" (continuing)", line)
			continue
		} else {
			addrs = append(addrs, addr)
		}
	}

	result := util.PortScan(addrs, rset, util.PortScanPolicy{
		MaxInFlight: *max_inflight,
	})

	for sockAddr, _ := range(result) {
		log.Printf("%s\n", sockAddr)
	}
}
