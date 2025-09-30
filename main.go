package main

import (
	"fmt"
	"net/http"
)

func main() {
	const port string = "8080"
	mux := http.NewServeMux()
	mux.Handle("/app/", http.StripPrefix("/app", http.FileServer(http.Dir("."))))

	mux.HandleFunc("/healthz", checkHealth)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}
	fmt.Printf("Running Server\n")
	err := srv.ListenAndServe()
	if err != nil {
		fmt.Printf("An error ocured %v", err)
	}

}

func checkHealth(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}
