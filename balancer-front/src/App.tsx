import { useEffect, useState } from "react"
import { CardReplic, type ReplicInfo } from "./components/CardReplic"
import { connectWebSocket } from "./service/WebSocketService"

function App() {
  const [replics, setReplics] = useState<ReplicInfo[]>([])

  useEffect(() => {
    const socket = connectWebSocket((data) => {
      if (Array.isArray(data)) {
        setReplics(data)
      } else {
        setReplics((prev) => {
          const idx = prev.findIndex((r) => r.id === data.id)
          if (idx !== -1) {
            const updated = [...prev]
            updated[idx] = data
            return updated
          }
          return [...prev, data]
        })
      }
    })

    return () => socket.close()
  }, [])
  return (
    <>
      <div className="w-auto h-auto flex gap-4">
        {
          replics.map(({ id, ema_ms, error_rate, alive, url }) => <CardReplic key={id} id={id} ema_ms={ema_ms} error_rate={error_rate} alive={alive} url={url} />)
        }
        {/* <CardReplic id={0} ema_ms={0} error_rate={0} alive={true} url="http://localhost:9000" />
        <CardReplic id={1} ema_ms={0} error_rate={0} alive={false} url="http://localhost:9001" /> */}
      </div>
    </>
  )
}

export default App
