package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func checkHealth(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func (c *apiConfig) handlerMetrics(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(200)
	fmt.Fprintf(w, `<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, c.fileServerHits.Load())
}

func (c *apiConfig) handlerReset(w http.ResponseWriter, req *http.Request) {
	c.fileServerHits.Swap(0)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func handlerValidateChirp(w http.ResponseWriter, req *http.Request) {
	replacement := "****"
	badWords := map[string]string{
		"kerfuffle": replacement,
		"sharbert":  replacement,
		"fornax":    replacement,
	}

	type parameters struct {
		Body string `json:"body"`
	}

	params := parameters{}
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}

	if len(params.Body) > 140 {
		respondWithError(w, 400, "Chirp is too long")
		return
	}
	cleanedBody := replaceBadWords(params.Body, badWords)

	respondWithJSON(w, 200, map[string]string{"cleaned_body": cleanedBody})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	data, err := json.Marshal(payload)
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
	}
	w.WriteHeader(code)
	w.Write(data)
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	respondWithJSON(w, code, map[string]string{"error": msg})
}

func replaceBadWords(msg string, badWords map[string]string) string {
	splitedString := strings.Split(msg, " ")
	for i, word := range splitedString {
		if replace, valid := badWords[strings.ToLower(word)]; valid {
			splitedString[i] = replace
		}
	}
	return strings.Join(splitedString, " ")
}
