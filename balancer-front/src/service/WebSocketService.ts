export const connectWebSocket = (onMessage: (data: any) => void) => {
  const socket = new WebSocket("ws://localhost:9000/metrics/ws")

  socket.onopen = () => console.log("✅ WebSocket conectado");
  socket.onclose = () => console.log("🔌 WebSocket cerrado");
  socket.onerror = (err) => console.error("❌ Error WS:", err);

  socket.onmessage = (event) => {
    try {
      const data = JSON.parse(event.data);
      onMessage(data);
    } catch (e) {
      console.error("❗ Error al parsear mensaje:", e);
    }
  };

  return socket;
}