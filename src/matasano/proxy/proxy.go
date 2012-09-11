package proxy

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"matasano/ca"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

type Handler interface {
	HandleRequest(raw *bytes.Buffer)
	HandleResponse(raw *bytes.Buffer)
}

type Proxy struct {
	addr       string
	listener   net.Listener
	ca         *ca.CA
	handler    Handler
	Completion chan bool
}

type Connection struct {
	buffer        *bytes.Buffer
	inbound       net.Conn
	outbound      net.Conn
	out_host      string
	chained_proxy string
	bp            *Proxy
}

type ConnectRequest struct {
	host string
	port int16
}

var (
	content_length = regexp.MustCompile("(?m)\nContent\\-[lL]ength:\\s+(\\d+)")
	connect_req    = regexp.MustCompile("^CONNECT\\s+(.*?):(\\d+)")
)

const (
	RX = iota
	TX
)

func (self *Proxy) NewConnection(c net.Conn) *Connection {
	return &Connection{
		buffer:   bytes.NewBuffer(make([]byte, 0, 1000)),
		inbound:  c,
		outbound: nil,
		bp:       self,
	}
}

type read_check func(buf *bytes.Buffer) bool

func http_read_completed(buf *bytes.Buffer) bool {
	skip := 4
	eoh := bytes.Index(buf.Bytes(), []byte("\r\n\r\n"))
	if eoh == -1 {
		skip = 2
		eoh = bytes.Index(buf.Bytes(), []byte("\n\n"))
		if eoh == -1 {
			return false
		}
	}

	m := content_length.FindSubmatch(buf.Bytes())
	if m == nil {
		return true
	} else {
		cl, err := strconv.ParseInt(string(m[1]), 0, 32)
		if err != nil || len(buf.Bytes()) >= int(cl)+skip+eoh {
			return true
		}
	}

	return false
}

func atomicio(conn net.Conn, tbuf *bytes.Buffer, dir int, check read_check) error {
	var buf [1024]byte
	for {
		var (
			l   int
			err error
		)

		if dir == TX {
			l, err = conn.Write(tbuf.Bytes())
		} else {
			l, err = conn.Read(buf[0:])
		}

		if err != nil {
			return err
		}

		if l == 0 {
			return nil
		}

		if dir == TX {
			tbuf.Next(l)
			if tbuf.Len() == 0 {
				break
			}
		} else {
			tbuf.Write(buf[0:l])

			if check != nil {
				if check(tbuf) {
					return nil
				}
			}
		}
	}

	return nil
}

func (self *Connection) Loop() {
	defer self.clean()
	for {
		var buf [1024]byte
		l, err := self.inbound.Read(buf[0:])
		if l < 1 && err != nil {
			log.Print(err)
			return
		}

		self.buffer.Write(buf[0:l])

		for {
			log.Printf("io: %v\n", self.buffer.String())
			r, c, err := self.check()
			if err != nil {
				self.e500("BAD REQUEST")
				log.Print(err)
				return
			}

			if c != nil {
				err := self.tls_connect(c)
				if err != nil {
					self.e500("BAD TLS CONNECT REQUEST")
					log.Print(err)
					return
				}
			}

			if r != nil {
				if r.URL.Host == "" && self.outbound == nil {
					self.e500("NO HOST IN URL")
					return
				}

				if self.out_host != r.URL.Host || self.outbound == nil {
					if strings.Index(r.URL.Host, ":") == -1 {
						r.URL.Host = net.JoinHostPort(self.out_host, "80")
					}

					if self.chained_proxy == "" {
						self.outbound, err = net.Dial("tcp", self.chained_proxy)
					} else {
						self.outbound, err = net.Dial("tcp", r.URL.Host)
					}

					if err != nil {
						self.e500(fmt.Sprintf("CANNOT CONNECT TO %s", r.URL.Host))
						return
					}

					if self.chained_proxy == "" {
						r.URL.Host = ""
					}
				}

				outb := bytes.NewBuffer(make([]byte, 0, 100))
				r.Write(outb)

				self.bp.handler.HandleRequest(outb)

				err := atomicio(self.outbound, outb, TX, nil)
				if err != nil {
					self.e500(fmt.Sprintf("Writing: %v", err))
					return
				}

				outb.Reset()

				err = atomicio(self.outbound, outb, RX, http_read_completed)
				if err != nil {
					self.e500(fmt.Sprintf("Reading: %v", err))
					return
				}

				self.bp.handler.HandleResponse(outb)

				err = atomicio(self.inbound, outb, TX, nil)
				if err != nil {
					return
				}
			}

			if c == nil && r == nil {
				break
			}
		}
	}
}

func (self *Connection) clean() {
	self.inbound.Close()
	if self.outbound != nil {
		self.outbound.Close()
	}
}

func (self *Connection) tls_connect(c *ConnectRequest) (err error) {
	err = nil

	var cert *tls.Certificate
	if cert, err = self.bp.ca.Lookup(c.host); err != nil {
		return
	}

	if self.chained_proxy == "" {
		var conn *tls.Conn
		conf := tls.Config{
			InsecureSkipVerify: true,
		}

		if conn, err = tls.Dial("tcp", fmt.Sprintf("%s:%d", c.host, c.port), &conf); err != nil {
			return
		}

		self.inbound.Write([]byte("HTTP/1.1 200 Connection established\r\n\r\n"))

		inconf := tls.Config{
			Certificates:       []tls.Certificate{*cert},
		}
		self.inbound = tls.Server(self.inbound, &inconf)
		self.out_host = net.JoinHostPort(c.host, strconv.Itoa(int(c.port)))
		self.outbound = conn
	} else {
		var conn net.Conn
		if conn, err = net.Dial("tcp", self.chained_proxy); err != nil {
			return
		}

		b := bytes.NewBuffer(make([]byte, 0, 100))
		b.Write([]byte(fmt.Sprintf("CONNECT %s:%d HTTP/1.0\r\nHost: %s\r\n\r\n", c.host, c.port, c.host)))

		if err = atomicio(conn, b, TX, nil); err != nil {
			return
		}

		b.Reset()

		if err = atomicio(conn, b, RX, http_read_completed); err != nil {
			return
		}

		fmt.Println(b.String())

		if err = atomicio(self.inbound, b, TX, nil); err != nil {
			return
		}

		conf := tls.Config{
			InsecureSkipVerify: true,
			Certificates:       []tls.Certificate{*cert},
		}

		self.inbound = tls.Server(self.inbound, &conf)
		self.out_host = net.JoinHostPort(c.host, strconv.Itoa(int(c.port)))
		self.outbound = conn
	}

	return
}

func (self *Connection) check() (*http.Request, *ConnectRequest, error) {
	skip := 4
	eoh := bytes.Index(self.buffer.Bytes(), []byte("\r\n\r\n"))
	if eoh == -1 {
		skip = 2
		eoh = bytes.Index(self.buffer.Bytes(), []byte("\n\n"))
	}
	if eoh == -1 {
		return nil, nil, nil
	}

	fmt.Println(self.buffer.String())

	thisreq := self.buffer.Bytes()[0 : eoh+skip]
	m := content_length.FindSubmatch(thisreq)
	if m != nil {
		conlen, err := strconv.ParseInt(string(m[1]), 0, 32)
		if err != nil {
			return nil, nil, errors.New("bad request")
		}

		if self.buffer.Len() >= (eoh + skip + int(conlen)) {
			thisreq = self.buffer.Bytes()[0 : eoh+skip+int(conlen)]
		} else {
			return nil, nil, nil
		}
	} else {
		m := connect_req.FindSubmatch(thisreq)
		if m != nil {
			port, err := strconv.ParseInt(string(m[2]), 0, 16)
			if err != nil {
				port = 443
			}
			return nil,
				&ConnectRequest{host: string(m[1]),
					port: int16(port)},
				nil
		}
	}

	reader := bufio.NewReader(bytes.NewReader(thisreq))
	req, err := http.ReadRequest(reader)
	self.buffer.Next(len(thisreq))
	return req, nil, err
}

func (self *Connection) e500(msg string) {
	self.inbound.Write([]byte(fmt.Sprintf("HTTP/1.0 500 %s\r\n\r\n", msg)))
}

func NewProxy(addr string, ca *ca.CA, handler Handler) (proxy *Proxy, err error) {
	proxy = &Proxy{
		addr:       addr,
		ca:         ca,
		handler:    handler,
		Completion: make(chan bool),
	}
	proxy.listener, err = net.Listen("tcp", proxy.addr)
	return
}

func (self *Proxy) Loop() {
	for {
		conn, err := self.listener.Accept()
		if err != nil {
			log.Print("Warn: ", err)
		}

		c := self.NewConnection(conn)
		go c.Loop()
	}
	self.Completion <- true
}

func Test() {
	fmt.Println("It worked")
}
