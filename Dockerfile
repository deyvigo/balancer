# Etapa 1
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN cd service && go build -ldflags "-s -w" -o ../service .

# Etapa 2
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/service .
EXPOSE 8080

CMD ["./service", "8080"]