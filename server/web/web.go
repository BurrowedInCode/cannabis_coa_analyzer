package web

import (
	"embed"
	"errors"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:dist
var distFS embed.FS

func Handler() http.Handler {
	dist, _ := fs.Sub(distFS, "dist")
	fileServer := http.FileServer(http.FS(dist))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/")
		if p == "" {
			p = "index.html"

		}
		if _, err := fs.Stat(dist, p); errors.Is(err, fs.ErrNotExist) {
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})
}
