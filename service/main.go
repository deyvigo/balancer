package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"
)

type HealthStatus struct {
	Status string `json:"status"`
}

func main() {
	rand.Seed(time.Now().UnixNano())

	port := "8080"
	if len(os.Args) > 1 {
		port = os.Args[1]
	}

	backendID := "service-" + port

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(HealthStatus{Status: "healthy"})
	})

	http.HandleFunc("/api/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Backend", backendID)

		var sleepTime time.Duration
		switch port {
		case "8080":
			if rand.Float32() < 0.3 {
				sleepTime = time.Duration(rand.Intn(40)+10) * time.Millisecond
			}
		case "8081":
			if rand.Float32() < 0.4 {
				sleepTime = time.Duration(rand.Intn(100)+50) * time.Millisecond
			}
		case "8082":
			if rand.Float32() < 0.5 {
				sleepTime = time.Duration(rand.Intn(200)+100) * time.Millisecond
			}
		}

		if sleepTime > 0 {
			time.Sleep(sleepTime)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "Hello from ` + backendID + `"}`))
	})
	log.Printf("Servicio %s escuchando en :%s", backendID, port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
