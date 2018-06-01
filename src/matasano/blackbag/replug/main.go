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

		ptls := func(s string) (int, error, bool)  {
			if s[0] == '+' {
				p, e := strconv.ParseInt(s[1:], 0, 32)
				return int(p), e, true
			} 

			p, e := strconv.ParseInt(s[0:], 0, 32)
			return int(p), e, false
		}
	 
	 	host := string(m[1])

		var (
			port, lport int
			err error
			tls, ltls bool
		)

		port, err, tls = ptls(string(m[2]))
		if err != nil {
			log.Fatal("can't parse outgoing port: ", err)
		}

		if len(m) > 2 && len(m[3]) > 0 {
			lport, err, ltls = ptls(string(m[3]))
			if err != nil { 
				log.Fatal("can't parse incoming port: ", err)
			}
		} else {
			lport = port
			ltls = tls
		}
	 	
	 	r, e := blackbag.NewReplug(host, int(port), tls, int(lport), ltls)
	 	if e != nil {
	 		log.Fatal("Bind: ", e)
	 	}
	 
		wait = r.Wait

	 	go r.Loop()
	}

	<- wait	
}
