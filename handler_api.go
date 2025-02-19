package main

import (
	"encoding/json"
	"internal/database"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	platform       string
	db             *database.Queries
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

type chirpPost struct {
	Body   string    `json:"body"`
	UserID uuid.UUID `json:"user_id"`
}

type chirpResponse struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

type createUserRequest struct {
	Email string `json:"email"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func handlerReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) handlerChirps(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	chirp := chirpPost{}
	err := decoder.Decode(&chirp)
	if err != nil {
		log.Printf("Error decoding chirp: %s", err)
		respondWithError(w, 500, "Something went wrong")
		return
	}

	if len(chirp.Body) > 140 {
		respondWithError(w, 400, "Chirp is too long")
		return
	}

	newChirp, err := cfg.db.CreateChirp(r.Context(), database.CreateChirpParams{
		Body:   replaceProfanity(chirp.Body),
		UserID: chirp.UserID,
	})
	if err != nil {
		log.Printf("Error creating chirp: %s", err)
		respondWithError(w, 500, "Something went wrong")
		return
	}
	respondWithJSON(w, 201, chirpResponse{
		ID:        newChirp.ID,
		CreatedAt: newChirp.CreatedAt,
		UpdatedAt: newChirp.UpdatedAt,
		Body:      newChirp.Body,
		UserID:    newChirp.UserID,
	})
}

func (cfg *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	newUserReq := createUserRequest{}
	err := decoder.Decode(&newUserReq)
	if err != nil {
		log.Printf("Error parsing request: %s", err)
		respondWithError(w, 500, "Something went wrong")
		return
	}

	user, err := cfg.db.CreateUser(r.Context(), newUserReq.Email)
	if err != nil {
		log.Printf("Error creating user: %v", err)
		respondWithError(w, 500, "Error creating user")
		return
	}

	respondWithJSON(w, 201, User{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
	})
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	respondWithJSON(w, code, errorResponse{
		Error: msg,
	})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}

func replaceProfanity(body string) string {
	words := strings.Split(body, " ")
	for i := range words {
		switch strings.ToLower(words[i]) {
		case
			"kerfuffle",
			"sharbert",
			"fornax":
			words[i] = "****"
		}
	}
	return strings.Join(words, " ")
}
