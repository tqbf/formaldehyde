package web

import (
	"fmt"
	"bytes"
)

func AnnotateHttp(buf []byte) []byte {
	nodes := []HttpNode{}

	for t := range ParseHttp(buf) {
		urlcomponents := false

		if Debug {
			fmt.Printf("%v: %s\n", t, string(buf[t.Start:t.Stop]))
		}

		switch t.Code {
		case NodeUrlPath:
			fallthrough
		case NodeArgKey:
			fallthrough
		case NodeArgValue:
			urlcomponents = true
			nodes = append(nodes, t)
		case NodeUrl:
			if !urlcomponents {
				nodes = append(nodes, t)
			}
		case NodeBodyArgKey:
			fallthrough
		case NodeBodyArgValue:
			fallthrough
		case NodeCookieName:
			fallthrough
		case NodeCookieValue:
			nodes = append(nodes, t)			
		}
	}

	var out bytes.Buffer
	last := 0
	cur := -1
	sz := len(buf)
	for i := 0; i < sz; i++ {				
		for c := last; c < len(nodes); c++ {
			if nodes[c].Start == uint(i) {
				last = c
				cur = c
				out.Write([]byte("{{"))
			} else if nodes[c].Start > uint(i) {
				break
			}
		}

		if cur != -1 && nodes[cur].Stop == uint(i) {
			out.Write([]byte(fmt.Sprintf("}}(%d)", cur)))
			cur = -1
		}

		if i != (sz - 1) || buf[i] != '\n' {
			out.Write(buf[i:i+1])
		}
	}

	if cur != -1 {
		out.Write([]byte(fmt.Sprintf("}}(%d)", cur)))
	}

	return out.Bytes()
}
