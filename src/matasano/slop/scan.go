package main

import (
	"os"
	"io/ioutil"
	"fmt"
	"matasano/web"
)


func main() {
	f, _ := os.Open("req.raw")
	buf, _ := ioutil.ReadAll(f)

	result := web.AnnotateHttp(buf)

	f, _ = os.Open("rules")
	buf, _ = ioutil.ReadAll(f)

	r := web.ParseRules(buf)

	subs, _ := web.RunAnnotated(result, r)

	for s := range(subs) { 
		s.Raw = s.Raw[0:5]
		fmt.Printf("inj: %v\n", s)
	}
}
