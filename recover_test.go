package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"testing"
	"time"
)

type testHandler interface {
	Handle(http.ResponseWriter, *http.Request)
	response() []byte
	desc() string
}

func TestRecoverMux(t *testing.T) {
	// Some server URL stuff
	addr := ":5050"
	path := "/test"
	url := "http://localhost" + addr + path

	// Test table of handlers
	ths := []testHandler{
		newTestHandlerOK("good path"),
		newTestHandlerPanic("panic message"),
		newTestHandlerWithCode(201, http.StatusText(201)),
		newTestHandlerWithCode(1, "Something went wrong."),
		newTestHandlerWithCode(1000, "Something went wrong."),
	}

	// Run each test in the table
	for _, th := range ths {
		// Create recoverMux under test
		mux := newRecoverMux()
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			th.Handle(w, r)
		})
		s := &http.Server{
			Addr:    addr,
			Handler: mux,
		}

		// Start server
		go s.ListenAndServe()

		// Get response from server
		resp, err := http.Get(url)

		// Ensure normal response
		if err != nil {
			t.Errorf("test error: %s HTTP GET error %s\n", th.desc(), err.Error())
		} else {
			// Read response body
			var bresp []byte
			bresp, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Errorf("test error: %s read response error %s", th.desc(), err.Error())
			}

			// Check response body equals expected value
			if !bytes.Equal(th.response(), bresp) {
				t.Errorf("test error: %s response not equal to expected response", th.desc())
			}

			resp.Body.Close()
		}

		// Close server
		err = s.Close()
		if err != nil {
			t.Errorf("test error: %s close error %s", th.desc(), err.Error())
		}
	}
}

type testHandlerOK struct {
	resp string
}

func newTestHandlerOK(resp string) *testHandlerOK {
	return &testHandlerOK{
		resp: resp,
	}
}

func (h *testHandlerOK) Handle(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, h.resp)
}

func (h *testHandlerOK) response() []byte {
	return []byte(h.resp)
}

func (h *testHandlerOK) desc() string {
	return "testHandlerOK (" + h.resp + ")"
}

// TestHandlerPanic panics in its handler after trying to write response data
// to the http.ResponseWriter.
type testHandlerPanic struct {
	msg string // panic message
	rng *rand.Rand
}

func newTestHandlerPanic(msg string) *testHandlerPanic {
	return &testHandlerPanic{
		msg: msg,
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Handle attempts to send a response but then panics. Attempted message
// contains a random number to try to make it different for each test
// so it can't be accidentally anticipated.
func (h *testHandlerPanic) Handle(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "this should never be received %d", h.rng.Uint64())
	panic(h.msg)
}

func (h *testHandlerPanic) response() []byte {
	return []byte("Something went wrong.")
}

func (h *testHandlerPanic) desc() string {
	return "testHandlerPanic (" + h.msg + ")"
}

// TestHandlerWithCode changes the status code but the handler itself completes
// without explicitly panicking.
type testHandlerWithCode struct {
	sc   int
	resp string
}

func newTestHandlerWithCode(statusCode int, resp string) *testHandlerWithCode {
	return &testHandlerWithCode{
		sc:   statusCode,
		resp: resp,
	}
}

func (h *testHandlerWithCode) Handle(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(h.sc) // note this may panic if status code is out of bounds
	fmt.Fprintf(w, h.resp)
}

func (h *testHandlerWithCode) response() []byte {
	return []byte(h.resp)
}

func (h *testHandlerWithCode) desc() string {
	return fmt.Sprintf("testHandlerWithCode (%d %s)", h.sc, h.resp)
}
