package main

import (
	"io/ioutil"
	"strings"
	"flag"
	"fmt"
	"crypto/tls"
	"errors"
	"log"
	"os"
	"net"
	"time"
)

func TlsCommonName(host string, port *int) (string, error) {
	dport := 443
	if port == nil { 
		port = &dport
	}


	conn, err := net.DialTimeout(
				"tcp", 
				fmt.Sprintf("%s:%d", host, *port), 
				time.Millisecond * 750)


	if err != nil || conn == nil {
		return "", err
	}					

	type completion struct { 
		name string
		err error
	}

	done := make(chan completion)
	go func() { 
	 	session := tls.Client(conn, &tls.Config{
	 		InsecureSkipVerify: true,
	 	})
	 	if err := session.Handshake(); err != nil { 
			done <- completion{ "", err }
	 	}
	 
	 	state := session.ConnectionState()
	 	
	 	if len(state.PeerCertificates) > 0 {
	 		done <- completion{ state.PeerCertificates[0].Subject.CommonName, nil }
	 	}

		done <- completion{ "", errors.New("can't read certificates") }
	}()

	select {
	case v := <- done:
		return v.name, v.err
	case <- time.After(time.Millisecond * 750):
		break
	}

	return "", errors.New("timed out in handshake")
}


func main() {
	port := flag.Int("port", 443, "tls port")
	file := flag.String("hosts", "hosts", "file containing IPs")

	flag.Parse()

	f, err := os.Open(*file)
	if err != nil {
		log.Fatalf("can't open %s: %v", *file, err)
	}
	buf, _ := ioutil.ReadAll(f)

	hosts := []string{}

	for _, line := range(strings.Split(string(buf), "\n")) {
		hosts = append(hosts, line)
	}

	done := make(chan bool)
	requests := make(chan string)
	live := 10
	for i := 0; i < 10; i++ {
		go func() { 
			for { 
	 			host, ok := <- requests
	 			if !ok {
	 				live -= 1
					if live == 0 { 
						close(done)
					}
	 				return
	 			}

				name, err := TlsCommonName(host, port)
				if err == nil {
					fmt.Printf("%s: %s\n", host, name)
				}
			}
		}()
	}

	for _, line := range(hosts) { 
		requests <- line
	}
	
	close(requests)

	_ = <- done
}