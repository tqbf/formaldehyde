package web

import (
	"container/list"
	"fmt"
)

var Debug bool = false

func debugf(format string, arg ...interface{}) {
	if Debug {
		fmt.Printf(format, arg...)
	}
}

type nodeCode int

type HttpNode struct {
	Code  nodeCode
	Start uint
	Stop  uint
	Err   string
}

const (
	NodeErr nodeCode = iota
	NodeUrl
	NodeUrlScheme
	NodeArgKey
	NodeArgValue
	NodeUrlPath
	NodeVerb
	NodeVersion
	NodeHeaderKey
	NodeHeaderValue
	NodeCookieName
	NodeCookieValue
	NodeRawBody
	NodeBodyArgKey
	NodeBodyArgValue
	NodeBody
	NodeRequestLineRemnants
)

var nodeNames []string = []string{
	"NodeErr",
	"NodeUrl",
	"NodeUrlScheme",
	"NodeArgKey",
	"NodeArgValue",
	"NodeUrlPath",
	"NodeVerb",
	"NodeVersion",
	"NodeHeaderKey",
	"NodeHeaderValue",
	"NodeCookieName",
	"NodeCookieValue",
	"NodeRawBody",
	"NodeBodyArgKey",
	"NodeBodyArgValue",
	"NodeBody",
	"NodeRequestLineRemnants",
}

func (n HttpNode) String() string { 
	return fmt.Sprintf("%s", nodeNames[n.Code])
}

func (n *HttpNode) Snapshot(buf []byte) []byte {
	if int(n.Start) > (len(buf) - 1) {
		return nil
	}

	if int(n.Stop) >= (len(buf) - 1) {
		return buf[n.Start:len(buf)-1]
	} 

	return buf[n.Start:n.Stop]
}

var (
	collectRestOfLine []tokenCode = []tokenCode{
		TokIdent,
		TokNum,
		TokPunct,
		TokWs,
		TokSlash,
		TokDot,
		TokColon,
		TokEq,
		TokAnd,
		TokQuery,
		TokSemi,
	}

	collectArgKeyValue []tokenCode = []tokenCode{
		TokIdent,
		TokNum,
		TokPunct,
		TokSlash,
		TokDot,
		TokColon,
		TokQuery,
		TokSemi,
	}

	collectUrl []tokenCode = []tokenCode{
		TokIdent,
		TokNum,
		TokPunct,
		TokSlash,
		TokDot,
		TokColon,
		TokEq,
		TokAnd,
		TokQuery,
		TokSemi,
	}

	collectCookie []tokenCode = []tokenCode{
		TokIdent	, 
		TokNum		, 
		TokPunct	, 
		TokSlash	, 
		TokDot		, 
		TokColon	, 
		TokAnd		, 
		TokQuery	, 
	}

	collectCookieValue []tokenCode = []tokenCode{
		TokIdent	, 
		TokNum		, 
		TokEq		,
		TokPunct	, 
		TokSlash	, 
		TokDot		, 
		TokColon	, 
		TokAnd		, 
		TokQuery	, 
	}
	
	collectHeaderKey []tokenCode = []tokenCode{
		TokIdent,
		TokNum,
		TokSlash,
		TokDot,
		TokQuery,
		TokSemi,
		TokPunct,
	}

	collectHeaderValue []tokenCode = []tokenCode{
		TokIdent,
		TokNum,
		TokColon,
		TokSlash,
		TokDot,
		TokQuery,
		TokSemi,
		TokPunct,
		TokEq,
		TokAnd,
		TokWs,
	}
)

type parser struct {
	stack   *list.List
	stacksz uint
	tokens  chan Token
	out     chan HttpNode
	sz	uint
}

func (p *parser) pop() Token {
	var t Token

	if p.stacksz > 0 {
		e := p.stack.Back()
		t = e.Value.(Token)
		p.stacksz -= 1
		p.stack.Remove(e)
		debugf("~-> %v\n", t)
		return t
	}

	t, ok := <-p.tokens
	if !ok {
		debugf("~-> eof\n")
		return Token{
			Cur:  p.sz,
			Pos:  p.sz,
			Code: TokEof,
		}
	}

	debugf("--> %v\n", t)
	return t
}

func (p *parser) errf(format string, args ...interface{}) {
	p.out <- HttpNode{
		Code: NodeErr,	
		Err: fmt.Sprintf(format, args...),
	}	
}

func (p *parser) push(t Token) {
	p.stack.PushBack(interface{}(t))
	p.stacksz += 1
}

func (p *parser) rewind(ts []Token) {
	for i := len(ts); i != 0; i-- {
		p.push(ts[i-1])
	}
}

func parseTryScheme(p *parser, first Token) bool {
	nvalid := []tokenCode{
		TokIdent,
		TokNum,
		TokColon,
	}

	if !first.IsAmong(nvalid) {
		return false
	}

	acc := []Token{ first }

	t := p.pop()
	
	for t.IsAmong(nvalid) {
		if t.Is(TokColon) {
			p.out <- HttpNode{
				Start: first.Cur,
				Stop:  t.Cur - 1,
				Code:  NodeUrlScheme,
			}
			return true
		}

		acc = append(acc, t)
	}

	p.rewind(acc)
	return false
}


func (p *parser) collect(valid []tokenCode) ([]Token, Token) {
	acc := []Token{}
	
	all := false
	if len(valid) == 0 {
		all = true
	}

	var t Token
	for t = p.pop(); (all && !t.Is(TokEof)) || t.IsAmong(valid); t = p.pop() {
		acc = append(acc, t)
	}

	return acc, t
}

func parseKvp(p *parser, key, val nodeCode) bool {
	k, t := p.collect(collectArgKeyValue)
	if len(k) > 0 {
		p.out <- HttpNode{
			Start: k[0].Cur,
			Stop:  k[len(k)-1].Pos,
			Code:  key,
		}
	}

	if t.Is(TokAnd) {
		return true
	}

	if !t.Is(TokEq) {
		p.push(t)
		return false
	}

	k, t = p.collect(collectArgKeyValue)
	if len(k) > 0 {
		p.out <- HttpNode{
			Start: k[0].Cur,
			Stop:  k[len(k)-1].Pos,
			Code:  val,
		}
	}

	if !t.Is(TokAnd) {
		p.push(t)
		return false
	}

	return true
}

func parseUrl(p *parser) bool {
	t := p.pop()
	if !t.IsAmong(collectUrl) {
		return false
	}

	first := t

	canpath := true
	segmenttoks := []Token{}

	parseTryScheme(p, t)

	for t.IsAmong(collectUrl) {
		if canpath {
			if t.Is(TokSlash) {
				if len(segmenttoks) > 0 { 
					p.out <- HttpNode{
						Start: segmenttoks[0].Cur,
						Stop:  segmenttoks[len(segmenttoks)-1].Pos,
						Code:  NodeUrlPath,
					}
				}

				segmenttoks = []Token{}
			} else if t.Is(TokQuery) {
				canpath = false

				if len(segmenttoks) > 0 { 
					p.out <- HttpNode{
						Start: segmenttoks[0].Cur,
						Stop:  segmenttoks[len(segmenttoks)-1].Pos,
						Code:  NodeUrlPath,
					}
				}
				
				segmenttoks = []Token{}
			} else {
				segmenttoks = append(segmenttoks, t)
			}
		} else {
			p.push(t)
			parseKvp(p, NodeArgKey, NodeArgValue)
		}

		t = p.pop()
	}

	if len(segmenttoks) > 0 { 
		p.out <- HttpNode{
			Start: segmenttoks[0].Cur,
			Stop:  segmenttoks[len(segmenttoks)-1].Pos,
			Code:  NodeUrlPath,
		}
	}


	p.out <- HttpNode{
		Start: first.Cur,
		Stop:  t.Cur,
		Code:  NodeUrl,
	}
	p.push(t)
	return true
}

func parseRequestLine(p *parser) bool {
	var t Token

	if t := p.pop(); !t.Is(TokIdent) {
		return false
	}

	p.out <- HttpNode{
		Start: t.Cur,
		Stop:  t.Pos,
		Code:  NodeVerb,
	}

	if t = p.pop(); !t.Is(TokWs) {
		return false
	} else {
		for t = p.pop(); t.Is(TokWs); {
			t = p.pop()
		}
	}

	p.push(t)
	parseUrl(p)

	if t = p.pop(); !t.Is(TokWs) {
		return false
	}

	if t = p.pop(); !t.Is(TokIdent) {
		return false
	}
	first := t

	if t = p.pop(); !t.Is(TokSlash) {
		return false
	}

	if t = p.pop(); !t.Is(TokNum) {
		return false
	}

	if t = p.pop(); !t.Is(TokDot) {
		return false
	}

	if t = p.pop(); !t.Is(TokNum) {
		return false
	}

	p.out <- HttpNode{
		Start: first.Cur,
		Stop:  t.Pos,
		Code:  NodeVersion,
	}

	if t = p.pop(); !t.Is(TokNewline) {
		return false
	}

	return true
}

func parseCookie(p *parser) bool {
	var nxt Token
	var all []Token

	for {
		all, nxt = p.collect(collectCookie)

		if len(all) > 0 { 
			p.out <- HttpNode{
				Start: all[0].Cur,
				Stop: all[len(all)-1].Pos,
				Code: NodeCookieName,
			}
		} 
	
		if nxt.Is(TokEq) {
			all, nxt = p.collect(collectCookieValue)
			if len(all) > 0 {
	 			p.out <- HttpNode{
	 				Start: all[0].Cur,
	 				Stop: all[len(all)-1].Pos,
	 				Code: NodeCookieValue,
	 			}
			}
		} 

	
		for nxt.Is(TokWs) {
			nxt = p.pop()
		}

		if !nxt.Is(TokSemi) {
			break
		}

		nxt = p.pop()

		for nxt.Is(TokWs) {
			nxt = p.pop()
		}

		p.push(nxt)
	}

	if !nxt.Is(TokNewline) {
		p.push(nxt)
		return false
	}

	return true
}

func parseHeader(p *parser) bool {
	cookie := false

	all, nxt := p.collect(collectHeaderKey)
	if len(all) > 0 {
		if len(all) == 1 && string(all[0].Value) == "Cookie" {
			cookie = true
		}

		p.out <- HttpNode{
			Start: all[0].Cur,
			Stop:  all[len(all)-1].Pos,
			Code:  NodeHeaderKey,
		}
	}

	if !nxt.Is(TokColon) {
		return false
	}

	all, nxt = p.collect([]tokenCode{TokWs})
	p.push(nxt)

	if !cookie {
		all, nxt = p.collect(collectHeaderValue)
		if len(all) > 0 {
			p.out <- HttpNode{
				Start: all[0].Cur,
				Stop:  all[len(all)-1].Pos,
				Code:  NodeHeaderValue,
			}
		}

		if nxt.Is(TokNewline) {
			return true
		}

		return false
	}

	return parseCookie(p)
}

func parseBody(p *parser) bool {
	first := p.pop()
	p.push(first)

	c := 0
	
	for parseKvp(p, NodeBodyArgKey, NodeBodyArgValue) {
		c += 1
	}

	if c == 0 { 
		return false
	}

	last := p.pop()
	p.out <- HttpNode{
		Start: first.Cur,
		Stop: last.Cur - 1,
		Code: NodeBody,
	}

	return true
}

func ParseHttp(buf []byte) chan HttpNode {
	l, o := NewLexer(buf)
	go l.run()

	p := parser{
		sz:	uint(len(buf)),
		tokens: o,
		stack:  list.New(),
		out:    make(chan HttpNode),
	}

	go func() {
		if !parseRequestLine(&p) {
			all, nxt := p.collect(collectRestOfLine)
			if !nxt.Is(TokNewline) {
				p.errf("REQUEST-LINE-MANGLED-IRREPERABLY")
				return
			} else {				
				p.errf("REQUEST-LINE-MANGLED")
				if len(all) > 0 { 
					p.out <- HttpNode{
						Start: all[0].Cur,
						Stop: all[len(all)-1].Pos,
						Code: NodeRequestLineRemnants,
					}
				}
			}
		}

		for parseHeader(&p) {
			// for side effect
		}

		if !parseBody(&p) {
			all, _ := p.collect([]tokenCode{})
			if len(all) > 0 { 
				p.out <- HttpNode{
					Start: all[0].Cur,
					Stop: all[len(all)-1].Pos,
					Code: NodeRawBody,
				}
			}
		}

		close(p.out)
	}()

	return p.out
}

