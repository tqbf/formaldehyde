package my

import (
	"net/url"
	"os"
	"os/signal"
)

func PanicOnSignal(s os.Signal) {
	c := make(chan os.Signal)

	go func() {
		_ = <-c
		panic("signal")
	}()

	signal.Notify(c, s)
}

// FormargsFromMap returns a url.Values from a simpler map, for
// one-liners when you know you don't have multiple instances of
// a key.
func FormargsFromMap(m map[string]string) url.Values {
	f := url.Values{}
	for k, v := range m {
		f.Set(k, v)
	}
	return f
}

// Formargs returns a url.Values from a sequence of key, value,
// key, value, for simple one-liners.
func Formargs(kvs ...string) url.Values {
	f := url.Values{}
	var k string

	for i, v := range kvs {
		if i%2 == 0 {
			k = v
		} else {
			f.Set(k, v)
		}
	}

	return f
}
