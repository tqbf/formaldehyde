package web

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// this is obviously cadged from Pike's talk, is overkill, and pretty clumsy; I was
// just playing with the idea.

type tokenCode int

const (
	TokErr tokenCode = iota
	TokEof
	TokIdent
	TokNum
	TokPunct
	TokWs
	TokNewline
	TokSlash
	TokDot
	TokColon
	TokEq
	TokAnd
	TokQuery
	TokSemi
)

var tokNames []string = []string{
	"TokErr",
	"TokEof",
	"TokIdent",
	"TokNum",
	"TokPunct",
	"TokWs",
	"TokNewline",
	"TokSlash",
	"TokDot",
	"TokColon",
	"TokEq",
	"TokAnd",
	"TokQuery",
	"TokSemi",
}

type Token struct {
	Code  tokenCode
	Value []byte
	Cur   uint
	Pos   uint
}

var (
	charsWs    string = " \t"
	charsCaps  string = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	charsLower string = strings.ToLower(charsCaps)
	charsAlpha string = strings.Join([]string{
		charsCaps,
		charsLower,
	}, "")
	charsIdent string = strings.Join([]string{
		charsAlpha,
		charsNum,
		"_-%",
	}, "")
	charsNum   string = "0123456789"
	charsHex   string = "0123456789abcdefABCDEF"
	charsPunct string = "!@#$%^&*()_-+={}[]\\\":|;'`~<>,./?"
	charsPrint string = strings.Join([]string{
		charsAlpha,
		charsNum,
		charsPunct,
	}, "")
)

func (t Token) String() string {
	return fmt.Sprintf("%s: %d:%d=%s", tokNames[t.Code], t.Cur, t.Pos, t.Value)
}

func (t Token) Is(k tokenCode) bool {
	if t.Code == k {
		return true
	}
	return false
}

func (t Token) IsAmong(ks []tokenCode) bool {
	for _, x := range ks {
		if t.Is(x) {
			return true
		}
	}
	return false
}


type lexer struct {
	buf []byte
	cur uint
	pos uint
	w   uint
	out chan Token
}

func NewLexer(buf []byte) (*lexer, chan Token) {
	ret := &lexer{
		buf: buf,
		out: make(chan Token),
	}

	return ret, ret.out
}

func (l *lexer) Run() {
	l.run()
}

func (l *lexer) run() {
	for state := stateStart; state != nil; {
		state = state(l)
	}
	close(l.out)
}

var EOF rune = -1

func (l *lexer) next() rune {
	if l.pos == uint(len(l.buf)-1) {
		l.pos += 1
		return EOF
	}

	r, w := utf8.DecodeRune(l.buf[l.pos:])
	l.w = uint(w)
	l.pos += l.w
	return r
}

func (l *lexer) back() {
	l.pos -= l.w
}

func (l *lexer) peek() rune {
	defer l.back()
	return l.next()
}

func (l *lexer) accept(valid string) bool {
	r := l.next()
	if r == EOF {
		return false
	}

	if strings.IndexRune(valid, r) >= 0 {
		return true
	}
	l.back()
	return false
}

func (l *lexer) runOf(valid string) {
	for {
		if !l.accept(valid) {
			break
		}
	}
}

func (l *lexer) errf(format string, args ...interface{}) {
	l.out <- Token{
		Value: []byte(fmt.Sprintf(format, args...)),
		Code:  TokErr,
		Cur:   l.cur,
		Pos:   l.cur,
	}
}

func (l *lexer) spit(t tokenCode) {
	l.out <- Token{
		Code:  t,
		Cur:   l.cur,
		Pos:   l.pos,
		Value: l.buf[l.cur:l.pos],
	}

	l.cur = l.pos
}

type stateFn func(*lexer) stateFn

func stateStart(l *lexer) stateFn {
	return stateNext
}

func stateNum(l *lexer) stateFn {
	l.runOf(charsNum)
	l.spit(TokNum)
	return stateNext
}

func stateIdent(l *lexer) stateFn {
	l.runOf(charsIdent)
	l.spit(TokIdent)
	return stateNext
}

func stateVerb(l *lexer) stateFn {
	if l.accept(charsCaps) {
		l.runOf(charsCaps)
		l.spit(TokIdent)
		return stateNext
	}

	l.errf("Expected VERB")
	return nil
}

func stateNext(l *lexer) stateFn {
	if l.peek() == EOF {
		l.spit(TokEof)
		return nil
	}

	if l.accept(charsWs) {
		l.runOf(charsWs)
		l.spit(TokWs)
	}

	r := l.next()
	switch {
	case r == '\r':
		r = l.next()
		if r != '\n' {
			l.back()
			return stateNext
		}
		fallthrough
	case r == '\n':
		l.spit(TokNewline)
		return stateNext
	case r == '/':
		l.spit(TokSlash)
		return stateNext
	case r == '.':
		l.spit(TokDot)
		return stateNext
	case r == ':':
		l.spit(TokColon)
		return stateNext
	case r == '=':
		l.spit(TokEq)
		return stateNext
	case r == '&':
		l.spit(TokAnd)
		return stateNext
	case r == '?':
		l.spit(TokQuery)
		return stateNext
	case r == ';':
		l.spit(TokSemi)
		return stateNext
	case r == EOF:
		l.spit(TokEof)
		break
	case strings.ContainsRune(charsNum, r):
		l.back()
		return stateNum
	case strings.ContainsRune(charsAlpha, r):
		l.back()
		return stateIdent
	case strings.ContainsRune(charsPunct, r):
		l.spit(TokPunct)
		return stateNext
	}

	return nil
}
