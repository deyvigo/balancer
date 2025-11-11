export interface ReplicInfo {
  id: number,
  url: string,
  ema_ms: number,
  error_rate: number,
  alive: boolean,
  last_checked: Date
}

export const CardReplic = ({ id, url, ema_ms, error_rate, alive }: Omit<ReplicInfo, "last_checked">) => {
  return (
    <section className="w-[400px] border-2 border-sky-950 rounded-xl p-4 bg-cyan-800 flex flex-col gap-4">
      <h3>
        Réplica <span className="font-bold">{id}</span>
      </h3>
      <div className="flex flex-col">
        <div>
          <p>URL: <span className="italic ">{url}</span></p>
        </div>
        <div className="flex gap-2 justify-between">
          <div>
            <p>EMA: {ema_ms.toFixed(4)}</p>
          </div>
          <div>
            <p>Error rate: {error_rate.toFixed(2)}</p>
          </div>
          <div>
            <p>Alive: <span className={`font-bold ${alive ? "text-green-700" : "text-red-700"}`}>{alive.toString()}</span></p>
          </div>
        </div>
      </div>
    </section>
  )
}