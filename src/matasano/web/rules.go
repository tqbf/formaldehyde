package web

import (
	"fmt"
	"strconv"
	"bytes"
)

type ruleCode int 

const (
	RuleNum ruleCode = iota
	RuleList 
)

type pipeCode int 

const (
	PipeRadix pipeCode = iota
)

type Pipe struct {
	Code pipeCode
	
	Radix int64
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
	Files []int
	Pipes []Pipe
}

func ParseRules(buf []byte) []Rule { 
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

	l, o := NewLexer(buf)
	go l.Run()	
	
	rules := []Rule{}
	cur := Rule{}
	curPipe := Pipe{}	

	state := sStart
	quote := []byte{}

	err := func(t Token) {
		cur.Invalid = true
		cur.Errtok = string(t.Value)
		state = sErr
	}

	wordtoks := [][]byte{}

	for t := range(o) { 
		fmt.Printf("%s - %s\n", statenames[state], string(t.Value))
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
			if t.Is(TokPunct) && string(t.Value) == "|" {
				state = sPipe
			} else if t.Is(TokNewline) {
				rules = append(rules, cur)
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
				cur.Files = append(cur.Files, len(cur.Strings))

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
			curPipe = Pipe{}

			if t.Is(TokIdent) { 
				switch string(t.Value) { 
				case "radix":
					curPipe.Code = PipeRadix
					state = sRadix
				default:
					err(t)
				}
			} else {
				err(t)
			}

		case sRadix:
			if t.Is(TokNum) { 
				curPipe.Radix, _ = strconv.ParseInt(string(t.Value), 10, 32)
				cur.Pipes = append(cur.Pipes, curPipe)
				state = sMaybePipe
			} else {
				err(t)
			}
		}
	}	

	return rules
}
