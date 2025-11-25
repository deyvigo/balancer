# Script de Prueba de Load Balancing
# Verifica la distribución de carga entre backends

Write-Host "⚖️ Iniciando prueba de distribución de carga..." -ForegroundColor Magenta
Write-Host "Verificando distribución entre backends" -ForegroundColor Yellow

# Contador de respuestas por backend
$backendCounts = @{}
$totalRequests = 60

Write-Host "`n🚀 Enviando $totalRequests requests para analizar distribución..." -ForegroundColor Green

for ($i = 1; $i -le $totalRequests; $i++) {
    try {
        $response = Invoke-WebRequest -Uri "http://localhost:8089/api/hello" -UseBasicParsing -TimeoutSec 5
        
        # Intentar extraer información del backend de la respuesta
        $content = $response.Content
        if ($content -match "Backend|Port|Server") {
            # Buscar patrones que indiquen el backend
            if ($content -match "808(\d)") {
                $backend = "Backend-$($Matches[1])"
            } elseif ($content -match "service(\d)") {
                $backend = "Service-$($Matches[1])"
            } else {
                $backend = "Unknown"
            }
        } else {
            # Si no se puede determinar, usar el hash del contenido
            $backend = "Response-" + ($content.GetHashCode() % 3)
        }
        
        if ($backendCounts.ContainsKey($backend)) {
            $backendCounts[$backend]++
        } else {
            $backendCounts[$backend] = 1
        }
        
        Write-Host "✅ Request $i - Backend: $backend" -ForegroundColor Green
    }
    catch {
        Write-Host "❌ Request $i - Error: $($_.Exception.Message)" -ForegroundColor Red
    }
    Start-Sleep -Milliseconds 100
}

Write-Host "`n⚖️ Resultados de Distribución de Carga:" -ForegroundColor Cyan
Write-Host "   Total de requests: $totalRequests" -ForegroundColor White
Write-Host "   Distribución por backend:" -ForegroundColor Yellow

$sortedBackends = $backendCounts.GetEnumerator() | Sort-Object Key
foreach ($backend in $sortedBackends) {
    $percentage = [math]::Round(($backend.Value / $totalRequests) * 100, 1)
    $bar = "█" * [math]::Floor($percentage / 3)
    Write-Host "     $($backend.Key): $($backend.Value) requests ($percentage%) $bar" -ForegroundColor Green
}

# Calcular desviación estándar para evaluar balance
$values = $backendCounts.Values
if ($values.Count -gt 1) {
    $mean = ($values | Measure-Object -Average).Average
    $variance = ($values | ForEach-Object { [math]::Pow($_ - $mean, 2) } | Measure-Object -Sum).Sum / $values.Count
    $stdDev = [math]::Sqrt($variance)
    $coefficientOfVariation = ($stdDev / $mean) * 100
    
    Write-Host "`n📊 Análisis de Balance:" -ForegroundColor Cyan
    Write-Host "   Promedio por backend: $([math]::Round($mean, 1))" -ForegroundColor White
    Write-Host "   Desviación estándar: $([math]::Round($stdDev, 1))" -ForegroundColor White
    Write-Host "   Coeficiente de variación: $([math]::Round($coefficientOfVariation, 1))%" -ForegroundColor White
    
    if ($coefficientOfVariation -lt 20) {
        Write-Host "   ✅ Balance EXCELENTE (CV < 20%)" -ForegroundColor Green
    } elseif ($coefficientOfVariation -lt 40) {
        Write-Host "   ⚠️  Balance BUENO (CV < 40%)" -ForegroundColor Yellow
    } else {
        Write-Host "   ❌ Balance POBRE (CV > 40%)" -ForegroundColor Red
    }
}

# Consultar métricas del balanceador
Write-Host "`n📈 Métricas del Load Balancer:" -ForegroundColor Cyan
try {
    $metrics = Invoke-RestMethod -Uri "http://localhost:9000/api/metrics" -Method GET
    Write-Host "$($metrics | ConvertTo-Json -Depth 3)" -ForegroundColor White
}
catch {
    Write-Host "Error al consultar métricas: $($_.Exception.Message)" -ForegroundColor Red
}
