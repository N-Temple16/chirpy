package main

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func handleHealthz (w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	w.WriteHeader(http.StatusOK)

	w.Write([]byte("OK"))
}

func (cfg *apiConfig) handleNumRequests (w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	count := cfg.fileserverHits.Load()
	body := fmt.Sprintf("Hits: %d\n", count)
	w.WriteHeader(http.StatusOK)	
	w.Write([]byte(body))
}

func (cfg *apiConfig) handleReset (w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)	
	w.Write([]byte("OK"))
}


func main() {
	const filepathRoot = "."
	const port = "8080"
	apiCfg := &apiConfig{}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/healthz", handleHealthz)
	mux.HandleFunc("GET /api/metrics", apiCfg.handleNumRequests)
	mux.HandleFunc("POST /api/reset", apiCfg.handleReset)
	handler := http.StripPrefix("/app", http.FileServer(http.Dir(filepathRoot)))
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(handler))

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)
	log.Fatal(server.ListenAndServe())
}