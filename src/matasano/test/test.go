package main

import (
 	"fmt"
// 	"log"
// 	"bytes"
// 	"net"
	"flag"
	"matasano/util"
)

func main() { 
	flag.Parse()
	
	r := util.ParsePortRanges(flag.Arg(0))

	for _, e := range(r) { 
		fmt.Println(e)
	}

	for _, e := range(r.Randomizer()) {
		fmt.Println(e)
	}
}

// type Handler struct {
//  
// }
//  
// func (self Handler) HandleRequest(raw *bytes.Buffer) {
//  	fmt.Print(raw.String())
// }
//  
// func (self Handler) HandleResponse(raw *bytes.Buffer) {
//  	fmt.Print(raw.String())
// }

//func main() { 
// 	var buf [1024]byte
// 	listener, err := net.Listen("tcp", ":7778")
// 	conn, err := listener.Accept()
// 	l, err := conn.Read(buf[0:])
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	conn.Write([]byte("HTTP/1.1 200 Connection established\r\n\r\n"))
// 
// 	ca, err := ca.NewCA("/tmp", "/tmp/cert.der", "/tmp/key.der")
// 	cert, err := ca.Lookup("hreservice2-qc.hewitt.com")
// 	conf := tls.Config{
// 		Certificates: []tls.Certificate{*cert},
// 	}
// 
// 	tlsconn := tls.Server(conn, &conf)
// 	l, err = tlsconn.Read(buf[0:2])
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 
// 	log.Print(string(buf[0:l]))
// 	return
// 
////	h := Handler{}
////	ca, err := ca.NewCA("/tmp", "/tmp/cert.der", "/tmp/key.der")
////	if err != nil { 
////		log.Fatal(err)
////	}
////	p, err := proxy.NewProxy(":7778", ca, h)
////	if err != nil {
////		log.Fatal(err)
////	}
////
////	go p.Loop()
////
////	<- p.Completion
//}
