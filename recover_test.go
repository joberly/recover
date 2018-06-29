package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"testing"
	"time"
)

type testHandler interface {
	Handle(http.ResponseWriter, *http.Request)
	response() string
	desc() string
}

// TestRecoverMux tests the default, production-like behavior of the
// recoverMux. Ensures normal and panicking client handlers work as expected.
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
		t.Run(fmt.Sprintf("TestRecoverMux %s", th.desc()), func(t *testing.T) {
			mux := NewRecoverMux()
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
				t.Errorf("HTTP GET error %s\n", err.Error())
				return
			}

			// Read response body
			sc := bufio.NewScanner(resp.Body)
			b := sc.Scan()
			if !b {
				err = sc.Err()
				if err != nil {
					t.Errorf("read response error %s", sc.Err().Error())
				} else {
					t.Errorf("unexpected EOF")
				}
			} else {
				// Check response body equals expected value
				ln := sc.Text()
				if ln != th.response() {
					t.Errorf("response mismatch actual \"%s\" expected \"%s\"",
						ln, th.response())
				}
			}

			resp.Body.Close()

			// Close server
			err = s.Close()
			if err != nil {
				t.Errorf("server close error %s", err.Error())
			}
		})
	}
}

// TestDebugOKRecoverMux tests behavior of the recoverMux with the DumpStack
// flag set to true for a normal client handler.
func TestDebugOKRecoverMux(t *testing.T) {
	// Some server URL stuff
	addr := ":5050"
	path := "/test"
	url := "http://localhost" + addr + path

	// Test normal handler
	th := newTestHandlerOK("good path")
	mux := NewRecoverMux()
	mux.DumpStack = true
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		th.Handle(w, r)
	})
	s := &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	go s.ListenAndServe()

	resp, err := http.Get(url)
	if err != nil {
		t.Errorf("HTTP GET error %s\n", err.Error())
		return
	}

	// Read response body
	sc := bufio.NewScanner(resp.Body)
	b := sc.Scan()
	if !b {
		err = sc.Err()
		if err != nil {
			t.Errorf("read response error %s", sc.Err().Error())
		} else {
			t.Errorf("unexpected EOF")
		}
	} else {
		ln := sc.Text()
		// Check response body equals expected value
		if ln != th.response() {
			t.Errorf("response not equal to expected response")
		}
	}

	resp.Body.Close()

	err = s.Close()
	if err != nil {
		t.Errorf("server close error %s", err.Error())
	}
}

// TestDebugPanicRecoverMux tests behavior of the recoverMux with the DumpStack
// flag set to true for a panicking client handler.
func TestDebugPanicRecoverMux(t *testing.T) {
	// Some server URL stuff
	addr := ":5050"
	path := "/test"
	url := "http://localhost" + addr + path

	// Test normal handler
	th := newTestHandlerPanic("panicking with stack")
	mux := NewRecoverMux()
	mux.DumpStack = true
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		th.Handle(w, r)
	})
	s := &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	go s.ListenAndServe()

	resp, err := http.Get(url)
	if err != nil {
		t.Errorf("HTTP GET error %s\n", err.Error())
		return
	}

	// Validate parts of response body
	sc := bufio.NewScanner(resp.Body)

	// Check initial message
	b := sc.Scan()
	if !b {
		err = sc.Err()
		if err != nil {
			t.Errorf("read response error %s", sc.Err().Error())
		} else {
			t.Errorf("unexpected EOF")
		}
	} else {
		ln := sc.Text()
		if ln != string(th.response()) {
			t.Errorf("initial string mismatch: \"%s\"\n", ln)
		}

		// Skip a line
		sc.Scan()

		exps := []string{
			"goroutine",
			"Stack",
			"",
			"recoverHandler",
			"",
			"func1",
			"",
			"panic",
			"",
			"Handle",
		}

		for _, lnexp := range exps {
			// Scan the line with the function name and check it
			b = sc.Scan()
			if b {
				ln = sc.Text()
				if !strings.Contains(ln, lnexp) {
					t.Errorf("stack line mismatch, actual \"%s\" exp \"%s\"",
						ln, lnexp)
				}
			} else {
				t.Errorf("unexpected EOF")
				break
			}
		}
	}

	resp.Body.Close()

	// Close server
	err = s.Close()
	if err != nil {
		t.Errorf("test error: %s close error %s", th.desc(), err.Error())
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

func (h *testHandlerOK) response() string {
	return h.resp
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

func (h *testHandlerPanic) response() string {
	return "Something went wrong."
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

func (h *testHandlerWithCode) response() string {
	return h.resp
}

func (h *testHandlerWithCode) desc() string {
	return fmt.Sprintf("testHandlerWithCode (%d %s)", h.sc, h.resp)
}
