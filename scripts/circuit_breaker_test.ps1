# Script de Prueba de Circuit Breaker
# Simula fallos para activar los circuit breakers

Write-Host "🛡️ Iniciando prueba de circuit breakers..." -ForegroundColor Blue
Write-Host "Simulando fallos para activar protecciones" -ForegroundColor Yellow

# Primero verificar estado inicial
Write-Host "`n📊 Estado inicial de circuit breakers:" -ForegroundColor Cyan
try {
    $initialStatus = Invoke-RestMethod -Uri "http://localhost:9000/api/circuit-breaker" -Method GET
    Write-Host "$($initialStatus | ConvertTo-Json -Depth 3)" -ForegroundColor White
}
catch {
    Write-Host "Error al consultar estado inicial: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host "`n🚀 Enviando requests normales para establecer baseline..." -ForegroundColor Green
for ($i = 1; $i -le 10; $i++) {
    try {
        $response = Invoke-WebRequest -Uri "http://localhost:8089/api/hello" -UseBasicParsing -TimeoutSec 5
        Write-Host "✅ Baseline request $i - OK" -ForegroundColor Green
    }
    catch {
        Write-Host "❌ Baseline request $i - Error" -ForegroundColor Red
    }
    Start-Sleep -Milliseconds 300
}

Write-Host "`n💥 Simulando fallos (requests a endpoint inexistente)..." -ForegroundColor Red
$failureCount = 0
for ($i = 1; $i -le 20; $i++) {
    try {
        $response = Invoke-WebRequest -Uri "http://localhost:8089/api/nonexistent" -UseBasicParsing -TimeoutSec 3
    }
    catch {
        $failureCount++
        if ($_.Exception.Message -like "*404*") {
            Write-Host "💥 Failure request $i - 404 (esperado)" -ForegroundColor Yellow
        } else {
            Write-Host "💥 Failure request $i - $($_.Exception.Message)" -ForegroundColor Red
        }
    }
    Start-Sleep -Milliseconds 200
}

Write-Host "`n📊 Estado después de fallos:" -ForegroundColor Cyan
try {
    $postFailureStatus = Invoke-RestMethod -Uri "http://localhost:9000/api/circuit-breaker" -Method GET
    Write-Host "$($postFailureStatus | ConvertTo-Json -Depth 3)" -ForegroundColor White
}
catch {
    Write-Host "Error al consultar estado: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host "`n🔄 Probando requests normales después de fallos..." -ForegroundColor Cyan
$successAfterFailure = 0
$errorsAfterFailure = 0

for ($i = 1; $i -le 10; $i++) {
    try {
        $response = Invoke-WebRequest -Uri "http://localhost:8089/api/hello" -UseBasicParsing -TimeoutSec 5
        if ($response.StatusCode -eq 200) {
            $successAfterFailure++
            Write-Host "✅ Post-failure request $i - OK" -ForegroundColor Green
        }
    }
    catch {
        $errorsAfterFailure++
        Write-Host "❌ Post-failure request $i - Error: $($_.Exception.Message)" -ForegroundColor Red
    }
    Start-Sleep -Seconds 1
}

Write-Host "`n📊 Estado final:" -ForegroundColor Cyan
try {
    $finalStatus = Invoke-RestMethod -Uri "http://localhost:9000/api/circuit-breaker" -Method GET
    Write-Host "$($finalStatus | ConvertTo-Json -Depth 3)" -ForegroundColor White
}
catch {
    Write-Host "Error al consultar estado final: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host "`n🛡️ Resultados de la Prueba de Circuit Breaker:" -ForegroundColor Cyan
Write-Host "   💥 Fallos simulados: $failureCount" -ForegroundColor Red
Write-Host "   ✅ Éxitos post-fallo: $successAfterFailure" -ForegroundColor Green
Write-Host "   ❌ Errores post-fallo: $errorsAfterFailure" -ForegroundColor Red
