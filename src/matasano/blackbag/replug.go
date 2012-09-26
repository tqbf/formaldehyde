package blackbag

import (
	"io/ioutil"
	"fmt"
	"net"
	"io"
	"log"
	"os"
	"strconv"
	"sync"
)

type Replug struct {
	Host	string
	Port	int
	Lport	int
	
	srv	net.Listener

	Wait	chan bool
}

func NewReplug(host string, port, lport int) (Replug, error) {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", lport))
	if err != nil {
		return Replug{}, err
	}

	return Replug{
		Host: host,
		Port: port,
		Lport: lport,
		srv: l,
		Wait: make(chan bool),
	}, nil
}

func teeCopy(r io.Reader, w io.Writer, tee func([]byte)) (tl int, e error) {
	buf := make([]byte, 1024)
	tl = 0

	for {
		var rl int
		rl, e = r.Read(buf)
		switch {
		case e != nil:
			break
		case rl > 0:
			tl += rl
			_, e = w.Write(buf[0:rl])
			switch {
			case e != nil:
				break
			default:
				tee(buf[0:rl])			
			}
		}
	}

	return
}

func (self *Replug) Loop() {
	for { 
		incoming, err := self.srv.Accept()
		if err != nil { 
			log.Printf("[!!!incoming: %v]\n", err)
		}

		go func() { 
			log.Printf("[incoming: %v]\n", incoming.RemoteAddr())

			canlog := sync.Mutex{}

			outgoing, err := net.Dial("tcp", fmt.Sprintf("%s:%d", self.Host, self.Port))		
			if err == nil {
				log.Printf("[outgoing: %v]\n", outgoing.RemoteAddr())

				go teeCopy(incoming, outgoing, func(buf []byte) { 
					canlog.Lock()
					defer canlog.Unlock()
				
					log.Printf("[%v -> %v]\n", incoming.RemoteAddr(), outgoing.RemoteAddr())
					Hexdump(os.Stderr, buf)
				})

				go teeCopy(outgoing, incoming, func(buf []byte) { 
					canlog.Lock()
					defer canlog.Unlock()

					log.Printf("[%v <- %v]\n", incoming.RemoteAddr(), outgoing.RemoteAddr())
					Hexdump(os.Stderr, buf)
				})
			} else {
				log.Printf("Outgoing: %v %v", outgoing, err)
			}
		}()
	}
}

func Hexdump(f io.Writer, buf []byte) {
	var (
		l int = len(buf)
		off int = 0
	)

	for off < l {
		fmt.Fprintf(f, "%08x  ", off)

		for i := 0; i < 8; i++ {
			if off + i >= l {
				fmt.Fprintf(f, "-- ")
			} else {
				fmt.Fprintf(f, "%02x ", buf[off + i])
			}
		}

		fmt.Fprintf(f, " ")

		for i := 0; i < 8; i++ {
			if off + i + 8>= l {
				fmt.Fprintf(f, "-- ")
			} else {
				fmt.Fprintf(f, "%02x ", buf[off + i + 8])
			}
		}

		fmt.Fprintf(f, " |")
	
		for i := 0; i < 16; i++ {
			if off + i >= l {				
				fmt.Fprintf(f, " ")
			} else {
				if strconv.IsPrint(rune(buf[off + i])) {
					fmt.Fprintf(f, "%c", buf[off + i])
				} else {
					fmt.Fprintf(f, ".")
				}
			}
		}

		off += 16
		fmt.Fprintf(f, "|\n")		
	}
}

func Test() { 
	f, _ := os.Open("/etc/ttys")
	buf, _ := ioutil.ReadAll(f)

	Hexdump(os.Stdout, buf)

}
