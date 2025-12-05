# Etapa 1: build da aplicação Go
FROM golang:1.25-alpine AS builder

# Instala dependências básicas
RUN apk add --no-cache git

# Define diretório de trabalho
WORKDIR /app

# Copia arquivos Go
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Compila a aplicação
RUN go build -o consumer main.go

# Etapa 2: imagem final mínima
FROM alpine:3.18

# Instala dependências básicas
RUN apk add --no-cache ca-certificates bash

# Copia binário compilado
WORKDIR /app
COPY --from=builder /app/consumer .

# Permissão de execução
RUN chmod +x ./consumer

# Define comando padrão
CMD ["./consumer"]
