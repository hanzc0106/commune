package http

import (
	stdhttp "net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
)

type Options struct {
	StaticDir  string
	APIHandler stdhttp.Handler
}

func NewHandler(options Options) stdhttp.Handler {
	r := chi.NewRouter()

	r.Get("/healthz", func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(stdhttp.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	if options.APIHandler != nil {
		r.Mount("/api", options.APIHandler)
	}

	if options.StaticDir != "" {
		fileServer := stdhttp.FileServer(stdhttp.Dir(options.StaticDir))
		r.NotFound(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
			cleanPath := strings.TrimPrefix(filepath.Clean(r.URL.Path), string(filepath.Separator))
			path := filepath.Join(options.StaticDir, cleanPath)
			if info, err := os.Stat(path); err == nil && !info.IsDir() {
				fileServer.ServeHTTP(w, r)
				return
			}
			stdhttp.ServeFile(w, r, filepath.Join(options.StaticDir, "index.html"))
		})
	}

	return r
}
