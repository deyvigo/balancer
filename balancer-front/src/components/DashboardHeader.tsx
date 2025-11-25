import { useEffect, useState } from 'react'

export const DashboardHeader = () => {
  const [currentTime, setCurrentTime] = useState(new Date())

  useEffect(() => {
    const timer = setInterval(() => {
      setCurrentTime(new Date())
    }, 1000)

    return () => clearInterval(timer)
  }, [])

  return (
    <header className="bg-gradient-to-r from-blue-600 to-purple-600 text-white p-6 rounded-lg shadow-lg mb-8">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-4xl font-bold mb-2">Load Balancer Dashboard</h1>
          <p className="text-blue-100">
            Monitoreo en tiempo real del sistema de balanceo de carga empresarial
          </p>
        </div>
        <div className="text-right">
          <div className="text-sm text-blue-200">Última actualización</div>
          <div className="text-xl font-mono">
            {currentTime.toLocaleTimeString()}
          </div>
          <div className="text-sm text-blue-200">
            {currentTime.toLocaleDateString()}
          </div>
        </div>
      </div>
      
      {/* Status indicators */}
      <div className="mt-4 flex gap-4">
        <div className="flex items-center gap-2">
          <div className="w-3 h-3 bg-green-400 rounded-full animate-pulse"></div>
          <span className="text-sm">Sistema Operacional</span>
        </div>
        <div className="flex items-center gap-2">
          <div className="w-3 h-3 bg-blue-400 rounded-full"></div>
          <span className="text-sm">Métricas WebSocket</span>
        </div>
        <div className="flex items-center gap-2">
          <div className="w-3 h-3 bg-purple-400 rounded-full"></div>
          <span className="text-sm">Admin API</span>
        </div>
      </div>
    </header>
  )
}