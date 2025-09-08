package main

import (
	"encoding/json"
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

func handleValidation (w http.ResponseWriter, r *http.Request) {
	type parameters struct {
        Body string `json:"body"`
    }
	type errorReturn struct {
        Error string `json:"error"`
    }
	type successReturn struct {
        Valid bool `json:"valid"`
    }

	w.Header().Set("Content-Type", "application/json")

    decoder := json.NewDecoder(r.Body)
    params := parameters{}
    if err := decoder.Decode(&params); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		dat, mErr := json.Marshal(errorReturn{Error: "Something went wrong"})
		if mErr != nil {
			w.Write([]byte(`{"error":"Something went wrong"}`))
			return
		}
		w.Write(dat)
		return
    }
	if len(params.Body) > 140 {
		w.WriteHeader(http.StatusBadRequest)
		dat, mErr := json.Marshal(errorReturn{Error: "Chirp is too long"})
		if mErr != nil {
			w.Write([]byte(`{"error":"Something went wrong"}`))
			return
		}
		w.Write(dat)
		return
	}
	w.WriteHeader(http.StatusOK)
	dat, mErr := json.Marshal(successReturn{Valid: true})
	if mErr != nil {
		w.Write([]byte(`{"error":"Something went wrong"}`))
		return
	}
	w.Write(dat)
}

func (cfg *apiConfig) handleNumRequests (w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	count := cfg.fileserverHits.Load()
	body := fmt.Sprintf("<html>\n<body>\n<h1>Welcome, Chirpy Admin</h1>\n<p>Chirpy has been visited %d times!</p>\n</body>\n</html>", count)
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
	mux.HandleFunc("GET /admin/metrics", apiCfg.handleNumRequests)
	mux.HandleFunc("POST /admin/reset", apiCfg.handleReset)
	mux.HandleFunc("POST /api/validate_chirp", handleValidation)
	handler := http.StripPrefix("/app", http.FileServer(http.Dir(filepathRoot)))
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(handler))

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)
	log.Fatal(server.ListenAndServe())
}