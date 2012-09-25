package main

import (
	"fmt"
	"bytes"
	"net/http"
	"github.com/bmizerany/pat"
	"html/template"
)

type sessionHandler func(w http.ResponseWriter, r *http.Request, s *Sessionkv)
 
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
			http.Redirect(w, r, "/login", 302)
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

func CpuInterface(templates string) (m *pat.PatternServeMux) {
	m = pat.New()

	render := func(w http.ResponseWriter, name string, data interface{} ) {
		t := template.Must(template.ParseFiles(fmt.Sprintf("%s/%s", templates, name)))
		t.Execute(w, data)
	}

	m.Get("/login", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { 
		render(w, "login.html", nil)
	}));

	m.Post("/login", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { 

	}));

	m.Post("/cpu", mustSession(func(w http.ResponseWriter, r *http.Request, s *Sessionkv) {
		
	}))
	
	return 
}
