package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/Pepegakac123/chirpy/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileServerHits atomic.Int32
	db             *database.Queries
	platform       string
}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Println(err)
		return
	}
	dbQueries := database.New(db)
	apiCfg := apiConfig{fileServerHits: atomic.Int32{}, db: dbQueries, platform: os.Getenv("PLATFORM")}
	const port string = "8080"
	mux := http.NewServeMux()
	handleRouting(mux, &apiCfg)
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}
	fmt.Printf("Running Server\n")
	err = srv.ListenAndServe()
	if err != nil {
		fmt.Printf("An error ocured %v", err)
	}

}

func handleRouting(mux *http.ServeMux, apiCfg *apiConfig) {
	mux.Handle("/app/", http.StripPrefix("/app", apiCfg.middlewareMetricsInc(http.FileServer(http.Dir(".")))))
	mux.HandleFunc("GET /api/healthz", checkHealth)
	mux.HandleFunc("POST /api/users", apiCfg.handlerUsers)
	mux.HandleFunc("POST /api/chirps", apiCfg.handlerChirps)
	mux.HandleFunc("GET /admin/metrics", apiCfg.handlerMetrics)
	mux.HandleFunc("POST /admin/reset", apiCfg.handlerReset)
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileServerHits.Add(1)
		next.ServeHTTP(w, r)
	})
}
