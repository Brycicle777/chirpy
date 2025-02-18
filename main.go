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
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	dbQueries := database.New(db)

	mux := http.NewServeMux()
	apiCfg := apiConfig{
		platform: platform,
		db:       dbQueries,
	}
	handlerApp := http.FileServer(http.Dir("."))
	handlerApp = http.StripPrefix("/app", handlerApp)
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(handlerApp))
	mux.Handle("GET /api/healthz", http.HandlerFunc(handlerReadiness))
	mux.Handle("POST /api/validate_chirp", http.HandlerFunc(handlerValidateChirp))
	mux.Handle("POST /api/users", http.HandlerFunc(apiCfg.handlerCreateUser))
	mux.Handle("GET /admin/metrics", http.HandlerFunc(apiCfg.handlerMetrics))
	mux.Handle("POST /admin/reset", http.HandlerFunc(apiCfg.handlerReset))
	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	server.ListenAndServe()
}
