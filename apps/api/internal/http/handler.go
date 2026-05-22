package http

import (
	stdhttp "net/http"

	"github.com/go-chi/chi/v5"
)

func NewHandler() stdhttp.Handler {
	r := chi.NewRouter()

	r.Get("/healthz", func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(stdhttp.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	return r
}
