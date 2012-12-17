package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/bmizerany/pat"
	"html/template"
	"io/ioutil"
	"net/http"
	"newmadrid/msp43x"
	"strconv"
)

type sessionHandler func(w http.ResponseWriter, r *http.Request, s *Sessionkv)
type cpuHandler func(w http.ResponseWriter, r *http.Request, s *Sessionkv, c *UserCpu)

func errorResponse(w http.ResponseWriter, reason string) {
	out, _ := json.Marshal(&map[string]interface{}{
		"data": &map[string]interface{}{
			"success": false, 
			"reason": reason,
		},
	})
	fmt.Fprintf(w, "%s", string(out))
}

func mustNotBeSleeping(handler cpuHandler) http.HandlerFunc {
	return mustSession(func(w http.ResponseWriter, r *http.Request, s *Sessionkv) {
		cpu := GetCpu(s.Map()["name"])

		// log.Printf(">> %v -> %p\n", r.URL, cpu)

		if(cpu != nil && cpu.State != CpuIoSleep) {
			handler(w, r, s, cpu)
		} else {
			errorResponse(w, "not ready for this operation")
		}
	})	
}

func mustCpu(handler cpuHandler) http.HandlerFunc {
	return mustSession(func(w http.ResponseWriter, r *http.Request, s *Sessionkv) {
		cpu := GetCpu(s.Map()["name"])

		// log.Printf(">> %v -> %p\n", r.URL, cpu)

		handler(w, r, s, cpu)
	})
}

func mustSession(handler sessionHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var ses *Sessionkv

		for _, c := range r.Cookies() {
			if c.Name == "S" {
				ses = RestoreSession([]byte(c.Value))
				break
			}
		}

		if ses == nil {
			if name := r.URL.Query().Get(":name"); name != "" {
				ses = NewSession()
				ses.Map()["name"] = name
				handler(w, r, ses)
			} else {
				http.Redirect(w, r, "/login", 302)
			}
		} else {
			h := ses.Hash()
			handler(w, r, ses)
			if bytes.Compare(h, ses.Hash()) != 0 {
				http.SetCookie(w, &http.Cookie{
					Name:  "S",
					Value: string(ses.Encode()),
				})
			}
		}
	}
}

func CpuInterface(templates string, redis *RedisLand) (m *pat.PatternServeMux) {
	m = pat.New()

	render := func(w http.ResponseWriter, name string, data interface{}) {
		t := template.Must(template.ParseFiles(fmt.Sprintf("%s/%s", templates, name)))
		t.Execute(w, data)
	}

	m.Get("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/cpu", 302)
	}))

	m.Get("/login", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		render(w, "login.html", nil)
	}))

	m.Post("/login", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s := NewSession()
		s.Map()["name"] = r.FormValue("name")

		http.SetCookie(w, &http.Cookie{
			Name:  "S",
			Value: string(s.Encode()),
		})

		http.Redirect(w, r, "/cpu", 302)
	}))

	m.Post("/logout", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:   "S",
			MaxAge: -1,
			Value:  "",
		})

		http.Redirect(w, r, "/login", 302)
	}))

	m.Get("/cpu/:name", mustSession(func(w http.ResponseWriter, r *http.Request, s *Sessionkv) {
		type response struct {
			Name string
			Cpu  interface{}
		}

		cpu := GetCpu(s.Map()["name"])

		render(w, "cpu.html", response{
			Name: s.Map()["name"],
			Cpu:  cpu.MCU,
		})
	}))

	type RegsData struct {
		Regs [16]int `json:"regs"`
	}

	type RegsTop struct {
		Data RegsData `json:"data"`
	}

	type Result struct {
		Data interface{} `json:"data"`
	}

	m.Get("/cpu/:name/regs_form", mustCpu(func(w http.ResponseWriter, r *http.Request, s *Sessionkv, c *UserCpu) {
		render(w, "regs.html", struct {
			Regs [16]uint16
		}{c.MCU.GetRegs()})
	}))

	m.Get("/cpu/:name/regs", mustCpu(func(w http.ResponseWriter, r *http.Request, s *Sessionkv, c *UserCpu) {
		var regsout [16]int

		for i, v := range c.MCU.GetRegs() {
			regsout[i] = int(v)
		}

		out, _ := json.Marshal(RegsTop{
			Data: RegsData{
				Regs: regsout,
			},
		})

		fmt.Fprintf(w, "%s", string(out))
	}))

	type requestResult struct {
		success bool   `json:"success"`
		data    string `json:"data"`
	}

	m.Get("/cpu/:name/state", mustCpu(func(w http.ResponseWriter, r *http.Request, s *Sessionkv, c *UserCpu) {
		complete := make(chan int)

		c.Comm <- func(c *UserCpu) {
			complete <- c.State
		}

		res := <-complete

		out, _ := json.Marshal(&map[string]interface{}{
			"data": &map[string]interface{}{
				"state": fmt.Sprintf("%d", res),
			},
		})
		fmt.Fprintf(w, "%s", string(out))
	}))

	m.Post("/cpu/:name/regs", mustCpu(func(w http.ResponseWriter, r *http.Request, s *Sessionkv, c *UserCpu) {
		result := &requestResult{
			success: true,
			data:    "",
		}
		parsed := &RegsTop{}
		body, err := ioutil.ReadAll(r.Body)
		if err == nil {
			err = json.Unmarshal(body, &parsed)
			if err == nil {
				complete := make(chan bool)
				c.Comm <- func(c *UserCpu) {
					regs := c.MCU.GetRegs()

					for i := 0; i < 16; i++ {
						if parsed.Data.Regs[i] >= 0 {
							regs[i] = uint16(parsed.Data.Regs[i])
						}
					}

					c.MCU.SetRegs(regs)
					complete <- true
				}
				result.success = <-complete
			}
		}

		out, _ := json.Marshal(&map[string]interface{}{
			"data": &map[string]interface{}{
				"success": true,
				"data":    "",
			},
		})
		fmt.Fprintf(w, "%s", string(out))
	}))

	m.Post("/cpu/:name/load", mustCpu(func(w http.ResponseWriter, r *http.Request, s *Sessionkv, c *UserCpu) {
		type loadRequest struct {
			Key string `json:"key"`
		}
		type wrapper struct {
			Data loadRequest `json:"data"`
		}

		parsed := &wrapper{}
		result := &requestResult{
			success: false,
			data:    "",
		}

		body, err := ioutil.ReadAll(r.Body)
		if err == nil {
			err = json.Unmarshal(body, &parsed)
			if err == nil {
				err = c.LoadHexFromRedis(parsed.Data.Key)
			}
		}

		if err != nil {
			result.data = fmt.Sprintf("%v", err)
		} else {
			c.Image = parsed.Data.Key
			result.success = true
		}

		out, _ := json.Marshal(&map[string]interface{}{
			"data": &map[string]interface{}{
				"success": result.success,
			},
		})
		fmt.Fprintf(w, "%s", string(out))
	}))

	m.Get("/cpu/:name/memory/:address", mustCpu(func(w http.ResponseWriter, r *http.Request, s *Sessionkv, c *UserCpu) {
		var (
			addrs, lens string
			addr, len   int64
			raw         []byte
			err         error
		)

		addrs = r.URL.Query().Get(":address")
		if addrs != "" {
			addr, err = strconv.ParseInt(addrs, 16, 32)
			if err == nil {
				lens = r.URL.Query().Get("len")
				if lens == "" {
					lens = "32"
				}

				len, err = strconv.ParseInt(lens, 0, 32)
				if err == nil {
					complete := make(chan bool)
					c.Comm <- func(c *UserCpu) {
						raw, err = c.Mem.Read(uint16(addr), uint16(len))
						complete <- true
					}
					<-complete
				}
			}
		}

		out, _ := json.Marshal(&map[string]interface{}{
			"error": err,
			"raw":   raw,
		})
		fmt.Fprintf(w, "%s", string(out))

	}))

	m.Get("/cpu/:name/memory/:address/insns", mustCpu(func(w http.ResponseWriter, r *http.Request, s *Sessionkv, c *UserCpu) {
		var (
			addrs, cs string
			addr, cnt int64
			err       error
		)

		insns := make([]string, 0, 10)

		addrs = r.URL.Query().Get(":address")
		if addrs != "" {
			addr, err = strconv.ParseInt(addrs, 16, 32)
			if err == nil {
				cs = r.URL.Query().Get("count")
				if cs == "" {
					cs = "10"
				}

				cnt, err = strconv.ParseInt(cs, 0, 32)
				if err == nil {
					if cnt > 100 || cnt < 0 {
						cnt = 100
					}

					complete := make(chan bool)
					c.Comm <- func(c *UserCpu) {
						for i := 0; i < int(cnt); i++ {
							bytes, err := c.Mem.Load6Bytes(uint16(addr))
							if err != nil {
								break
							}

							i, err := msp43x.Disassemble(bytes)
							if err != nil {
								break
							}

							insns = append(insns, i.String())

							addr += int64(i.Width)
						}

						complete <- true
					}
					<-complete
				}
			}
		}

		out, _ := json.Marshal(&map[string]interface{}{
			"data": &map[string]interface{}{
				"insns": insns,
			},
		})
		fmt.Fprintf(w, "%s", string(out))
	}))

	m.Get("/cpu/:name/events", mustCpu(func(w http.ResponseWriter, r *http.Request, s *Sessionkv, c *UserCpu) {
		type results struct {
			success bool
			events  map[string]int
		}

		result := &results{events: make(map[string]int)}
		complete := make(chan bool)

		c.Comm <- func(c *UserCpu) {
			for key, value := range c.Breakpoints {
				result.events[fmt.Sprintf("%x", key)] = value
			}
			complete <- true
		}

		result.success = <-complete

		out, _ := json.Marshal(&map[string]interface{}{
			"data": &map[string]interface{}{
				"success": result.success,
				"events":  result.events,
			},
		})
		fmt.Fprintf(w, "%s", string(out))

	}))


	m.Post("/cpu/:name/event", mustCpu(func(w http.ResponseWriter, r *http.Request, s *Sessionkv, c *UserCpu) {
		type cpuEvent struct {
			Addr  string `json:"addr"`
			Event int    `json:"event"`
		}

		type wrapper struct {
			Data cpuEvent `json:"data"`
		}

		parsed := &wrapper{}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			errorResponse(w, fmt.Sprintf("can't read body: %v", err))
			return
		}

		err = json.Unmarshal(body, &parsed)
		if err != nil {
			errorResponse(w, fmt.Sprintf("can't parse: %v", err))
			return
		}

		addr, err := strconv.ParseInt(parsed.Data.Addr, 16, 32)
		if err != nil { 
			errorResponse(w, fmt.Sprintf("can't parse address: %v", err))
			return
		}

		complete := make(chan bool)

		c.Comm <- func(c *UserCpu) {
			if parsed.Data.Event == -1 { 
				if _, ok := c.Breakpoints[uint16(addr)]; ok {
					delete(c.Breakpoints, uint16(addr))
				}
			} else {
				c.Breakpoints[uint16(addr)] = parsed.Data.Event
			}

			complete <- true
		}

		success := <- complete

		out, _ := json.Marshal(&map[string]interface{}{
			"data": &map[string]interface{}{
				"success": success,
			},
		})
		fmt.Fprintf(w, "%s", string(out))
	}))

	stateSetter := func(state int, cb func(c *UserCpu) error) http.HandlerFunc {
		return mustNotBeSleeping(func(w http.ResponseWriter, r *http.Request, s *Sessionkv, c *UserCpu) {
			if(cb != nil) { 
				if err := cb(c); err != nil { 
					errorResponse(w, fmt.Sprintf("%v", err))
					return
				}
			}

			result := &requestResult{
				success: true,
				data:    "",
			}

			complete := make(chan bool)

			c.Comm <- func(c *UserCpu) {
				c.State = state
				complete <- true
			}

			_ = <-complete

			out, _ := json.Marshal(&map[string]interface{}{
				"data": &map[string]interface{}{
					"success": result.success,
				},
			})
			fmt.Fprintf(w, "%s", string(out))
		})
	}

	m.Post("/cpu/:name/boot", stateSetter(CpuRunning, nil))
	m.Post("/cpu/:name/continue", stateSetter(CpuRunning, func(c *UserCpu) (err error) {
		complete := make(chan error)

		c.Comm <- func(c *UserCpu) {
			complete <- c.MCU.Step()
		}
		err = <- complete

		return
	}))

	m.Post("/cpu/:name/step", stateSetter(CpuStopped, func(c *UserCpu) (err error) {
		complete := make(chan error)

		c.Comm <- func(c *UserCpu) {
			complete <- c.MCU.Step()
		}

		err = <- complete

		return
	}))

	m.Post("/cpu/:name/stop", stateSetter(CpuStopped, nil))

	m.Post("/cpu/:name/reset", mustCpu(func(w http.ResponseWriter, r *http.Request, s *Sessionkv, c *UserCpu) {
		complete := make(chan bool)

		c.LoadHexFromRedis(c.Image)
		c.Comm <- func(c *UserCpu) {
			c.MCU.SetRegs([16]uint16{0x4400})
			complete <- true
		}

		_ = <-complete

		out, _ := json.Marshal(&map[string]interface{}{
			"data": &map[string]interface{}{
				"success": true,
			},
		})
		fmt.Fprintf(w, "%s", string(out))
	}))

	return
}
