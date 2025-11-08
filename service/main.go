package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"time"
)

type HealthStatus struct {
	Status string `json:"status"`
}

func main() {
	rand.Seed(time.Now().UnixNano())

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(HealthStatus{Status: "healthy"})
	})

	http.HandleFunc("/api/hello", func(w http.ResponseWriter, r *http.Request) {
		if rand.Float32() < 0.2 {
			time.Sleep(time.Duration(rand.Intn(2000)+1000) * time.Millisecond)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "Hello from service"}`))
	})
	log.Println("Servicio escuchando en :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
