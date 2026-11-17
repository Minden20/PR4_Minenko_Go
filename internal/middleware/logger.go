package middleware

import (
	"log"
	"net/http"
	"time"

	chimw "github.com/go-chi/chi/v5/middleware"
)

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		status := ww.Status()
		if status == 0 {
			status = http.StatusOK
		}
		log.Printf("[HTTP] [%s] %s - %s %s -> %d %s (%v)",
			chimw.GetReqID(r.Context()),
			r.RemoteAddr,
			r.Method,
			r.URL.RequestURI(),
			status,
			http.StatusText(status),
			time.Since(start),
		)
	})
}
