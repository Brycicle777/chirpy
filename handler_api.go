package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

type chirpPost struct {
	Body string `json:"body"`
}

type chirpValidation struct {
	Error string `json:"error"`
	Valid bool   `json:"valid"`
}

func handlerReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func handlerValidateChirp(w http.ResponseWriter, r *http.Request) {
	respBody := chirpValidation{
		Error: "",
		Valid: true,
	}

	decoder := json.NewDecoder(r.Body)
	chirp := chirpPost{}
	err := decoder.Decode(&chirp)
	if err != nil {
		log.Printf("Error decoding chirp: %s", err)
		respBody.Error = "Something went wrong"
		respBody.Valid = false
		w.WriteHeader(500)
		return
	}

	if len(chirp.Body) > 140 {
		respBody.Error = "Chirp is too long"
		respBody.Valid = false
		w.WriteHeader(400)
	} else {
		w.WriteHeader(200)
	}

	resp, err := json.Marshal(respBody)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}
