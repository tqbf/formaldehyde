package formaldehyd

import (
	"fmt"
	"io/ioutil"
	"testing"
)

func fixture(name string) []byte {
	buf, err := ioutil.ReadFile(fmt.Sprintf("fixtures/%s", name))
	if err != nil {
		panic(err)
	}

	return buf
}

func ok(t *testing.T, err error) bool {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
		return false
	}
	return true
}

func TestScrap(t *testing.T) {
	buf := fixture("form.1")

	tokens := tokenize(buf)
	_ = tokens

	// for _, t := range tokens {
	//  	dbg("%s", ToString(buf, t))
	// }

	n, err := Parse(buf)
	ok(t, err)

	dbg("%s", n.String())

	dbg("%s", n.JSON())
}
