package formaldehyd

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/latacora/scan"
)

const (
	tokWs scan.Code = iota + 1000
	tokNewline
	tokPhrase
	tokDashLine
	tokSquigLine
	tokPlusMinus
	tokSlider
	tokOButton
	tokCButton
	tokSwitchOn
	tokSwitchOff
)

func ToString(buf []byte, t scan.Token) string {
	val := scan.TokenText(buf, []scan.Token{t})

	switch t.Code {
	case tokWs:
		return fmt.Sprintf("Ws: <%s>", val)
	case tokNewline:
		return fmt.Sprintf("Newline: <%s>", val)
	case tokPhrase:
		return fmt.Sprintf("Phrase: <%s>", val)
	case tokDashLine:
		return fmt.Sprintf("DashLine: <%s>", val)
	case tokSquigLine:
		return fmt.Sprintf("SquigLine: <%s>", val)
	case tokPlusMinus:
		return fmt.Sprintf("PlusMinus: <%s>", val)
	case tokSlider:
		return fmt.Sprintf("Slider: <%s>", val)
	case tokOButton:
		return fmt.Sprintf("OButton: <%s>", val)
	case tokCButton:
		return fmt.Sprintf("CButton: <%s>", val)
	case tokSwitchOn:
		return fmt.Sprintf("SwitchOn: <%s>", val)
	case tokSwitchOff:
		return fmt.Sprintf("SwitchOff: <%s>", val)
	case scan.TokEOF:
		return fmt.Sprintf("EOF: <%s>", val)
	default:
		return fmt.Sprintf("unknown: <%s>", val)
	}
}

func dbg(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}

var (
	startAlnum = strings.Join([]string{scan.CharsIdent, ""}, "")
	innerAlnum = strings.Join([]string{scan.CharsIdent, ".!?,;\"*'(@+-`#)&:<>/\\{}[]"}, "")
)

func scanPhrase(s *scan.Scanner) {
	for {
		s.Accept(startAlnum)
		s.AcceptRun(innerAlnum)
		if s.Peek(scan.CharsHzWs) {
			s.Next()
			if !s.Peek(startAlnum) {
				s.Back()
				break
			}
		} else {
			break
		}
	}

	s.Emit(tokPhrase)
}

func tokenize(buf []byte) (ret []scan.Token) {
	tokens := []scan.Token{}
	s := scan.New(buf, func(t scan.Token) { tokens = append(tokens, t) })

	for !s.IsEOF() {
		switch {
		case s.Peek("\r"):
			s.Discard("\r")

		case s.Peek("\n"):
			s.AcceptAndEmit("\n", tokNewline)

		case s.Peek(scan.CharsHzWs):
			s.EmitRun(scan.CharsHzWs, tokWs)

		case s.AcceptExact("+/-"):
			s.Emit(tokPlusMinus)

		case s.AcceptExact("-o-"):
			s.Emit(tokSlider)

		case s.AcceptExact("<-"):
			s.Emit(tokPhrase)

		case s.AcceptExact("->"):
			s.Emit(tokPhrase)

		case s.AcceptExact("[("):
			s.Emit(tokOButton)

		case s.AcceptExact(")]"):
			s.Emit(tokCButton)

		case s.AcceptExact("*_"):
			s.Emit(tokSwitchOn)

		case s.AcceptExact("_*"):
			s.Emit(tokSwitchOff)

		case s.Peek(startAlnum):
			scanPhrase(s)

		case s.Peek("("):
			s.AcceptAndEmit("(", scan.Code('('))

		case s.Peek(")"):
			s.AcceptAndEmit(")", scan.Code(')'))

		case s.Peek("["):
			s.AcceptAndEmit("[", scan.Code('['))

		case s.Peek("]"):
			s.AcceptAndEmit("]", scan.Code(']'))

		case s.Peek("*"):
			s.AcceptAndEmit("*", scan.Code('*'))

		case s.Peek("#"):
			s.AcceptAndEmit("#", scan.Code('#'))

		case s.Peek("|"):
			s.AcceptAndEmit("|", scan.Code('|'))

		case s.Peek("-"):
			s.EmitRun("-", tokDashLine)

		case s.Peek("~"):
			s.EmitRun("~", tokSquigLine)

		default:
			dbg("eat: <%s>", s.CurrentString())
			s.Next()
		}
	}

	return tokens
}

const (
	NDocument = iota
	NLabel
	NHeading
	NText
	NField
	NPage
	NTextField
	NRadioField
	NCheckField
	NDropField
	NSelection
	NButton
	NNumberField
	NSwitchField
)

var nodeNames = []string{
	"Document",
	"Label",
	"Heading",
	"Text",
	"Field",
	"Page",
	"TextField",
	"RadioField",
	"CheckField",
	"DropField",
	"Selection",
	"Button",
	"NumberField",
	"SwitchField",
}

type Node struct {
	Kind     int
	Text     string
	Parent   *Node
	Children []*Node
	Line     int
	Hash     string
	Opt      string
	Attrs    map[string]string
}

type parser struct {
	buf         []byte
	tokens      []scan.Token
	off         int
	optTag      string
	state       int
	accum       []scan.Token
	current     *Node
	twidth      int
	line        int
	err         error
	currentHash string
}

func (p *parser) next() *scan.Token {
	if (p.off + 1) >= len(p.tokens) {
		return nil
	}

	p.off++

	if p.tokens[p.off].Code == tokNewline {
		p.line++
	}

	return &p.tokens[p.off]
}

func (p *parser) neednext() *scan.Token {
	t := p.next()
	if t == nil {
		p.err = fmt.Errorf("unexpected end of input at")
		return &scan.Token{Code: scan.TokEOF}
	}

	return t
}

func (p *parser) addChild(kind int, tox []scan.Token) *Node {
	new := &Node{
		Kind:   kind,
		Text:   cleansingFire(scan.TokenText(p.buf, tox)),
		Parent: p.current,
		Line:   p.line,
		Hash:   p.currentHash,
		Opt:    p.optTag,
		Attrs:  map[string]string{},
	}

	p.currentHash = ""

	p.current.Children = append(p.current.Children, new)
	return new
}

func (p *parser) at(off int) scan.Code {
	if off >= len(p.tokens) {
		return scan.TokEOF
	}

	return p.tokens[off].Code
}

func (p *parser) unexpected(t *scan.Token, context, message string) {
	val := scan.TokenText(p.buf, []scan.Token{*t})

	p.err = fmt.Errorf(`at line %d:
while %s,
got "%s"
but expected %s`, p.line, context, val, message)
}

func (p *parser) addAccum(t *scan.Token) {
	p.accum = append(p.accum, *t)
}

func (p *parser) resetAccum() {
	p.accum = []scan.Token{}
}

func (p *parser) textField() {
	for p.err == nil {
		t := p.neednext()

		if t.Code == scan.Code(']') {
			l := 1
			for _, nt := range p.accum {
				if nt.Code == tokNewline {
					l += 1
				}
			}

			p.current.Attrs["width"] = strconv.Itoa(p.twidth)
			p.current.Attrs["height"] = strconv.Itoa(l)
			p.current.Attrs["default"] = cleansingFire(scan.TokenText(p.buf, p.accum))
			p.twidth = 0
			p.current = p.current.Parent
			return
		}

		if t.Code == tokPlusMinus {
			p.current.Kind = NNumberField
			p.current.Attrs["plusminus"] = "t"
		} else if t.Code == tokSlider {
			p.current.Kind = NNumberField
			p.current.Attrs["slider"] = "t"
		} else {
			p.twidth += scan.TokenSpan([]scan.Token{*t})
			p.addAccum(t)
		}
	}
}

func (p *parser) checkOrText() {
	t := p.neednext()
	w := scan.TokenSpan([]scan.Token{*t})

	switch {
	case t.Code == scan.Code('*') && w == 1 && p.at(p.off+1) == scan.Code(']'):
		p.current.Attrs["checked"] = "t"
		fallthrough

	case t.Code == tokWs && w == 1 && p.at(p.off+1) == scan.Code(']'):
		p.current.Kind = NCheckField
		t = p.neednext()
		if t.Code == scan.Code(']') {
			p.current = p.current.Parent
			return
		} else {
			p.unexpected(t, "parsing a checkbox", "a ] to close the checkbox")
		}

	default:
		p.current.Kind = NTextField
		p.twidth = 0

		if t.Code == scan.Code(']') {
			p.current = p.current.Parent
			return
		}

		p.addAccum(t)
		p.twidth += w
		p.textField()
	}
}

func (p *parser) dropNextField() {
	for p.err == nil {
		t := p.neednext()

		switch t.Code {
		case tokWs, tokPhrase:
			p.addAccum(t)

		case tokNewline:
			li := p.off
			c := scan.Code(0)

			// look ahead and see if there's more text
			// on the continuation line
			for {
				c = p.at(li)
				li++
				if c != tokWs && c != tokNewline {
					break
				}
			}

			if c == tokPhrase {
				p.addAccum(t)
			} else {
				p.addChild(NSelection, p.accum)
				p.resetAccum()
				return
			}
		default:

		}
	}
}

func (p *parser) dropField() {
	t := p.neednext()
	if t.Code != tokDashLine {
		p.unexpected(t, "parsing a dropdown selector", "a dashed line")
		return
	}

	for p.err == nil {
		t = p.neednext()

		switch t.Code {
		case tokWs, tokNewline:
		case scan.Code('*'):
			p.dropNextField()

		case tokDashLine:
			p.current = p.current.Parent
			return

		default:
			p.unexpected(t, "parsing the selections of a dropdown selector",
				"whitespace, a star marking the next selector, or a dashed line ending the dropdown")
		}
	}
}

func (p *parser) radio() {
	for p.err == nil {
		t := p.neednext()

		if t.Code == scan.Code('*') {
			p.current.Attrs["selected"] = "t"
		} else if t.Code == tokSwitchOn {
			p.current.Kind = NSwitchField
			p.current.Attrs["on"] = "t"
		} else if t.Code == tokSwitchOff {
			p.current.Kind = NSwitchField
		} else if t.Code == scan.Code(')') {
			p.current = p.current.Parent
			return
		} else if t.Code != tokWs {
			p.unexpected(t, "parsing a radio button",
				"an star checked marker, a switch *_ indicator, whitespace, or the close of the radio button")
		}
	}
}

func cleansingFire(input string) string {
	return strings.Join(strings.Fields(strings.Replace(input, "|", "", -1)), " ")
}

func (p *parser) field(t *scan.Token) {
	for i := len(p.accum) - 1; i >= 0; i-- {
		if p.accum[i].Code == tokNewline {
			p.addChild(NText, p.accum[0:i])
			p.accum = p.accum[i:]
			break
		}
	}

	p.current = p.addChild(NField, nil)
	p.current.Attrs["label"] = cleansingFire(scan.TokenText(p.buf, p.accum))
	p.resetAccum()

	switch t.Code {
	case scan.Code('['):
		p.checkOrText()

	case scan.Code('('):
		p.current.Kind = NRadioField
		p.radio()

	default:
		p.current.Kind = NDropField
		p.dropField()
	}

}

func (p *parser) page() {
	if p.current.Kind != NDocument && p.current.Kind != NPage {
		p.err = fmt.Errorf("can't nest pages")
		return
	}

	if p.current.Kind == NPage {
		p.current = p.current.Parent
	}

	p.current = p.addChild(NPage, nil)
	p.current.Attrs["label"] = cleansingFire(scan.TokenText(p.buf, p.accum))
	p.resetAccum()
}

func (p *parser) text(t *scan.Token) {
	var prev *scan.Token

	for t != nil && p.err == nil {
		switch t.Code {
		case tokWs, tokNewline, tokPhrase:
			p.addAccum(t)

		case scan.Code('#'):
			if prev != nil && prev.Code == tokNewline {
				p.addChild(NText, p.accum)
				p.resetAccum()
				p.hashtagOrHeader()
				return
			}

			p.addAccum(t)

		case tokDashLine:
			p.page()
			return

		case scan.Code('['), scan.Code('('), scan.Code('*'):
			p.field(t)
			return

		default:
			p.unexpected(t, "parsing a run of text",
				"a page marker, a hash tag, the start of a field, or a drop-down")
			return
		}

		prev = t
		t = p.next()
	}
}

func (p *parser) hashtagOrHeader() {
	t := p.neednext()
	switch t.Code {
	case tokPhrase:
		p.currentHash = scan.TokenText(p.buf, []scan.Token{*t})

	case tokWs:
		for t.Code != tokNewline && p.err == nil {
			p.addAccum(t)
			t = p.neednext()
		}

		if p.err != nil {
			return
		}

		p.addChild(NHeading, p.accum)
		p.resetAccum()

	default:
		p.unexpected(t, "parsing a hashtag", "the alphanumeric characters of a hash tag")
	}
}

func (p *parser) opt() {
	for p.err == nil {
		t := p.neednext()

		if t.Code != tokNewline && t.Code != tokSquigLine {
			p.addAccum(t)
		} else {
			p.optTag = cleansingFire(scan.TokenText(p.buf, p.accum))
			return
		}
	}
}

func (p *parser) button() {
	for p.err == nil {
		t := p.neednext()
		switch {
		case t.Code == tokCButton && len(p.accum) == 0:
			p.err = fmt.Errorf("can't have button without label")
		case t.Code == tokNewline:
			p.err = fmt.Errorf("buttons fit on one line please")
		case t.Code == tokCButton:
			p.addChild(NButton, p.accum)
			p.resetAccum()
			return
		default:
			p.addAccum(t)
		}
	}
}

func (p *parser) document() {
	t := p.next()

	for t != nil && p.err == nil {
		switch t.Code {
		case tokWs, tokNewline:
		case tokPhrase:
			p.text(t)

		case tokOButton:
			p.button()

		case tokSquigLine:
			p.opt()

		case scan.Code('#'):
			p.hashtagOrHeader()

		default:
			p.unexpected(t, "parsing the document", "whitespace, text, a button, or a hash tag")
		}

		t = p.next()
	}
}

func Parse(buf []byte) (node *Node, err error) {
	p := &parser{
		buf:         buf,
		tokens:      tokenize(buf),
		optTag:      "",
		current:     &Node{Kind: NDocument},
		twidth:      0,
		line:        1,
		off:         -1,
		currentHash: "",
	}

	p.document()

	for p.current.Parent != nil {
		p.current = p.current.Parent
	}

	return p.current, p.err
}

func (n *Node) stringRec(w io.Writer, depth int) {
	for i := 0; i < depth; i++ {
		w.Write([]byte("  "))
	}

	w.Write([]byte(nodeNames[n.Kind]))

	if n.Text != "" {
		w.Write([]byte(" " + n.Text + " "))
	}

	if len(n.Attrs) > 0 {
		w.Write([]byte(fmt.Sprintf(" %s", n.Attrs)))
	}

	w.Write([]byte{'\n'})

	for _, kid := range n.Children {
		kid.stringRec(w, depth+1)
	}
}

func (n *Node) String() string {
	w := &bytes.Buffer{}
	n.stringRec(w, 0)
	return w.String()
}
