package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
)

// A recoverMux recovers from panics during an http.Handler, sending an error
// response to a client in the event a panic occurs.
type recoverMux struct {
	mux *http.ServeMux
}

// NewRecoverMux returns a new recoverMux, creating and wrapping a new
// http.ServeMux.
func newRecoverMux() *recoverMux {
	return &recoverMux{
		mux: http.NewServeMux(),
	}
}

// HandleFunc wraps the caller's handler with a recovery handler which recovers
// from panics in the caller's handler. Response data is only written if the
// caller's handler completes without panicking.
func (rmux *recoverMux) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	f := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if x := recover(); x != nil {
				recoverHandler(w, r)
			}
		}()
		rw := newResponseWriter(w)
		handler(rw, r)
		err := rw.complete()
		if err != nil {
			log.Printf("error completing handler (URL %s): %s", r.URL, err.Error())
		}
	}
	rmux.mux.HandleFunc(pattern, f)
}

// ServeHTTP uses the wrapped http.ServeMux to serve recoverable handlers.
func (rmux *recoverMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rmux.mux.ServeHTTP(w, r)
}

// RecoverHandler is the handler invoked when the client's handler panics.
func recoverHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Something went wrong.")
}

// A responseWriter wraps http.ResponseWriter for a recoverMux.
type responseWriter struct {
	buf *bytes.Buffer
	sc  int
	w   http.ResponseWriter
}

// NewResponseWriter returns a new responseWriter for an http.ResponseWriter
// for passing to a real handler by the recoverMux's handler.
func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		buf: &bytes.Buffer{},
		sc:  -1, // Flag value to indicate it has not been written
		w:   w,
	}
}

// Header simply returns the real http.ResponseWriter's Header.
func (w *responseWriter) Header() http.Header {
	return w.w.Header()
}

// Write buffers response writes so the recoverMux can throw them away in case
// the real handler panics.
func (w *responseWriter) Write(p []byte) (int, error) {
	return w.buf.Write(p)
}

// WriteHeader saves the status code from the real handler so it can be thrown
// away if the real handler panics. It panics if the code is not valid.
func (w *responseWriter) WriteHeader(statusCode int) {
	if statusCode < 100 || statusCode > 999 {
		panic(fmt.Sprintf("invalid WriteHeader code %v", statusCode))
	}
	w.sc = statusCode
}

// Complete sends the full response to the client in cases where the real
// handler completes without panicking.
func (w *responseWriter) complete() error {
	if w.sc > 0 {
		w.w.WriteHeader(w.sc)
	}
	_, err := w.w.Write(w.buf.Bytes())
	return err
}
