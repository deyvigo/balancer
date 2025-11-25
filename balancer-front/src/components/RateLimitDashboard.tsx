export interface RateLimitData {
  enabled: boolean
  type: string
  global_limit: number
  per_ip_limit: number
  active_ips: number
  global_tokens?: number
}

export const RateLimitDashboard = ({ data }: { data: RateLimitData | null }) => {
  if (!data) {
    return (
      <div className="bg-white rounded-lg shadow-md p-6">
        <h3 className="text-lg font-semibold text-gray-800 mb-4 flex items-center">
          ⚡ Rate Limiting
        </h3>
        <p className="text-gray-500">Cargando...</p>
      </div>
    )
  }

  return (
    <div className="bg-white rounded-lg shadow-md p-6">
      <h3 className="text-lg font-semibold text-gray-800 mb-4 flex items-center">
        ⚡ Rate Limiting
      </h3>
      
      <div className="space-y-3">
        <div className="flex justify-between items-center">
          <span className="text-sm text-gray-600">Estado:</span>
          <span className={`px-2 py-1 rounded-full text-xs font-medium ${
            data.enabled ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'
          }`}>
            {data.enabled ? 'Activo' : 'Inactivo'}
          </span>
        </div>
        
        <div className="flex justify-between items-center">
          <span className="text-sm text-gray-600">Tipo:</span>
          <span className="text-sm font-medium text-gray-800">{data.type}</span>
        </div>
        
        <div className="flex justify-between items-center">
          <span className="text-sm text-gray-600">Límite Global:</span>
          <span className="text-sm font-medium text-gray-800">{data.global_limit}</span>
        </div>
        
        <div className="flex justify-between items-center">
          <span className="text-sm text-gray-600">Límite por IP:</span>
          <span className="text-sm font-medium text-gray-800">{data.per_ip_limit}</span>
        </div>
        
        <div className="flex justify-between items-center">
          <span className="text-sm text-gray-600">IPs Activas:</span>
          <span className="text-sm font-medium text-blue-600">{data.active_ips}</span>
        </div>
        
        {data.global_tokens !== undefined && (
          <div className="flex justify-between items-center">
            <span className="text-sm text-gray-600">Tokens Globales:</span>
            <span className="text-sm font-medium text-green-600">{data.global_tokens}</span>
          </div>
        )}
      </div>
    </div>
  )
}