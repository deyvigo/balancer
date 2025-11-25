# Script de Prueba de Rate Limiting
# Envía requests rápidos para activar el rate limiting

Write-Host "⚡ Iniciando prueba de rate limiting..." -ForegroundColor Yellow
Write-Host "Enviando requests rápidos para activar límites" -ForegroundColor Yellow

$successCount = 0
$rateLimitedCount = 0
$errorCount = 0

Write-Host "`n🚀 Fase 1: Requests rápidos (sin delay)" -ForegroundColor Cyan
for ($i = 1; $i -le 30; $i++) {
    try {
        $response = Invoke-WebRequest -Uri "http://localhost:8089/api/hello" -UseBasicParsing -TimeoutSec 5
        if ($response.StatusCode -eq 200) {
            $successCount++
            Write-Host "✅ Request $i - OK" -ForegroundColor Green
        }
    }
    catch {
        if ($_.Exception.Message -like "*429*" -or $_.Exception.Message -like "*rate*") {
            $rateLimitedCount++
            Write-Host "⚠️  Request $i - Rate Limited" -ForegroundColor Yellow
        } else {
            $errorCount++
            Write-Host "❌ Request $i - Error: $($_.Exception.Message)" -ForegroundColor Red
        }
    }
    # Sin delay para activar rate limiting
}

Write-Host "`n⏳ Esperando 5 segundos para que se restablezca el rate limit..." -ForegroundColor Cyan
Start-Sleep -Seconds 5

Write-Host "`n🚀 Fase 2: Requests después del reset" -ForegroundColor Cyan
for ($i = 31; $i -le 40; $i++) {
    try {
        $response = Invoke-WebRequest -Uri "http://localhost:8089/api/hello" -UseBasicParsing -TimeoutSec 5
        if ($response.StatusCode -eq 200) {
            $successCount++
            Write-Host "✅ Request $i - OK" -ForegroundColor Green
        }
    }
    catch {
        if ($_.Exception.Message -like "*429*" -or $_.Exception.Message -like "*rate*") {
            $rateLimitedCount++
            Write-Host "⚠️  Request $i - Rate Limited" -ForegroundColor Yellow
        } else {
            $errorCount++
            Write-Host "❌ Request $i - Error: $($_.Exception.Message)" -ForegroundColor Red
        }
    }
    Start-Sleep -Milliseconds 200
}

Write-Host "`n⚡ Resultados de la Prueba de Rate Limiting:" -ForegroundColor Cyan
Write-Host "   ✅ Exitosos: $successCount" -ForegroundColor Green
Write-Host "   ⚠️  Rate Limited: $rateLimitedCount" -ForegroundColor Yellow
Write-Host "   ❌ Otros Errores: $errorCount" -ForegroundColor Red

# Verificar estado del rate limiter
Write-Host "`n📊 Consultando estado del rate limiter..." -ForegroundColor Cyan
try {
    $rateLimitStatus = Invoke-RestMethod -Uri "http://localhost:9000/api/rate-limit" -Method GET
    Write-Host "   Estado: $($rateLimitStatus | ConvertTo-Json -Depth 3)" -ForegroundColor White
}
catch {
    Write-Host "   Error al consultar estado: $($_.Exception.Message)" -ForegroundColor Red
}
