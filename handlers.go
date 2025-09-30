package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

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
	if c.platform != "dev" {
		respondWithError(w, 403, "You can not reset user anywhere else than in dev PLATFORM")
		return
	}
	c.fileServerHits.Swap(0)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	err := c.db.DeleteAllUsers(req.Context())
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("%v\n", err))
		return
	}
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

func (c *apiConfig) handlerUsers(w http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Email string `json:"email"`
	}
	var params parameters
	w.Header().Set("Content-Type", "application/json")
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}

	user, err := c.db.CreateUser(req.Context(), params.Email)
	if err != nil {
		fmt.Println(err)
		respondWithError(w, 500, "Something went wrong whe connecting to the database")
		return
	}
	respondWithJSON(w, 201, User{ID: user.ID, CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt, Email: user.Email})

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
