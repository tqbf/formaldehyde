package formaldehyd

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// type Node struct {
//  	Kind     int
//  	Text     string
//  	Parent   *Node
//  	Children []*Node
//  	Line     int
//  	Hash     string
//  	Opt      string
//  	Attrs    map[string]string
// }

type JButton struct {
	Kind  string `json:"type"`
	Label string `json:"label"`
	Line  int    `json:"line"`
	Tag   string `json:"tag"`
	Opt   string `json:"opt"`
}

type JDropField struct {
	Kind    string   `json:"type"`
	Label   string   `json:"label"`
	Options []string `json:"options"`
	Line    int      `json:"line"`
	Tag     string   `json:"tag"`
	Opt     string   `json:"opt"`
}

type JHeader struct {
	Kind string `json:"type"`
	Text string `json:"text"`
	Tag  string `json:"tag"`
	Opt  string `json:"opt"`
}

type JText struct {
	Kind string `json:"type"`
	Text string `json:"text"`
	Tag  string `json:"tag"`
	Opt  string `json:"opt"`
}

type JRadioField struct {
	Kind     string `json:"type"`
	Label    string `json:"label"`
	Selected bool   `json:"selected"`
	Line     int    `json:"line"`
	Tag      string `json:"tag"`
	Opt      string `json:"opt"`
}

type JCheckField struct {
	Kind    string `json:"type"`
	Label   string `json:"label"`
	Checked bool   `json:"checked"`
	Line    int    `json:"line"`
	Tag     string `json:"tag"`
	Opt     string `json:"opt"`
}

type JTextField struct {
	Kind    string `json:"type"`
	Label   string `json:"label"`
	Default string `json:"default"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
	Line    int    `json:"line"`
	Tag     string `json:"tag"`
	Opt     string `json:"opt"`
}

type JNumberField struct {
	Kind      string `json:"type"`
	Label     string `json:"label"`
	Default   int    `json:"default"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Slider    bool   `json:"slider"`
	PlusMinus bool   `json:"plusminus"`
	Line      int    `json:"line"`
	Tag       string `json:"tag"`
	Opt       string `json:"opt"`
}

type JPage struct {
	Kind     string        `json:"type"`
	Label    string        `json:"label"`
	Children []interface{} `json:"children"`
}

type JDocument struct {
	Children []interface{} `json:"children"`
}

func (n *Node) JSON() string {
	d := &JDocument{}

	rti := func(s string) int { ret, _ := strconv.Atoi(s); return ret }

	handle := func(k *Node) interface{} {
		switch k.Kind {
		case NButton:
			return &JButton{
				Kind:  "button",
				Label: k.Text,
				Tag:   k.Hash,
				Opt:   k.Opt,
				Line:  k.Line,
			}

		case NText:
			return &JText{
				Kind: "text",
				Text: k.Text,
				Tag:  k.Hash,
				Opt:  k.Opt,
			}

		case NHeading:
			return &JHeader{
				Kind: "heading",
				Text: k.Text,
				Tag:  k.Hash,
				Opt:  k.Opt,
			}

		case NNumberField:
			return &JNumberField{
				Kind:      "numberfield",
				Label:     k.Attrs["label"],
				Default:   rti(k.Attrs["default"]),
				Slider:    k.Attrs["slider"] == "t",
				PlusMinus: k.Attrs["plusminus"] == "t",
				Width:     rti(k.Attrs["width"]),
				Height:    rti(k.Attrs["height"]),
				Line:      k.Line,
				Tag:       k.Hash,
				Opt:       k.Opt,
			}

		case NTextField:
			return &JTextField{
				Kind:    "textfield",
				Label:   k.Attrs["label"],
				Default: k.Attrs["default"],
				Width:   rti(k.Attrs["width"]),
				Height:  rti(k.Attrs["height"]),
				Line:    k.Line,
				Tag:     k.Hash,
				Opt:     k.Opt,
			}

		case NCheckField:
			var checked bool
			if k.Attrs["checked"] == "t" {
				checked = true
			}
			return &JCheckField{
				Kind:    "check",
				Label:   k.Attrs["label"],
				Checked: checked,
				Line:    k.Line,
				Tag:     k.Hash,
				Opt:     k.Opt,
			}

		case NRadioField:
			var checked bool
			if k.Attrs["selected"] == "t" {
				checked = true
			}
			return &JRadioField{
				Kind:     "radio",
				Label:    k.Attrs["label"],
				Selected: checked,
				Line:     k.Line,
				Tag:      k.Hash,
				Opt:      k.Opt,
			}

		case NDropField:
			drop := &JDropField{
				Kind:  "select",
				Label: k.Attrs["label"],
				Line:  k.Line,
				Tag:   k.Hash,
				Opt:   k.Opt,
			}

			for _, dcur := range k.Children {
				drop.Options = append(drop.Options, dcur.Text)
			}

			return drop
		}

		panic(fmt.Sprintf("notreached: %d", k.Kind))
		return nil
	}

	for _, cur := range n.Children {
		switch cur.Kind {
		case NPage:
			p := &JPage{
				Kind:  "page",
				Label: cur.Attrs["label"],
			}

			for _, pcur := range cur.Children {
				p.Children = append(p.Children, handle(pcur))
			}

			d.Children = append(d.Children, p)

		default:
			d.Children = append(d.Children, handle(cur))
		}
	}

	buf, _ := json.MarshalIndent(d, "", "  ")
	return string(buf)
}
