- Después de clonar el repositorio
- Instalar las dependencias
```bash
go mod tidy
```

1. Inicializar las réplicas
```bash
docker-compose up --build --scale go-service=3
```

2. Inicializar el balanceador
```bash
go run ./balancer/cmd/main.go
```

* Test
```bash
docker-compose ps -a
```

```bash
docker stop ${service}
```

```bash
docker start ${service}
```