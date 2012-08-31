package main

import (
	"fmt"
	"log"
	"bytes"
	"matasano/ca"
	"matasano/proxy"
)

type Handler struct {

}

func (self Handler) HandleRequest(raw *bytes.Buffer) {
	fmt.Print(raw.String())
}

func (self Handler) HandleResponse(raw *bytes.Buffer) {
	fmt.Print(raw.String())
}

func main() { 
	h := Handler{}
	ca, err := ca.NewCA("/tmp", "/tmp/cert.der", "/tmp/key.der")
	if err != nil { 
		log.Fatal(err)
	}
	p, err := proxy.NewProxy(":7777", ca, h)
	if err != nil {
		log.Fatal(err)
	}

	go p.Loop()

	<- p.Completion
}
