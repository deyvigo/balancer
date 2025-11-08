package web

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/deyvigo/balanceador/balancer/internal/monitor"
	"github.com/gorilla/websocket"
)

type WebSocketServer struct {
	Monitor *monitor.MonitorService
}

func (ws *WebSocketServer) MetricsHandler(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[websocket] error al abrir conexión: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("[websocket] cliente conectado al /metrics/ws")

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			log.Println("[websocket] cliente desconectado")
			return
		case <-ticker.C:
			metrics := ws.Monitor.SnapshotMetrics()
			data, _ := json.Marshal(metrics)

			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Println("[websocket] error al escribir mensaje:", err)
				return
			}
		}
	}
}
