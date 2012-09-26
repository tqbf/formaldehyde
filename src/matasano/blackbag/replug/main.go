package main

import (
	"matasano/blackbag"
	"os"
	"strings"
	"regexp"
	"log"
	"strconv"
)

var (
	spec_exp = regexp.MustCompile("(.*?):([0-9]+)@?([0-9]+)?")
)

func main() {
	rest := strings.Join(os.Args[1:], " ")
	
	m := spec_exp.FindSubmatch([]byte(rest))
	if m == nil { 
		log.Fatal("replug target[:port[@lport]]")		
	} 

	host := string(m[1])
	port, _ := strconv.ParseInt(string(m[2]), 0, 32)
	lport := port
	if len(m) > 2 {
		lport, _ = strconv.ParseInt(string(m[3]), 0, 32)
	}
	
	r, e := blackbag.NewReplug(host, int(port), int(lport))
	if e != nil {
		log.Fatal("Bind: ", e)
	}

	go r.Loop()
	
	<- r.Wait
}
