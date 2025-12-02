export interface CircuitBreakerData {
  [backend: string]: {
    state: string
    failure_count: number
    error_rate: number
    last_failure_time?: string
    next_attempt?: string
  }
}

export const CircuitBreakerDashboard = ({ data }: { data: CircuitBreakerData | null }) => {
  if (!data) {
    return (
      <div className="bg-white rounded-lg shadow-md p-6">
        <h3 className="text-lg font-semibold text-gray-800 mb-4 flex items-center">
          🛡️ Circuit Breakers
        </h3>
        <p className="text-gray-500">Cargando...</p>
      </div>
    )
  }

  const backends = Object.entries(data)

  const getStateColor = (state: string) => {
    switch (state.toLowerCase()) {
      case 'closed': return 'bg-green-100 text-green-800'
      case 'open': return 'bg-red-100 text-red-800'
      case 'half_open': return 'bg-yellow-100 text-yellow-800'
      default: return 'bg-gray-100 text-gray-800'
    }
  }

  return (
    <div className="bg-white rounded-lg shadow-md p-6">
      <h3 className="text-lg font-semibold text-gray-800 mb-4 flex items-center">
        🛡️ Circuit Breakers
      </h3>
      
      <div className="space-y-4 max-h-64 overflow-y-auto">
        {backends.map(([backend, info]) => {
          const backendName = backend.replace('http://localhost:', ':')
          
          return (
            <div key={backend} className="border rounded-lg p-3 bg-gray-50">
              <div className="flex justify-between items-center mb-2">
                <span className="text-sm font-medium text-gray-800">{backendName}</span>
                <span className={`px-2 py-1 rounded-full text-xs font-medium ${getStateColor(info.state)}`}>
                  {info.state.toUpperCase()}
                </span>
              </div>
              
              <div className="grid grid-cols-2 gap-2 text-xs">
                <div>
                  <span className="text-gray-600">Failures:</span>
                  <span className="ml-1 font-medium">{info.failure_count}</span>
                </div>
                <div>
                  <span className="text-gray-600">Error Rate:</span>
                  <span className="ml-1 font-medium">{(info.error_rate * 100).toFixed(1)}%</span>
                </div>
              </div>
            </div>
          )
        })}
        
        {backends.length === 0 && (
          <p className="text-gray-500 text-sm">No hay circuit breakers configurados</p>
        )}
      </div>
    </div>
  )
}