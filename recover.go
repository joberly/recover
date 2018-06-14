package main

import (
	"fmt"
	"net/http"
)

type recoverMux struct {
	mux *http.ServeMux
}

func newRecoverMux() *recoverMux {
	return &recoverMux{
		mux: http.NewServeMux(),
	}
}

func (rmux *recoverMux) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	f := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if x := recover(); x != nil {
				recoverHandler(w, r)
			}
		}()
		handler(w, r)
	}
	rmux.mux.HandleFunc(pattern, f)
}

func (rmux *recoverMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rmux.mux.ServeHTTP(w, r)
}

func recoverHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Something went wrong.")
}
