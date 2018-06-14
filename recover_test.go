package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
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

	// Test table
	ths := []testHandler{
		newTestHandlerOK("handler OK"),
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

func newTestHandlerOK(resp string) testHandler {
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
	return h.resp
}
