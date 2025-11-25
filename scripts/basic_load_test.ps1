# Script de Prueba de Carga Básica
# Envía requests simples para probar el balanceador básico

Write-Host "🚀 Iniciando prueba de carga básica..." -ForegroundColor Green
Write-Host "Enviando 50 requests con intervalo de 0.5 segundos" -ForegroundColor Yellow

$successCount = 0
$errorCount = 0
$totalTime = Measure-Command {
    for ($i = 1; $i -le 50; $i++) {
        try {
            $response = Invoke-WebRequest -Uri "http://localhost:8089/api/hello" -UseBasicParsing -TimeoutSec 10
            if ($response.StatusCode -eq 200) {
                $successCount++
                Write-Host "✅ Request $i - OK" -ForegroundColor Green
            }
        }
        catch {
            $errorCount++
            Write-Host "❌ Request $i - Error: $($_.Exception.Message)" -ForegroundColor Red
        }
        Start-Sleep -Milliseconds 500
    }
}

Write-Host "`n📊 Resultados de la Prueba Básica:" -ForegroundColor Cyan
Write-Host "   ✅ Exitosos: $successCount" -ForegroundColor Green
Write-Host "   ❌ Errores: $errorCount" -ForegroundColor Red
Write-Host "   ⏱️  Tiempo total: $($totalTime.TotalSeconds.ToString('F2')) segundos" -ForegroundColor Yellow
Write-Host "   📈 Promedio: $((50 / $totalTime.TotalSeconds).ToString('F2')) req/s" -ForegroundColor Magenta
