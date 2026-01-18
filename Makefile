.PHONY: build test clean install

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

# Build do binário
build:
	go build $(LDFLAGS) -o oi ./cmd/oi

# Build estático para Linux (produção)
build-static:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o oi-linux-amd64 ./cmd/oi

# Rodar testes unitários
test:
	go test -v ./...

# Rodar testes com cobertura
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Limpar artefatos de build
clean:
	rm -f oi oi-linux-amd64 coverage.out coverage.html

# Instalar localmente
install:
	go install $(LDFLAGS) ./cmd/oi

# Verificar se compila
check:
	go build ./...

# Formatar código
fmt:
	go fmt ./...

# Linter
lint:
	go vet ./...

# Atualizar dependências
deps:
	go mod tidy
