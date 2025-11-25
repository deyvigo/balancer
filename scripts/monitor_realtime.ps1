# Script de Monitoreo en Tiempo Real
# Ejecuta requests continuos mientras muestra métricas en vivo

Write-Host "📊 MONITOR EN TIEMPO REAL - Load Balancer" -ForegroundColor Magenta
Write-Host "===========================================" -ForegroundColor Magenta
Write-Host "Presiona Ctrl+C para detener el monitoreo" -ForegroundColor Yellow

$requestCount = 0
$successCount = 0
$errorCount = 0
$startTime = Get-Date

# Función para mostrar estadísticas
function Show-Stats {
    param($requestCount, $successCount, $errorCount, $startTime)
    
    $currentTime = Get-Date
    $elapsed = ($currentTime - $startTime).TotalSeconds
    $rps = if ($elapsed -gt 0) { [math]::Round($requestCount / $elapsed, 2) } else { 0 }
    $successRate = if ($requestCount -gt 0) { [math]::Round(($successCount / $requestCount) * 100, 1) } else { 0 }
    
    Write-Host "`r📈 Requests: $requestCount | ✅ Éxitos: $successCount | ❌ Errores: $errorCount | 📊 RPS: $rps | 💯 Éxito: $successRate%" -NoNewline -ForegroundColor Cyan
}

# Función para obtener métricas del API
function Get-LoadBalancerMetrics {
    try {
        $metrics = Invoke-RestMethod -Uri "http://localhost:9000/api/metrics" -Method GET -TimeoutSec 2
        $rateLimit = Invoke-RestMethod -Uri "http://localhost:9000/api/rate-limit" -Method GET -TimeoutSec 2
        $circuitBreaker = Invoke-RestMethod -Uri "http://localhost:9000/api/circuit-breaker" -Method GET -TimeoutSec 2
        
        return @{
            Metrics = $metrics
            RateLimit = $rateLimit  
            CircuitBreaker = $circuitBreaker
        }
    }
    catch {
        return $null
    }
}

# Bucle principal de monitoreo
try {
    while ($true) {
        $requestCount++
        
        try {
            # Hacer request al load balancer
            $response = Invoke-WebRequest -Uri "http://localhost:8089/api/hello" -UseBasicParsing -TimeoutSec 3
            if ($response.StatusCode -eq 200) {
                $successCount++
            }
        }
        catch {
            $errorCount++
        }
        
        # Mostrar estadísticas cada request
        Show-Stats $requestCount $successCount $errorCount $startTime
        
        # Cada 10 requests, mostrar métricas detalladas
        if ($requestCount % 10 -eq 0) {
            Write-Host ""
            Write-Host "`n🔍 Métricas Detalladas (Request #$requestCount):" -ForegroundColor Green
            
            $apiMetrics = Get-LoadBalancerMetrics
            if ($apiMetrics) {
                # Mostrar métricas principales
                if ($apiMetrics.Metrics.data) {
                    $data = $apiMetrics.Metrics.data
                    Write-Host "   🎯 Algoritmo: $($data.algorithm)" -ForegroundColor White
                    Write-Host "   📊 Total Requests: $($data.total_requests)" -ForegroundColor White
                    Write-Host "   🖥️  Backends Activos: $($data.active_backends)" -ForegroundColor White
                    Write-Host "   ⏱️  Tiempo Promedio: $($data.avg_response_time)ms" -ForegroundColor White
                }
                
                # Rate Limiting
                if ($apiMetrics.RateLimit.data) {
                    $rl = $apiMetrics.RateLimit.data
                    $status = if ($rl.enabled) { "✅ Activo" } else { "❌ Inactivo" }
                    Write-Host "   ⚡ Rate Limit: $status ($($rl.type))" -ForegroundColor Yellow
                    if ($rl.global_tokens) {
                        Write-Host "   🪙 Tokens Globales: $($rl.global_tokens)" -ForegroundColor Yellow
                    }
                }
                
                # Circuit Breakers
                if ($apiMetrics.CircuitBreaker.data) {
                    $cb = $apiMetrics.CircuitBreaker.data
                    foreach ($backend in $cb.PSObject.Properties) {
                        $state = $backend.Value.state
                        $color = switch ($state) {
                            "CLOSED" { "Green" }
                            "HALF_OPEN" { "Yellow" }
                            "OPEN" { "Red" }
                            default { "White" }
                        }
                        Write-Host "   🛡️  $($backend.Name): $state (Errores: $($backend.Value.failure_count))" -ForegroundColor $color
                    }
                }
            }
            Write-Host ""
        }
        
        Start-Sleep -Milliseconds 500
    }
}
catch {
    Write-Host "`n`n⏹️  Monitoreo detenido por el usuario" -ForegroundColor Yellow
}
finally {
    # Resumen final
    $finalTime = Get-Date
    $totalTime = ($finalTime - $startTime).TotalSeconds
    
    Write-Host "`n`n📋 RESUMEN FINAL:" -ForegroundColor Magenta
    Write-Host "   ⏱️  Duración: $([math]::Round($totalTime, 1)) segundos" -ForegroundColor White
    Write-Host "   📊 Total Requests: $requestCount" -ForegroundColor White
    Write-Host "   ✅ Exitosos: $successCount" -ForegroundColor Green
    Write-Host "   ❌ Errores: $errorCount" -ForegroundColor Red
    Write-Host "   📈 Promedio RPS: $([math]::Round($requestCount / $totalTime, 2))" -ForegroundColor Cyan
    Write-Host "   💯 Tasa Éxito: $([math]::Round(($successCount / $requestCount) * 100, 1))%" -ForegroundColor Magenta
}