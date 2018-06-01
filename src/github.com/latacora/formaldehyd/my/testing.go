package my

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

// BUG(tqbf): rename functions

// A TestResponse tries to capture all the stuff from an HTTP Response
// that you might want to peek at when validating a test.
type TestResponse struct {
	Path     string
	Method   string
	Code     int
	Body     []byte
	Err      error
	Response *http.Response
	T        *testing.T
	Cookies  map[string]string
}

func (r *TestResponse) String() string {
	err := ""
	if r.Err != nil {
		err = fmt.Sprintf(" ERROR: %s", r.Err)
	}

	return fmt.Sprintf(`%s %s (response: %d, %d bytes)%s
%s
`, r.Method, r.Path, r.Code, len(r.Body), err, hex.Dump(r.Body))
}

// Reader returns an io.Reader for the body of the Response
func (t *TestResponse) Reader() io.Reader {
	return bytes.NewReader(t.Body)
}

// Given a net/http Response, get a TestResponse.
func TestResponseFromResponse(res *http.Response) (ret *TestResponse) {
	ret = &TestResponse{
		Code:     res.StatusCode,
		Response: res,
	}

	ret.Body, ret.Err = ioutil.ReadAll(res.Body)
	if ret.Err != nil {
		return
	}

	ret.Cookies = map[string]string{}

	for _, cookie := range res.Cookies() {
		ret.Cookies[cookie.Name] = cookie.Value
	}

	return ret
}

// Given just an error, get a TestResponse to convey the error to the
// test (obviously everything else will be blank)
func testResponseFromError(err error) (ret *TestResponse) {
	return &TestResponse{
		Err: err,
	}
}

// OK returns true if the Response is superficially OK (no errors,
// etc)
func (r *TestResponse) OK() bool {
	return r.Err == nil
}

// AssetCode returns false, logs, and flags a test failure if the
// response has the wrong response code
func (r *TestResponse) AssertCode(code int) bool {
	r.T.Helper()
	if !r.OK() {
		r.T.Fatalf("request error: %s", r.Err)
		return false
	}

	if r.Code != code {
		r.T.Fatalf("expected code %d, got %d", code, r.Code)
		return false
	}

	return true
}

// AssetCode returns false, logs, and flags a test failure if the
// response has the wrong response code or fails to unmarshal to the
// expected object via JSON
func (r *TestResponse) AssertJson(v interface{}, code int) bool {
	r.T.Helper()
	if !r.AssertCode(code) {
		return false
	}

	decoder := json.NewDecoder(r.Reader())
	if err := decoder.Decode(v); err != nil {
		r.Err = err
		r.T.Fatalf("couldn't decode: %s", err)
		return false
	}

	return true
}

func (r *TestResponse) Assert200Contains(flag string) bool {
	r.T.Helper()
	if !r.AssertCode(200) {
		return false
	}

	if bytes.Index(r.Body, []byte(flag)) == -1 {
		r.T.Fatalf("flag '%s' not found", flag)
		return false
	}

	return true
}

// A Tester is a simple object that allows us to make test request to a
// given URL. Testers are how you get TestResponses. This is really just
// a simple wrapper around the net/http client (you can "test" anything
// with it)
type Tester struct {
	SessionKey string
	Session    string
	BaseURL    string
	T          *testing.T
}

// Given a base URL and the name of the cookie containing sessions,
// get a new tester.
func NewTester(base, sessionKey string) *Tester {
	return &Tester{
		SessionKey: sessionKey,
		BaseURL:    base,
	}
}

// Reset resets the tester, in particular forgetting the current remembered
// session
func (t *Tester) Reset() {
	t.Session = ""
}

// from a test response, find the session cookie and remember it
func (t *Tester) recoverSession(r *TestResponse) {
	if !r.OK() {
		return
	}

	for k, v := range r.Cookies {
		if k == t.SessionKey {
			t.Session = v
			break
		}
	}
}

// to an outgoing request, add the session cookie, if we have one yet
func (t *Tester) addSession(req *http.Request) {
	if t.Session != "" {
		req.AddCookie(&http.Cookie{
			Name:  t.SessionKey,
			Value: t.Session,
		})
	}
}

// launch an HTTP request with a valid session and use its response
// to create and return a TestResponse
func (t *Tester) exec(req *http.Request, path string) (ret *TestResponse) {
	t.addSession(req)

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	res, err := client.Do(req)
	if err != nil {
		ret = testResponseFromError(err)
		ret.T = t.T
		ret.Path = path
		ret.Method = req.Method
		return
	}

	ret = TestResponseFromResponse(res)
	ret.T = t.T
	ret.Path = path
	ret.Method = req.Method

	t.recoverSession(ret)

	return
}

// Perform an HTTP GET to the given relative URL path, returning
// a TestResponse
func (t *Tester) Get(path string) (ret *TestResponse) {
	req, err := http.NewRequest("GET", t.BaseURL+path, nil)
	if err != nil {
		return testResponseFromError(err)
	}

	return t.exec(req, path)
}

// Perform an HTTP POST to the given relative URL path, including
// the "data" string as the body, returning
// a TestResponse
func (t *Tester) Post(path, data string) (ret *TestResponse) {
	req, err := http.NewRequest("POST", t.BaseURL+path,
		strings.NewReader(data))
	if err != nil {
		return testResponseFromError(err)
	}

	return t.exec(req, path)
}

// Perform an HTTP POST w/ application/x-www-form-urlencoded to the given
// relative URL path, including the "data" string as the body, returning
// a TestResponse
func (t *Tester) PostForm(path, data string) (ret *TestResponse) {
	req, err := http.NewRequest("POST", t.BaseURL+path,
		strings.NewReader(data))
	if err != nil {
		return testResponseFromError(err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return t.exec(req, path)
}

// Perform an HTTP POST to the given relative URL path, including
// the "data" object as the body of the request after encoding it
// as JSON, returning a TestResponse
func (t *Tester) PostJson(path string, data interface{}) (ret *TestResponse) {
	buf, err := json.Marshal(data)
	if err != nil {
		return testResponseFromError(err)
	}

	req, err := http.NewRequest("POST", t.BaseURL+path,
		bytes.NewReader(buf))
	if err != nil {
		return testResponseFromError(err)
	}

	return t.exec(req, path)
}

// Perform an HTTP PUT to the given relative URL path, including
// the "data" string as the body, returning
// a TestResponse
func (t *Tester) Put(path, data string) (ret *TestResponse) {
	req, err := http.NewRequest("PUT", t.BaseURL+path,
		strings.NewReader(data))
	if err != nil {
		return testResponseFromError(err)
	}

	return t.exec(req, path)
}

// Perform an HTTP PUT to the given relative URL path, including
// the "data" object as the body of the request after encoding it
// as JSON, returning a TestResponse
func (t *Tester) PutJson(path string, data interface{}) (ret *TestResponse) {
	buf, err := json.Marshal(data)
	if err != nil {
		return testResponseFromError(err)
	}

	req, err := http.NewRequest("PUT", t.BaseURL+path,
		bytes.NewReader(buf))
	if err != nil {
		return testResponseFromError(err)
	}

	return t.exec(req, path)
}

// Perform an HTTP DELETE to the given relative URL path, returning
// a TestResponse
func (t *Tester) Delete(path string) (ret *TestResponse) {
	req, err := http.NewRequest("DELETE", t.BaseURL+path, nil)
	if err != nil {
		return testResponseFromError(err)
	}

	return t.exec(req, path)
}

type T struct {
	*testing.T
}

func NewT(t *testing.T) *T {
	return &T{t}
}

func (t *T) OK(err error, why ...string) bool {
	t.Helper()
	if err != nil {
		whyr := ""
		if len(why) > 0 {
			whyr = why[0]
		}
		t.Fatalf("%s: %s", whyr, err)
		return false
	}
	return true
}

func (t *T) Expect(s, want string) bool {
	t.Helper()
	if s != want {
		t.Fatalf("expected '%s', got '%s'", want, s)
		return false
	}

	return true
}

func (t *T) ExpectContains(s, want string) bool {
	t.Helper()
	if strings.Index(s, want) == -1 {
		t.Fatalf("expected '%s' to contain '%s'", s, want)
		return false
	}
	return true
}

func (t *T) ExpectNotContains(s, want string) bool {
	t.Helper()
	if strings.Index(s, want) != -1 {
		t.Fatalf("expected '%s' not to contain '%s'", s, want)
		return false
	}
	return true
}

func (t *T) ExpectInt(s, want int) bool {
	t.Helper()
	if s != want {
		t.Fatalf("expected '%d', got '%d'", want, s)
		return false
	}
	return true
}
