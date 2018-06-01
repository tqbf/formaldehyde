package my

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/fatih/color"
)

func UUID() string {
	return HumanToken(10)
}

func Token(size uint) []byte {
	if size < 10 {
		size = 10
	}

	ret := make([]byte, size)

	_, err := rand.Reader.Read(ret)
	if err != nil {
		panic("couldn't read urandom, erring on the side of panic")
	}

	return ret
}

func HumanToken(size uint) string {
	t := Token(size)
	return hex.EncodeToString(t)
}

type IntHeap []int

func (h IntHeap) Len() int           { return len(h) }
func (h IntHeap) Less(i, j int) bool { return h[i] < h[j] }
func (h IntHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *IntHeap) Push(x interface{}) {
	*h = append(*h, x.(int))
}
func (h *IntHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}
func (h *IntHeap) Top() int {
	old := *h
	n := len(old)
	x := old[n-1]
	return x
}

type UintHeap []uint

func (h UintHeap) Len() int           { return len(h) }
func (h UintHeap) Less(i, j int) bool { return h[i] < h[j] }
func (h UintHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *UintHeap) Push(x interface{}) {
	*h = append(*h, x.(uint))
}
func (h *UintHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}
func (h *UintHeap) Top() uint {
	old := *h
	n := len(old)
	x := old[n-1]
	return x
}

type Uint16Heap []uint16

func (h Uint16Heap) Len() int           { return len(h) }
func (h Uint16Heap) Less(i, j int) bool { return h[i] < h[j] }
func (h Uint16Heap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *Uint16Heap) Push(x interface{}) {
	*h = append(*h, x.(uint16))
}
func (h *Uint16Heap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}
func (h *Uint16Heap) Top() uint16 {
	old := *h
	n := len(old)
	x := old[n-1]
	return x
}

type Uint32Heap []uint32

func (h Uint32Heap) Len() int           { return len(h) }
func (h Uint32Heap) Less(i, j int) bool { return h[i] < h[j] }
func (h Uint32Heap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *Uint32Heap) Push(x interface{}) {
	*h = append(*h, x.(uint32))
}
func (h *Uint32Heap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}
func (h *Uint32Heap) Top() uint32 {
	old := *h
	n := len(old)
	x := old[n-1]
	return x
}

type Uint64Heap []uint64

func (h Uint64Heap) Len() int           { return len(h) }
func (h Uint64Heap) Less(i, j int) bool { return h[i] < h[j] }
func (h Uint64Heap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *Uint64Heap) Push(x interface{}) {
	*h = append(*h, x.(uint64))
}
func (h *Uint64Heap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}
func (h *Uint64Heap) Top() uint64 {
	old := *h
	n := len(old)
	x := old[n-1]
	return x
}

type TimeHeap []time.Time

func (h TimeHeap) Len() int           { return len(h) }
func (h TimeHeap) Less(i, j int) bool { return h[i].Before(h[j]) }
func (h TimeHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *TimeHeap) Push(x interface{}) {
	*h = append(*h, x.(time.Time))
}
func (h *TimeHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}
func (h *TimeHeap) Top() time.Time {
	old := *h
	n := len(old)
	x := old[n-1]
	return x
}

// Unexpected expects its error argument to be nil, and will noisily log
// if it isn't; either way, it returns its argument; use to log errors
// you don't expect to have happen without changing logic.
func Unexpected(err error) error {
	if err != nil {
		var buf [256]byte
		n := runtime.Stack(buf[:], false)
		log.Printf("unexpected error: %s\n%s", err, string(buf[0:n]))
	}

	return err
}

// OK returns true if its error argument is nil, and logs noisily if
// it isn't. Use the same with as Unexpected, but with marginally less
// typing effort.
func OK(err error) bool {
	if err != nil {
		var buf [256]byte
		n := runtime.Stack(buf[:], false)
		log.Printf("unexpected error: %s\n%s", err, string(buf[0:n]))
		return false
	}

	return true
}

func TestNoError(t *testing.T, err error) bool {
	if !OK(err) {
		t.Fatal(err)
		return false
	}
	return true
}

func TestAssert(t *testing.T, v bool) bool {
	if v == false {
		t.Fatalf("expected true")
		return false
	}
	return true
}

// Failsafe expects its error argument to be nil and will blow the
// program up if it isn't. Use for errors that can't happen ordinarily.
func Failsafe(err error) {
	if err != nil {
		panic(fmt.Sprintf("unexpected error: %s", err))
	}
}

// BUG(tqbf): without a panic or error exit here this is really
// "ShouldGetenv"
func MustGetenv(name string) string {
	ret := os.Getenv(name)
	if ret == "" {
		log.Printf("must provide '%s' environment variable", name)
	}
	return ret
}

type WrappedError struct {
	Parent, Child error
}

func (w *WrappedError) String() string {
	return fmt.Sprintf("%s: %s", w.Parent, w.Child)
}

func WrapError(p, c error) *WrappedError {
	return &WrappedError{
		Parent: p,
		Child:  c,
	}
}

var suppress = false
var suppressLock = &sync.Mutex{}

func SuppressDebug() {
	suppressLock.Lock()
	suppress = true
	suppressLock.Unlock()
}

func AllowDebug() {
	suppressLock.Lock()
	suppress = false
	suppressLock.Unlock()
}

func Debug(arg interface{}, args ...interface{}) {
	suppressLock.Lock()
	defer suppressLock.Unlock()

	if suppress == true {
		return
	}

	format, ok := arg.(string)
	if ok {
		color.New(color.FgYellow, color.Bold).Fprintf(os.Stderr, format+"\n", args...)
		return
	}

	err, ok := arg.(error)
	if ok {
		color.New(color.FgYellow, color.Bold).Fprintf(os.Stderr, err.Error()+"\n")
		return
	}

	c := color.New(color.FgYellow, color.Bold)
	c.Fprintf(os.Stderr, "%+v", format)
	for a := range args {
		c.Fprintf(os.Stderr, " %+v", a)
	}
	c.Fprintf(os.Stderr, "\n")

}
