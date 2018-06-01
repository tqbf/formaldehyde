package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/go-zoo/bone"
	"github.com/latacora/formaldehyd"
	"github.com/latacora/shamework"
)

type contextKey string

func (c contextKey) String() string {
	return "context key " + string(c)
}

var (
	contextKeyApp = contextKey("app")
)

func handleFoo(w http.ResponseWriter, r *http.Request) {
	a := r.Context().Value(contextKeyApp).(*app)

	fmt.Fprintf(w, "%s\n", a.Root.JSON())
}

func handleRoot(w http.ResponseWriter, rq *http.Request) {
	r := shamework.NewResponder(w, rq)
	r.Success()
}

type app struct {
	Root *formaldehyd.Node
	log  *shamework.RequestLogger
}

func (a *app) handler(rawHandler http.Handler) http.Handler {
	var h http.Handler

	h = shamework.Inject(rawHandler, contextKeyApp, a)
	h = a.log.Middleware(h)

	return h
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("server <file>")
	}

	buf, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		log.Fatalf("can't read %s: %s", os.Args[1], err)
	}

	a := &app{
		log: shamework.NewRequestLogger(true, true, true, os.Stderr),
	}

	a.Root, err = formaldehyd.Parse(buf)
	if err != nil {
		log.Fatalf("can't parse %s: %s", os.Args[1], err)
	}

	mux := bone.New()

	mux.Get("/", a.handler(http.HandlerFunc(handleRoot)))
	mux.Get("/form/foo", a.handler(http.HandlerFunc(handleFoo)))
	mux.Get("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("listening on :%s", port)

	log.Fatal(
		http.ListenAndServe(
			fmt.Sprintf(":%s", port),
			mux,
		))

	panic("notreached")
}
