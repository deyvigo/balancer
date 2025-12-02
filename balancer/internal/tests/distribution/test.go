package main

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

const (
	TargetURL   = "http://localhost:9000/api/hello"
	NumRequests = 100
)

var (
	requestCount  uint64
	successCount  uint64
	backendCounts = make(map[string]uint64)
	mu            sync.Mutex
)

func main() {
	fmt.Printf("Enviando %d peticiones para probar distribución...\n", NumRequests)

	var wg sync.WaitGroup

	// Enviar peticiones concurrentes
	for i := 0; i < NumRequests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			resp, err := http.Get(TargetURL)
			atomic.AddUint64(&requestCount, 1)

			if err != nil {
				fmt.Printf("Error en petición %d: %v\n", id, err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == 200 {
				atomic.AddUint64(&successCount, 1)

				// Contar qué backend respondió (si hay header identificativo)
				if backend := resp.Header.Get("X-Backend"); backend != "" {
					mu.Lock()
					backendCounts[backend]++
					mu.Unlock()
				}
			}
		}(i)

		// Pequeña pausa para no saturar
		time.Sleep(10 * time.Millisecond)
	}

	wg.Wait()

	fmt.Printf("\n=== Resultados ===\n")
	fmt.Printf("Peticiones enviadas: %d\n", requestCount)
	fmt.Printf("Peticiones exitosas: %d\n", successCount)
	fmt.Printf("Tasa de éxito: %.2f%%\n", float64(successCount)/float64(requestCount)*100)

	fmt.Printf("\nDistribución por backend:\n")
	if len(backendCounts) == 0 {
		fmt.Printf("  (No se recibieron headers X-Backend de los backends)\n")
		fmt.Printf("  Esto es normal si los backends no envían headers identificativos\n")
		fmt.Printf("  Para ver la distribución, reinicia los contenedores con el código actualizado\n")
	} else {
		for backend, count := range backendCounts {
			fmt.Printf("  %s: %d peticiones (%.1f%%)\n", backend, count, float64(count)/float64(successCount)*100)
		}
	}
}
