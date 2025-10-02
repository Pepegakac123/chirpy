package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Pepegakac123/chirpy/internal/auth"
	"github.com/Pepegakac123/chirpy/internal/database"
	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Email        string    `json:"email"`
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
}

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserId    uuid.UUID `json:"user_id"`
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

func (c *apiConfig) handlerLogin(w http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}
	const defaultExpirationTime = time.Hour

	var params parameters
	w.Header().Set("Content-Type", "application/json")
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}

	user, err := c.db.GetUserByEmail(req.Context(), params.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondWithError(w, 401, "Incorrect email or password")
			return
		}
		respondWithError(w, 500, "Database error")
		return
	}
	IsPwdOk, err := auth.CheckPasswordHash(params.Password, user.HashedPassword)
	if err != nil {
		respondWithError(w, 500, "Server Error")
		return
	}
	if !IsPwdOk {
		respondWithError(w, 401, "Incorrect email or password")
		return
	}

	token, err := auth.MakeJWT(user.ID, c.token, defaultExpirationTime)
	if err != nil {
		respondWithError(w, 500, "Failed to create token")
		return
	}

	refreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		respondWithError(w, 500, "Failed to create refresh token")
		return
	}

	_, err = c.db.CreateRefreshToken(req.Context(), database.CreateRefreshTokenParams{Token: refreshToken, UserID: user.ID, ExpiresAt: time.Now().UTC().AddDate(0, 0, 60)})
	if err != nil {
		respondWithError(w, 500, "Failed to create refresh token")
		return
	}
	respondWithJSON(w, 200, User{
		ID:           user.ID,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
		Email:        user.Email,
		Token:        token,
		RefreshToken: refreshToken,
	})

}
func (c *apiConfig) handlerRefresh(w http.ResponseWriter, req *http.Request) {
	refreshToken, err := auth.GetBearerToken(req.Header)
	if err != nil {
		respondWithError(w, 401, "Unauthorized")
		return
	}
	user, err := c.db.GetUserFromRefreshToken(req.Context(), refreshToken)
	if err != nil {
		respondWithError(w, 401, "Invalid refresh token")
		return
	}

	accessToken, err := auth.MakeJWT(user.ID, c.token, time.Hour)
	if err != nil {
		respondWithError(w, 500, "Failed to create token")
		return
	}
	respondWithJSON(w, 200, map[string]string{
		"token": accessToken,
	})

}
func (c *apiConfig) handlerRevoke(w http.ResponseWriter, req *http.Request) {
	refreshToken, err := auth.GetBearerToken(req.Header)
	if err != nil {
		respondWithError(w, 401, "Unauthorized")
		return
	}
	err = c.db.RevokeToken(req.Context(), refreshToken)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondWithError(w, 401, "Invalid refresh token")
			return
		}
		respondWithError(w, 500, "Failed to revoke token")
		return
	}
	w.WriteHeader(204)
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
	err = c.db.DeleteAllChirps(req.Context())
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("%v\n", err))
		return
	}
	w.Write([]byte("OK"))
}
func (c *apiConfig) handlerCreateChirps(w http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}
	var params parameters
	w.Header().Set("Content-Type", "application/json")
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}

	bearerToken, err := auth.GetBearerToken(req.Header)
	if err != nil {
		respondWithError(w, 401, "Unauthorized")
		return
	}
	userId, err := auth.ValidateJWT(bearerToken, c.token)
	if err != nil {
		respondWithError(w, 401, "Unauthorized")
		return
	}
	cleanedBody, err := validateChirp(params.Body)
	if err != nil {
		respondWithError(w, 400, err.Error())
		return
	}
	arg := database.CreateChirpParams{
		Body:   cleanedBody,
		UserID: userId,
	}
	chirp, err := c.db.CreateChirp(req.Context(), arg)
	if err != nil {
		respondWithError(w, 500, "Something went wrong creating chirp")
		return
	}
	respondWithJSON(w, 201, Chirp{ID: chirp.ID, CreatedAt: chirp.CreatedAt, UpdatedAt: chirp.UpdatedAt, Body: chirp.Body, UserId: chirp.UserID})
}

func (c *apiConfig) handlerGetChirps(w http.ResponseWriter, req *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	chirp, err := c.db.GetAllChirps(req.Context())
	if err != nil {
		respondWithError(w, 500, "Something went wrong creating chirp")
		return
	}
	respondWithJSON(w, 200, chirp)
}
func (c *apiConfig) handlerGetSingleChirp(w http.ResponseWriter, req *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	chirpIDString := req.PathValue("chirpID")
	chirpID, err := uuid.Parse(chirpIDString)
	if err != nil {
		respondWithError(w, 400, "Invalid chirp ID")
		return
	}
	chirp, err := c.db.GetSingleChirp(req.Context(), chirpID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondWithError(w, 404, "Chirp not found")
			return
		}
		respondWithError(w, 500, "Database error")
		return
	}
	respondWithJSON(w, 200, chirp)
}

func (c *apiConfig) handlerUsers(w http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}
	var params parameters
	w.Header().Set("Content-Type", "application/json")
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}
	hashedPwd, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}
	user, err := c.db.CreateUser(req.Context(), database.CreateUserParams{Email: params.Email, HashedPassword: hashedPwd})
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

func validateChirp(body string) (string, error) {
	replacement := "****"
	badWords := map[string]string{
		"kerfuffle": replacement,
		"sharbert":  replacement,
		"fornax":    replacement,
	}
	if len(body) > 140 {
		// respondWithError(w, 400, "Chirp is too long")
		return "", fmt.Errorf("Chirp is too long")
	}
	cleanedBody := replaceBadWords(body, badWords)

	return cleanedBody, nil
}
