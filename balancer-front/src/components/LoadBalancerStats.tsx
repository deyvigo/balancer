export interface LoadBalancerStatsData {
  algorithm?: string
  total_requests?: number
  active_backends?: number
  avg_response_time?: number
  requests_per_minute?: number
}

export const LoadBalancerStats = ({ data }: { data: LoadBalancerStatsData | null }) => {
  if (!data) {
    return (
      <div className="bg-white rounded-lg shadow-md p-6">
        <h3 className="text-lg font-semibold text-gray-800 mb-4 flex items-center">
          📊 Load Balancer Stats
        </h3>
        <p className="text-gray-500">Cargando...</p>
      </div>
    )
  }

  return (
    <div className="bg-white rounded-lg shadow-md p-6">
      <h3 className="text-lg font-semibold text-gray-800 mb-4 flex items-center">
        📊 Load Balancer Stats
      </h3>
      
      <div className="space-y-3">
        {data.algorithm && (
          <div className="flex justify-between items-center">
            <span className="text-sm text-gray-600">Algoritmo:</span>
            <span className="text-sm font-medium text-gray-800 capitalize">{data.algorithm.replace('_', ' ')}</span>
          </div>
        )}
        
        {data.active_backends !== undefined && (
          <div className="flex justify-between items-center">
            <span className="text-sm text-gray-600">Backends Activos:</span>
            <span className="text-sm font-medium text-green-600">{data.active_backends}</span>
          </div>
        )}
        
        {data.total_requests !== undefined && (
          <div className="flex justify-between items-center">
            <span className="text-sm text-gray-600">Total Requests:</span>
            <span className="text-sm font-medium text-blue-600">{data.total_requests.toLocaleString()}</span>
          </div>
        )}
        
        {data.avg_response_time !== undefined && (
          <div className="flex justify-between items-center">
            <span className="text-sm text-gray-600">Tiempo Promedio:</span>
            <span className="text-sm font-medium text-purple-600">{data.avg_response_time.toFixed(2)} ms</span>
          </div>
        )}
        
        {data.requests_per_minute !== undefined && (
          <div className="flex justify-between items-center">
            <span className="text-sm text-gray-600">Req/min:</span>
            <span className="text-sm font-medium text-orange-600">{data.requests_per_minute}</span>
          </div>
        )}
        
        {/* Status indicator */}
        <div className="flex justify-between items-center pt-2 border-t">
          <span className="text-sm text-gray-600">Estado:</span>
          <span className="px-2 py-1 rounded-full text-xs font-medium bg-green-100 text-green-800">
            Operacional
          </span>
        </div>
      </div>
    </div>
  )
}