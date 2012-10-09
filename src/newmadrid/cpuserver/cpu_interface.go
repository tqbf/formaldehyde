package main

import (
	"log"
	"io/ioutil"
	"fmt"
	"bytes"
	"net/http"
	"github.com/bmizerany/pat"
	"html/template"
	"encoding/json"
)

type sessionHandler func(w http.ResponseWriter, r *http.Request, s *Sessionkv)
type cpuHandler func(w http.ResponseWriter, r *http.Request, s *Sessionkv, c *UserCpu)

func mustCpu(handler cpuHandler) http.HandlerFunc {
	return mustSession(func(w http.ResponseWriter, r *http.Request, s *Sessionkv) {
		cpu := GetCpu(s.Map()["name"])
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
					Name: "S",
					Value: string(ses.Encode()),
				})
			}
		}
	}
}

func CpuInterface(templates string, redis *RedisLand) (m *pat.PatternServeMux) {
	m = pat.New()

	render := func(w http.ResponseWriter, name string, data interface{} ) {
		t := template.Must(template.ParseFiles(fmt.Sprintf("%s/%s", templates, name)))
		t.Execute(w, data)
	}

	m.Get("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { 
		http.Redirect(w, r, "/cpu", 302)		
	}));

	m.Get("/login", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { 
		render(w, "login.html", nil)
	}));

	m.Post("/login", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { 
		s := NewSession()
		s.Map()["name"] = r.FormValue("name")

		http.SetCookie(w, &http.Cookie{
			Name: "S", 
			Value: string(s.Encode()),
		})
		
		http.Redirect(w, r, "/cpu", 302)		
	}))

	m.Post("/logout", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { 
		http.SetCookie(w, &http.Cookie{
			Name: "S",
			MaxAge: -1, 
			Value: "",
		})
		
		http.Redirect(w, r, "/login", 302)		
	}))

	m.Get("/cpu/:name", mustSession(func(w http.ResponseWriter, r *http.Request, s *Sessionkv) {
		type response struct {
			Name string
			Cpu interface{}
		}

		cpu := GetCpu(s.Map()["name"])

		render(w, "cpu.html", response{
			Name: s.Map()["name"],
			Cpu: cpu.MCU,
		})			
	}))


	type RegsData struct {
		Regs [16]int `json:"regs"`
	}

	type RegsTop struct {
		Data RegsData `json:"data"`
	}

	m.Get("/cpu/:name/regs_form", mustCpu(func(w http.ResponseWriter, r *http.Request, s *Sessionkv, c *UserCpu) {
		render(w, "regs.html", struct{
			Regs [16]uint16
		}{ c.MCU.GetRegs() })			
	}))

	m.Get("/cpu/:name/regs", mustCpu(func(w http.ResponseWriter, r *http.Request, s *Sessionkv, c *UserCpu) {
		var regsout [16]int

		for i, v := range(c.MCU.GetRegs()) {
			regsout[i] = int(v)
		}

		log.Println(c.MCU.GetRegs())

		log.Printf("-> %p\n", c.MCU)

		out, _ := json.Marshal(RegsTop{
			Data: RegsData{
				Regs: regsout,
			},
		})

		fmt.Fprintf(w, "%s", string(out))
	}))

	type requestResult struct {
		success bool
		data string
	}

	m.Get("/cpu/:name/state", mustCpu(func(w http.ResponseWriter, r *http.Request, s *Sessionkv, c *UserCpu) {
		result := &requestResult{
			success: true,
			data:"",
		}

		complete := make(chan int)

		c.Comm <- func(c *UserCpu) { 
			complete <- c.State
		}
	
		res := <- complete
		result.data = fmt.Sprintf("%d", res)	
		out, _ := json.Marshal(result)
		fmt.Fprintf(w, "%s", string(out))		
	}))

	m.Post("/cpu/:name/regs", mustCpu(func(w http.ResponseWriter, r *http.Request, s *Sessionkv, c *UserCpu) {
		complete := make(chan bool)

		result := &requestResult{
			success: true,
			data: "",
		}
		parsed := &RegsTop{}
		body, err := ioutil.ReadAll(r.Body)
		if err == nil {
			err = json.Unmarshal(body, &parsed)
			if err == nil {
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
			}
		}

		result.success =<- complete
		out, _ := json.Marshal(result)
		fmt.Fprintf(w, "%s", string(out))		
	}))

	m.Post("/cpu/:name/load", mustCpu(func(w http.ResponseWriter, r *http.Request, s *Sessionkv, c *UserCpu) {
		type loadRequest struct {
			key string
		}

		parsed := &loadRequest{}
		result := &requestResult{
			success: false,
			data: "",
		}
		
		body, err := ioutil.ReadAll(r.Body)
		if err == nil {
			err = json.Unmarshal(body, &parsed)
			if err == nil {
				err = c.LoadHexFromRedis(parsed.key)
			}
		}

		if err != nil {
			result.data = fmt.Sprintf("%v", err)
		} else {
			result.success = true
		}

		out, _ := json.Marshal(result)
		fmt.Fprintf(w, "%s", string(out))
	}))

	m.Get("/cpu/:name/events", mustCpu(func(w http.ResponseWriter, r *http.Request, s *Sessionkv, c *UserCpu) {
		type results struct {
			success bool
			events map[uint16]int
		}

		result := &results{}
		complete := make(chan bool)

		c.Comm <- func(c *UserCpu) {
			result.events = c.Breakpoints
			complete <- true
		}

		result.success = <- complete

		out, _ := json.Marshal(result)
		fmt.Fprintf(w, "%s", string(out))

	}))

	m.Post("/cpu/:name/event", mustCpu(func(w http.ResponseWriter, r *http.Request, s *Sessionkv, c *UserCpu) {
	 	type CpuEvent struct {
	 		Addr	uint16
	 		Event	int
	 	}

		parsed := &CpuEvent{}
		result := &requestResult{
			success: false,
			data: "",
		}
		
		body, err := ioutil.ReadAll(r.Body)
		if err == nil {
			err = json.Unmarshal(body, &parsed)
			if err == nil {
				complete := make(chan bool)

				c.Comm <- func(c *UserCpu) { 
					c.Breakpoints[parsed.Addr] = parsed.Event
					complete <- true
				}

				result.success = <- complete
			}
		}

		if err != nil {
			result.data = fmt.Sprintf("%v", err)
		} 

		out, _ := json.Marshal(result)
		fmt.Fprintf(w, "%s", string(out))
	}))

	stateSetter := func(state int) http.HandlerFunc {
		return mustCpu(func(w http.ResponseWriter, r *http.Request, s *Sessionkv, c *UserCpu) {
			result := &requestResult{
				success: true,
				data: "",
			}

			complete := make(chan bool)

			c.Comm <- func(c *UserCpu) { 
				c.State = state
				complete <- true
			}

			_ = <- complete
			out, _ := json.Marshal(result)
			fmt.Fprintf(w, "%s", string(out))
		})
	}

	m.Post("/cpu/:name/boot", stateSetter(CpuRunning))
	m.Post("/cpu/:name/continue", stateSetter(CpuRunning))
	m.Post("/cpu/:name/stop", stateSetter(CpuStopped))

	return 
}
