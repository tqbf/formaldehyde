package web

import (
	"fmt"
	"bytes"
	"strconv"
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


type Injection struct {
	Raw []byte
	RuleIndex int
	RuleIteration int
}

const (
	spre = iota
	sin
	spostopen
	spost	
)

type specialFrag struct {
	ruleIndex int		
}

func findFrags(buf []byte) ([][]byte, map[int]specialFrag) {
	l := len(buf)

	specialFrags := make(map[int]specialFrag)
	frags := [][]byte{}

	state := spre
	fragc := 0	
	postfirst := 0
	ab := bytes.Buffer{}

	for i := 0; i < l; i++ { 
		switch state {
		case spre:
			if buf[i] == '{' && i < (l-1) && buf[i+1] == '{' {
				out := make([]byte, len(ab.Bytes()))
				copy(out, ab.Bytes())
				ab.Reset()
				frags = append(frags, out)
				fragc += 1
				i += 1 // skip next '{'
				state = sin
			} else {
				ab.Write(buf[i:i+1])
			}
		case sin:
			if buf[i] == '}' && i < (l-1) && buf[i+1] == '}' { 
				out := make([]byte, len(ab.Bytes()))
				copy(out, ab.Bytes())
				ab.Reset()
				frags = append(frags, out)
				specialFrags[fragc] = specialFrag{}
				fragc += 1
				i += 1 
				state = spostopen
			} else {
				ab.Write(buf[i:i+1])
			}
		case spostopen:
			if buf[i] == '(' {
				state = spost
				postfirst = i
			} else {
				state = spre
				ab.Write(buf[i:i+1])
			}		

		case spost:
			if buf[i] == ')' { 
				idx, _ := strconv.ParseInt(string(buf[postfirst:i]), 10, 32)
				f := specialFrags[fragc-1]
				f.ruleIndex = int(idx)
				state = spre				
			}
		}
	}	

	return frags, specialFrags
}

func RunAnnotated(buf []byte, rules Ruleset) (chan Injection, error) {
	frags, specialFrags := findFrags(buf)

	out := make(chan Injection)
	kill := make(chan bool, len(rules))

	readyrules := rules.Run(kill)
	rulesrun := 0

	fmt.Println(rules.Count())

	go func() { 
		ab := bytes.Buffer{}
		fragc := len(frags)	 

	 	for rule := range(readyrules) {
			fmt.Println(rule)
			c := 0			
			for injection := range(rule.Feed) {
				ab.Reset()
 
				for i := 0; i < fragc; i++ {
					if f, ok := specialFrags[i]; ok && f.ruleIndex == rule.Index { 
						ab.Write(injection)
					} else {
						ab.Write(frags[i])						
					}
				}
 
				injection := Injection{
					RuleIndex: rule.Index,
					RuleIteration: c,
				}
 
				injection.Raw = make([]byte, len(ab.Bytes()))
				copy(injection.Raw, ab.Bytes())
 
				out <- injection
 
				c += 1
			}

			rulesrun += 1
			if rulesrun >= rules.Count() { 
				break
			}
	 	}
		
		for _, _ = range(rules) { 
			kill <- true
		}

		close(out)
	}()

	return out, nil
}

