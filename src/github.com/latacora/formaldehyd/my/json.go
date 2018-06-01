// Package webapp contains helpful utilities for building web applications
// in go.

package my

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"
	"github.com/yosssi/ace"
)

// RenderJson writes "obj" to "w" as JSON, setting the appropriate headers. "obj" must
// be a pointer to an object that can be JSON-serialized. If JSON marshalling for "obj"
// fails, a 500 error will be written instead.
func RenderJson(w http.ResponseWriter, obj interface{}) error { //
	js, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)

	return nil
}

// Status is a reasonable default "ok"/"error" JSON object to write for situations where
// "ok" or an error is the only reasonable response
type Status struct {
	Ok    bool   `json:"ok"`
	Error string `json:"error"`
}

// RenderJsonError takes a standard Golang error object and writes it to the HTTP client
// as a "Status" object with its "error" set, and its "ok" field set to "false".
func RenderJsonError(w http.ResponseWriter, err error) error { //
	js, err := json.Marshal(&Status{
		Ok:    false,
		Error: err.Error(),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)

	return nil
}

func ReadJson(dst interface{}, w http.ResponseWriter, r *http.Request) bool {
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dst); !OK(err) {
		RenderJsonErrorWithResponseCode(w, err, 500)
		return false
	}
	return true
}

// RenderJsonError takes a standard Golang error object and writes it to the HTTP client
// as a "Status" object with its "error" set, and its "ok" field set to "false", and also sets the response
// code.  If you don't provide a response code, we'll sensibly default to HTTP 500.
func RenderJsonErrorWithResponseCode(w http.ResponseWriter, err error, code int) error {
	js, err := json.Marshal(&Status{
		Ok:    false,
		Error: err.Error(),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	if code == 0 {
		code = http.StatusInternalServerError
	}

	w.WriteHeader(code)
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)

	return nil
}

// DecodeOrError attempts to decode the JSON body of an http request to "v", and if it fails,
// returns a nonzero error (if it succeeds, "v" will be filled in from the JSON body)
func DecodeOrError(r *http.Request, v interface{}) error {
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(v); err != nil {
		return err
	}
	return nil
}

// DecodeOrJsonError works like DecodeOrError but also, if the attempt to decode fails,
// will write the returned error to the client as a JSON "Status" object, saving you the
// trouble of having to write that error yourself.
func DecodeOrJsonError(w http.ResponseWriter, r *http.Request, v interface{}) bool {
	if err := DecodeOrError(r, v); err != nil {
		RenderJsonErrorWithResponseCode(w, err, 500)
		return false
	}
	return true
}

// RenderJsonOk tells the client things were OK, using a JSON "Status" object.
func RenderJsonOk(w http.ResponseWriter) error { //
	js, err := json.Marshal(&Status{
		Ok: true,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)

	return nil
}

//RenderJsonAuthError tells the client to try again with BasicAuth.  Browsers will handily cache
//provided creds for this for us, so this makes it very explorable with a browser.
func RenderJsonBasicAuthError(w http.ResponseWriter, msg string, realm string) error {
	js, err := json.Marshal(&Status{
		Ok:    false,
		Error: msg,
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	w.WriteHeader(http.StatusUnauthorized)
	w.Header().Set("WWW-Authenticate", fmt.Sprintf("Basic realm=\"%s\"", realm))
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
	return nil
}

// Write an ACE template. You shouldn't ever need to do this because we use React.
func HandleTemplate(name string, data map[string]string) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		tpl, err := ace.Load("html/template", fmt.Sprintf("html/%s", name), nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		tmpdata := map[string]string{"Msg": "Remove Me!"}
		if data != nil {
			for k, v := range data {
				tmpdata[k] = v
			}
		}

		if err := tpl.Execute(w, tmpdata); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// FormInt recovers an integer from a POST FORM argument

func FormInt(r *http.Request, key string) (ret int64, err error) {
	v := r.FormValue(key)
	if v == "" {
		return 0, fmt.Errorf("no such key")
	}

	return strconv.ParseInt(v, 10, 32)
}

// ParamInt recovers an integer from an httprouter URL parameter

func ParamInt(p httprouter.Params, key string) (ret int, err error) {
	offstring := p.ByName(key)
	if offstring == "" {
		return 0, errors.New("key not found")
	}

	return strconv.Atoi(offstring)
}
