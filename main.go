package main

import (
	"database/sql"
	"internal/database"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")
	jwtSecret := os.Getenv("JWT_SECRET")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	dbQueries := database.New(db)

	mux := http.NewServeMux()
	apiCfg := apiConfig{
		platform:  platform,
		db:        dbQueries,
		jwtSecret: jwtSecret,
	}
	handlerApp := http.FileServer(http.Dir("."))
	handlerApp = http.StripPrefix("/app", handlerApp)
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(handlerApp))
	mux.Handle("GET /api/healthz", http.HandlerFunc(handlerReadiness))
	mux.Handle("GET /api/chirps", http.HandlerFunc(apiCfg.handlerGetChirps))
	mux.Handle("GET /api/chirps/{chirpID}", http.HandlerFunc(apiCfg.handlerGetChirp))
	mux.Handle("DELETE /api/chirps/{chirpID}", http.HandlerFunc(apiCfg.handlerDeleteChirp))
	mux.Handle("POST /api/chirps", http.HandlerFunc(apiCfg.handlerPostChirp))
	mux.Handle("POST /api/users", http.HandlerFunc(apiCfg.handlerCreateUser))
	mux.Handle("PUT /api/users", http.HandlerFunc(apiCfg.handlerUpdateUser))
	mux.Handle("POST /api/login", http.HandlerFunc(apiCfg.handlerLoginUser))
	mux.Handle("POST /api/refresh", http.HandlerFunc(apiCfg.handlerRefresh))
	mux.Handle("POST /api/revoke", http.HandlerFunc(apiCfg.handlerRevoke))
	mux.Handle("GET /admin/metrics", http.HandlerFunc(apiCfg.handlerMetrics))
	mux.Handle("POST /admin/reset", http.HandlerFunc(apiCfg.handlerReset))
	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	server.ListenAndServe()
}
