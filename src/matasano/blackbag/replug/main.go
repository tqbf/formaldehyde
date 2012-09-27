package main

import (
	"matasano/blackbag"
	"os"
	"regexp"
	"log"
	"strconv"
)

var (
	spec_exp = regexp.MustCompile("(.*?):([0-9]+)@?([0-9]+)?")
)

func main() {
	var wait chan bool

	for i := 1; i < len(os.Args); i++ {
		rest := os.Args[i]

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
	 
		wait = r.Wait

	 	go r.Loop()
	}

	<- wait	
}
