# Script de Inicio Rápido
# Inicia todos los servicios necesarios para las pruebas

Write-Host "🚀 INICIO RÁPIDO - Load Balancer Testing Environment" -ForegroundColor Green
Write-Host "====================================================" -ForegroundColor Green

$projectRoot = Split-Path -Parent $PSScriptRoot

# Verificar prerrequisitos
Write-Host "`n🔍 Verificando prerrequisitos..." -ForegroundColor Cyan

# Verificar Go
try {
    $goVersion = & go version 2>$null
    Write-Host "   ✅ Go disponible: $goVersion" -ForegroundColor Green
}
catch {
    Write-Host "   ❌ Go no encontrado. Instala Go para continuar." -ForegroundColor Red
    exit 1
}

# Verificar Docker
try {
    $dockerVersion = & docker --version 2>$null
    Write-Host "   ✅ Docker disponible: $dockerVersion" -ForegroundColor Green
}
catch {
    Write-Host "   ⚠️  Docker no encontrado. Los backends no estarán disponibles." -ForegroundColor Yellow
}

# Verificar Node/PNPM para frontend
$frontendPath = Join-Path $projectRoot "balancer-front"
if (Test-Path $frontendPath) {
    try {
        Set-Location $frontendPath
        $pnpmVersion = & pnpm --version 2>$null
        Write-Host "   ✅ PNPM disponible: v$pnpmVersion" -ForegroundColor Green
        Set-Location $projectRoot
    }
    catch {
        Write-Host "   ⚠️  PNPM no encontrado. Frontend no estará disponible." -ForegroundColor Yellow
    }
}

Write-Host "`n📦 Iniciando servicios..." -ForegroundColor Cyan

# 1. Iniciar backends Docker (si está disponible)
if (Get-Command docker -ErrorAction SilentlyContinue) {
    Write-Host "`n🐳 Iniciando backends Docker..." -ForegroundColor Blue
    Set-Location $projectRoot
    
    try {
        & docker-compose up -d
        Write-Host "   ✅ Backends Docker iniciados" -ForegroundColor Green
        
        # Esperar a que los backends estén listos
        Write-Host "   ⏳ Esperando que los backends estén listos..." -ForegroundColor Yellow
        Start-Sleep -Seconds 5
        
        # Verificar backends
        for ($port = 8080; $port -le 8082; $port++) {
            try {
                $response = Invoke-WebRequest -Uri "http://localhost:$port" -UseBasicParsing -TimeoutSec 3
                Write-Host "   ✅ Backend en puerto $port: OK" -ForegroundColor Green
            }
            catch {
                Write-Host "   ⚠️  Backend en puerto $port: No responde" -ForegroundColor Yellow
            }
        }
    }
    catch {
        Write-Host "   ❌ Error iniciando backends: $($_.Exception.Message)" -ForegroundColor Red
    }
}

# 2. Compilar y iniciar Load Balancer
Write-Host "`n⚖️ Iniciando Load Balancer..." -ForegroundColor Magenta
Set-Location $projectRoot

try {
    # Compilar
    Write-Host "   🔨 Compilando load balancer..." -ForegroundColor Yellow
    & go build -o balancer.exe ./balancer/cmd
    
    if (Test-Path "./balancer.exe") {
        Write-Host "   ✅ Compilación exitosa" -ForegroundColor Green
        
        # Iniciar en background
        Write-Host "   🚀 Iniciando load balancer..." -ForegroundColor Yellow
        $loadBalancerJob = Start-Job -ScriptBlock {
            param($projectPath)
            Set-Location $projectPath
            ./balancer.exe
        } -ArgumentList $projectRoot
        
        # Esperar a que esté listo
        Start-Sleep -Seconds 3
        
        # Verificar que esté funcionando
        try {
            $response = Invoke-WebRequest -Uri "http://localhost:8089/api/hello" -UseBasicParsing -TimeoutSec 5
            Write-Host "   ✅ Load balancer funcionando en puerto 8089" -ForegroundColor Green
            
            $adminResponse = Invoke-WebRequest -Uri "http://localhost:9000/api/metrics" -UseBasicParsing -TimeoutSec 5
            Write-Host "   ✅ API Admin funcionando en puerto 9000" -ForegroundColor Green
        }
        catch {
            Write-Host "   ⚠️  Load balancer iniciado pero no responde aún" -ForegroundColor Yellow
        }
    }
}
catch {
    Write-Host "   ❌ Error compilando/iniciando load balancer: $($_.Exception.Message)" -ForegroundColor Red
}

# 3. Iniciar Frontend (opcional)
if (Test-Path $frontendPath -and (Get-Command pnpm -ErrorAction SilentlyContinue)) {
    Write-Host "`n🌐 ¿Quieres iniciar el frontend dashboard? (y/n): " -ForegroundColor Cyan -NoNewline
    $startFrontend = Read-Host
    
    if ($startFrontend -eq 'y' -or $startFrontend -eq 'Y') {
        try {
            Set-Location $frontendPath
            Write-Host "   📦 Instalando dependencias frontend..." -ForegroundColor Yellow
            & pnpm install
            
            Write-Host "   🚀 Iniciando servidor de desarrollo..." -ForegroundColor Yellow
            $frontendJob = Start-Job -ScriptBlock {
                param($frontendPath)
                Set-Location $frontendPath
                & pnpm dev
            } -ArgumentList $frontendPath
            
            Write-Host "   ✅ Frontend iniciándose en http://localhost:5173" -ForegroundColor Green
            Set-Location $projectRoot
        }
        catch {
            Write-Host "   ❌ Error iniciando frontend: $($_.Exception.Message)" -ForegroundColor Red
            Set-Location $projectRoot
        }
    }
}

# Resumen del entorno
Write-Host "`n🎯 ENTORNO LISTO PARA PRUEBAS" -ForegroundColor Green
Write-Host "=============================" -ForegroundColor Green
Write-Host ""
Write-Host "📍 Servicios disponibles:" -ForegroundColor White
Write-Host "   🔗 Load Balancer (Proxy): http://localhost:8089" -ForegroundColor Cyan
Write-Host "   🔗 Admin API: http://localhost:9000" -ForegroundColor Cyan
Write-Host "   🔗 Frontend Dashboard: http://localhost:5173" -ForegroundColor Cyan
Write-Host "   🔗 Backends: http://localhost:8080-8082" -ForegroundColor Cyan

Write-Host "`n🧪 Scripts de prueba disponibles:" -ForegroundColor White
Write-Host "   📊 Prueba básica:         .\\scripts\\basic_load_test.ps1" -ForegroundColor Yellow
Write-Host "   ⚖️  Distribución de carga: .\\scripts\\load_balancing_test.ps1" -ForegroundColor Yellow
Write-Host "   ⚡ Rate limiting:         .\\scripts\\rate_limit_test.ps1" -ForegroundColor Yellow
Write-Host "   🛡️  Circuit breaker:      .\\scripts\\circuit_breaker_test.ps1" -ForegroundColor Yellow
Write-Host "   🔥 Prueba de estrés:      .\\scripts\\stress_test.ps1" -ForegroundColor Yellow
Write-Host "   📈 Monitor en tiempo real: .\\scripts\\monitor_realtime.ps1" -ForegroundColor Yellow
Write-Host "   🎯 Suite completa:        .\\scripts\\comprehensive_test.ps1" -ForegroundColor Yellow

Write-Host "`n💡 Comandos de ejemplo:" -ForegroundColor White
Write-Host "   # Prueba rápida" -ForegroundColor Gray
Write-Host "   .\\scripts\\basic_load_test.ps1" -ForegroundColor Gray
Write-Host ""
Write-Host "   # Ver métricas en vivo" -ForegroundColor Gray
Write-Host "   .\\scripts\\monitor_realtime.ps1" -ForegroundColor Gray
Write-Host ""
Write-Host "   # Suite completa de pruebas" -ForegroundColor Gray
Write-Host "   .\\scripts\\comprehensive_test.ps1" -ForegroundColor Gray

Write-Host "`n🎉 ¡Listo para hacer pruebas de carga!" -ForegroundColor Green