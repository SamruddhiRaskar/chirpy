package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

// Middleware: increments the counter for every /app request
func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

// metrics handler
func (cfg *apiConfig) metricsHandler(w http.ResponseWriter, r *http.Request) {
	hits := cfg.fileserverHits.Load()

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	fmt.Fprintf(w, "Hits: %d\n", hits)
}

// /reset handler
func (cfg *apiConfig) resetHandler(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	w.Write([]byte("Hits reset to 0\n"))
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func main() {
	// Create API config
	apiCfg := &apiConfig{}

	// Create router
	mux := http.NewServeMux()

	fileserver := http.FileServer(http.Dir("."))
	// here dir(.) says that look at the current directory means whats in folder

	appHandler := http.StripPrefix("/app", fileserver)

	mux.Handle(
		"/app/",
		apiCfg.middlewareMetricsInc(appHandler),
	)

	// Other routes
	mux.HandleFunc("GET /healthz", healthzHandler)
	mux.HandleFunc("GET /metrics", apiCfg.metricsHandler)
	mux.HandleFunc("POST /reset", apiCfg.resetHandler)

	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	server.ListenAndServe()
	//here it start the server

}
