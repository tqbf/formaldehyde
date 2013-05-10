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

	web.Debug = true
	result := web.AnnotateHttp(buf)

	fmt.Println(string(result))

	f, _ = os.Open("rules")
	buf, _ = ioutil.ReadAll(f)

	r := web.ParseRules(buf)

	for i, rr := range(r) { 
		fmt.Printf("%d: %v\n", i, rr)	
		if rr.Code == web.RuleList {
			for j := 0; j < len(rr.Strings); j++ { 
				fmt.Println(string(rr.Strings[j]))
			}
		}
	}
}
