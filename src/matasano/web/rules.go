package web

import (
	"strconv"
	"bytes"
	"fmt"
)

type ruleCode int 

const (
	RuleNum ruleCode = iota
	RuleList 
)

type Pipe interface {
	Transform([]byte) []byte
}

type PipeRadix uint
type PipePad uint

func (p PipeRadix) Transform(buf []byte) []byte {
	i, e := strconv.ParseInt(string(buf), 10, 64)
	if e != nil {
		return buf
	}

	return []byte(strconv.FormatInt(i, int(p)))
}

func (p PipePad) Transform(buf []byte) []byte {
	if len(buf) < int(p) {
		ab := bytes.Buffer{}
		for i := 0; i < (int(p) - len(buf)); i++ { 
			ab.Write([]byte("0"))
		}
		ab.Write(buf)
		return ab.Bytes()
	}

	return buf
}

type Rule struct {
	Invalid bool
	Code ruleCode
	Start int64
	Stop int64
	Step int64
	Index int

	Errtok string

	Strings [][]byte
	Files map[int]bool
	Pipes []Pipe
	
	Live bool
	Feed chan []byte
	Kill chan bool
}

func (r *Rule) Close() {
	r.Live = false
	close(r.Feed)
}

func (r *Rule) Out(buf []byte) bool { 
	for _, p := range(r.Pipes) { 
		buf = p.Transform(buf)
	}

	select { 
	case _, _ = <- r.Kill:
		return false
	case r.Feed <- buf:
	}

	return true
}

func (r *Rule) RunNum() {
	c := r.Start
	if r.Step == 0 {
		r.Step = 1
	}

	for {
		if !r.Out([]byte(strconv.FormatInt(c, 10))) {
			r.Close()
			return
		}
		c += r.Step

		if c > r.Stop {
			r.Close()
			return
		}	
	}
}

func (r *Rule) RunList() {
	c := 0

	for {
		if _, ok := r.Files[c]; ok {
			// XXX not implemented here
		} else {
			if !r.Out(r.Strings[c]) {
				r.Close()
				return
			}
		}

		c += 1

		if c >= len(r.Strings) {
			r.Close()
			return
		}
	}
}

type Ruleset []Rule

func (r *Ruleset) Count() int {
	return len([]Rule(*r))
}

func (r *Ruleset) Run(kill chan bool) (chan Rule) {
	ret := make(chan Rule, r.Count())

	for _, rule := range([]Rule(*r)) {
		rule.Kill = kill
		go rule.Run(ret)
	}

	return ret
}

func (r Rule) Run(ready chan Rule) {
	r.Live = true
	r.Feed = make(chan []byte)

	ready <- r

	switch r.Code {
	case RuleNum:
		r.RunNum()
	case RuleList:
		r.RunList()
	}	
}

func ParseRules(buf []byte) Ruleset { 
	const ( 
		sRadix = iota
		sPipe
		sListWord
		sList
		sMaybePipe
		sMaybeStep
		sStep
		sNumSecond
		sNumDash
		sNum
		sIntArg
		sKind
		sSep	
		sStart
		sErr 
	) 

	statenames := []string{
		"sRadix",
		"sPipe",
		"sListWord",
		"sList",
		"sMaybePipe",
		"sMaybeStep",
		"sStep",
		"sNumSecond",
		"sNumDash",
		"sNum",
		"sKind",
		"sSep",
		"sStart",
		"sErr",
	}

	_ = statenames

	l, o := NewLexer(buf)
	go l.Run()	
	
	rules := []Rule{}
	cur := Rule{}

	cur.Files = make(map[int]bool)

	state := sStart
	quote := []byte{}

	err := func(t Token) {
		cur.Invalid = true
		cur.Errtok = string(t.Value)
		state = sErr
	}

	wordtoks := [][]byte{}

	var intArg func(i int64) Pipe

	for t := range(o) { 
		if t.Is(TokWs) && state != sListWord {
			continue
		}

		switch state {
		case sErr: 
			if t.Is(TokNewline) {
				rules = append(rules, cur)
				state = sStart
				cur = Rule{}	
			}

		case sStart:
			if t.Is(TokNum) {
				i, _ := strconv.ParseInt(string(t.Value), 10, 32)
				cur.Index = int(i)
				state = sSep
			} else { 
				err(t) 
			} 

		case sSep:	
			if t.Is(TokColon) {
				state = sKind
			} else { 
				err(t) 
			} 

		case sKind:
			if t.Is(TokIdent) {
				switch string(t.Value) { 
				case "num":
					cur.Code = RuleNum
					state = sNum
				case "list":
					cur.Code = RuleList
					state = sList
				default:
					err(t)
				}
			} else { 
				err(t) 
			} 

		case sNum:
			if t.Is(TokNum) {
				cur.Start, _ = strconv.ParseInt(string(t.Value), 10, 32)
				state = sNumDash				
			} else { 
				err(t) 
			} 

		case sNumDash:
			if t.Is(TokPunct) && string(t.Value) == "-" {
				state = sNumSecond
			} else { 
				err(t) 
			} 

		case sNumSecond:
			if t.Is(TokNum) { 
				cur.Stop, _ = strconv.ParseInt(string(t.Value), 10, 32)
				state = sMaybeStep
			} else { 
				err(t) 
			} 

		case sStep:
			if t.Is(TokNum) { 
				cur.Step, _ = strconv.ParseInt(string(t.Value), 10, 32)
				state = sMaybePipe
			} else { 
				err(t) 
			} 

		case sMaybeStep:
			if t.Is(TokIdent) && string(t.Value) == "step" {
				state = sStep
				continue
			} else if !t.Is(TokPunct) || string(t.Value) != "|" { 
				err(t) 
				continue
			} 

			fallthrough
		case sMaybePipe:
			fmt.Printf("maybepipe: %s\n", string(t.Value))

			if t.Is(TokPunct) && string(t.Value) == "|" {
				state = sPipe
			} else if t.Is(TokNewline) {
				rules = append(rules, cur)
				fmt.Println("appending")
				state = sStart
				cur = Rule{}	
			} else { 
				err(t) 
			} 

		case sList:
			if t.Is(TokPunct) && string(t.Value) != "|" {
				state = sListWord
				quote = t.Value

			} else if t.Is(TokIdent) {
				cur.Strings = append(cur.Strings, t.Value)
				cur.Files[len(cur.Strings)] = true

			} else if t.Is(TokPunct) && string(t.Value) == "|" { 
				state = sPipe

			} else {
				err(t) 
			} 

		case sListWord:
			if t.Is(TokPunct) && string(t.Value) == string(quote) {
				quote = []byte("")
				bb := bytes.Buffer{}
				for i := 0; i < len(wordtoks); i++ {
					bb.Write(wordtoks[i])
				}
				
				wordtoks = [][]byte{}

				cur.Strings = append(cur.Strings, bb.Bytes())
				state = sList

			} else if t.Is(TokNewline) { 
				err(t)

			} else {
				wordtoks = append(wordtoks, t.Value)
			}

		case sPipe:
			if t.Is(TokIdent) { 
				switch string(t.Value) { 
				case "pad":	
					intArg = func(i int64) Pipe { return PipePad(i) }
					state = sIntArg
				case "radix":
					intArg = func(i int64) Pipe { return PipeRadix(i) }
					state = sIntArg
				default:
					err(t)
				}
			} else {
				err(t)
			}

		case sIntArg:
			if t.Is(TokNum) { 
				v, _ := strconv.ParseInt(string(t.Value), 10, 32)
				cur.Pipes = append(cur.Pipes, intArg(v))
				state = sMaybePipe
			} else {
				err(t)
			}
		}
	}


	return rules
}
