package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

const metricsHTML = `
<html>
	<body>
		<h1>Welcome, Chirpy Admin</h1>
		<p>Chirpy has been visited %d times!</p>
	</body>
</html>
`

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func handlerReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) handlerMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(200)
	hits := cfg.fileserverHits.Load()
	w.Write([]byte(fmt.Sprintf(metricsHTML, hits)))
}

func (cfg *apiConfig) handlerReset(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	cfg.fileserverHits.Store(0)
	w.Write([]byte("Hits reset."))
}

func main() {
	mux := http.NewServeMux()
	apiCfg := apiConfig{}
	handlerApp := http.FileServer(http.Dir("."))
	handlerApp = http.StripPrefix("/app", handlerApp)
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(handlerApp))
	mux.Handle("GET /api/healthz", http.HandlerFunc(handlerReadiness))
	mux.Handle("GET /admin/metrics", http.HandlerFunc(apiCfg.handlerMetrics))
	mux.Handle("POST /admin/reset", http.HandlerFunc(apiCfg.handlerReset))
	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	server.ListenAndServe()
}
