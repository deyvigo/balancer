package main

import (
	"io"
	"log"
	"net/http"
	"sync/atomic"
	"time"
)

var counter uint64

func main() {
	backends := []string{
		"http://localhost:8080",
		"http://localhost:8081",
		"http://localhost:8082",
	}

	// Verificar que los backends respondan
	log.Println("Verificando backends...")
	for _, backend := range backends {
		resp, err := http.Get(backend + "/health")
		if err != nil {
			log.Printf("Backend %s ERROR: %v", backend, err)
		} else {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			log.Printf("Backend %s OK: %s", backend, string(body))
		}
	}

	// Crear handler simple
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Round robin simple
		index := atomic.AddUint64(&counter, 1) % uint64(len(backends))
		backend := backends[index]

		log.Printf("Proxying to: %s", backend)

		// Crear request
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Get(backend + "/api/hello")
		if err != nil {
			log.Printf("Error proxying to %s: %v", backend, err)
			http.Error(w, "Service Error", 500)
			return
		}
		defer resp.Body.Close()

		// Copiar respuesta
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Error reading response: %v", err)
			http.Error(w, "Service Error", 500)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
		log.Printf("Response from %s: %s", backend, string(body))
	})

	log.Println("Test load balancer starting on :8089")
	log.Fatal(http.ListenAndServe(":8089", nil))
}
