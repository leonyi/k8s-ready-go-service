// Package debug provides handler support for the debugging endpoints of the service. This includes pprof and expvar.
package debug

import (
	"expvar"
	"net/http"
	"net/http/pprof"

	"github.com/arl/statsviz"
)

func Mux() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	mux.HandleFunc("/debug/vars", expvar.Handler().ServeHTTP)

	statsviz.Register(mux)

	return mux
}
