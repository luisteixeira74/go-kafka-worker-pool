# Rodar testes
test:
	go test -v ./...

# Gerar binário
build:
	go build -o worker-pool-service main.go

# Executar aplicação
run:
	go run main.go

# Limpar arquivos gerados
clean:
	rm -f worker-pool-service coverage.out
