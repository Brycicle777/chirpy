package main

import (
	"database/sql"
	"encoding/json"
	"internal/auth"
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
	jwtSecret      string
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
	Token     string    `json:"token"`
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

type userRequest struct {
	Email            string `json:"email"`
	Password         string `json:"password"`
	ExpiresInSeconds int    `json:"expires_in_seconds"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func handlerReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	newUserReq := userRequest{}
	err := decoder.Decode(&newUserReq)
	if err != nil {
		log.Printf("Error parsing request: %s", err)
		respondWithError(w, 500, "Something went wrong")
		return
	}

	hashedPassword, err := auth.HashPassword(newUserReq.Password)
	if err != nil || len(hashedPassword) == 0 {
		log.Printf("Error hashing password: %s", err)
		respondWithError(w, 500, "Something went wrong")
		return
	}

	user, err := cfg.db.CreateUser(r.Context(), database.CreateUserParams{
		Email:          newUserReq.Email,
		HashedPassword: hashedPassword,
	})
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

func (cfg *apiConfig) handlerLoginUser(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	userReq := userRequest{}
	err := decoder.Decode(&userReq)
	if err != nil {
		log.Printf("Error parsing request: %s", err)
		respondWithError(w, 500, "Something went wrong")
		return
	}

	user, err := cfg.db.GetUserFromEmail(r.Context(), userReq.Email)
	if err != nil {
		log.Printf("Error logging in: %s", err)
		respondWithError(w, 401, "Incorrect email or password")
		return
	}

	err = auth.CheckPasswordHash(userReq.Password, user.HashedPassword)
	if err != nil {
		log.Printf("Error logging in: %s", err)
		respondWithError(w, 401, "Incorrect email or password")
		return
	}

	expiresInSec := time.Duration(3600) * time.Second
	if userReq.ExpiresInSeconds > 0 && userReq.ExpiresInSeconds < 3600 {
		expiresInSec = time.Duration(userReq.ExpiresInSeconds) * time.Second
	}

	token, err := auth.MakeJWT(user.ID, cfg.jwtSecret, expiresInSec)
	if err != nil {
		log.Printf("Error creating token: %s", err)
		respondWithError(w, 500, "Something went wrong")
		return
	}

	respondWithJSON(w, 200, User{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
		Token:     token,
	})
}

func (cfg *apiConfig) handlerPostChirp(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	chirp := chirpPost{}
	err := decoder.Decode(&chirp)
	if err != nil {
		log.Printf("Error decoding chirp: %s", err)
		respondWithError(w, 500, "Something went wrong")
		return
	}
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("Error procesing header: %s", err)
		respondWithError(w, 500, "Something went wrong")
		return
	}
	userId, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		log.Printf("Invalid token: %s", err)
		respondWithError(w, 401, "Unauthorized")
		return
	}

	if len(chirp.Body) > 140 {
		respondWithError(w, 400, "Chirp is too long")
		return
	}

	newChirp, err := cfg.db.CreateChirp(r.Context(), database.CreateChirpParams{
		Body:   replaceProfanity(chirp.Body),
		UserID: userId,
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

func (cfg *apiConfig) handlerGetChirps(w http.ResponseWriter, r *http.Request) {
	chirpResponses, err := cfg.db.GetChirps(r.Context())
	if err != nil {
		log.Printf("Error retrieving chirp: %s", err)
		respondWithError(w, 500, "Something went wrong")
		return
	}
	var chirps []chirpResponse
	for _, chirp := range chirpResponses {
		chirps = append(chirps, chirpResponse{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID,
		})
	}
	respondWithJSON(w, 200, chirps)
}

func (cfg *apiConfig) handlerGetChirp(w http.ResponseWriter, r *http.Request) {
	chirp_id, err := uuid.Parse(r.PathValue("chirpID"))
	if err != nil {
		log.Printf("Error parsing provided chirp id: %s", err)
		respondWithError(w, 500, "Something went wrong")
		return
	}
	chirp, err := cfg.db.GetChirp(r.Context(), chirp_id)
	if err == sql.ErrNoRows {
		log.Printf("Chirp does not exist: %s", err)
		respondWithError(w, 404, "The requested chirp was not found")
		return
	}
	if err != nil {
		log.Printf("Error retrieving chirp: %s", err)
		respondWithError(w, 500, "Something went wrong")
		return
	}

	respondWithJSON(w, 200, chirpResponse{
		ID:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserID:    chirp.UserID,
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
