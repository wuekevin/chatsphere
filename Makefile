BINARY   := chatsphere
CMD      := ./cmd/chatsphere
VERSION  := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS  := -ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: all build run dev tidy clean test

all: build

## build: compile a release binary for the current platform
build: tidy
	go build $(LDFLAGS) -o $(BINARY) $(CMD)

## run: build and run (expects config.yaml or starts installer)
run: build
	./$(BINARY)

## dev: run with race detector and debug mode
dev:
	go run -race $(CMD)

## tidy: sync go.mod / go.sum
tidy:
	go mod tidy

## test: run all tests
test:
	go test ./... -v -count=1

## clean: remove build artefacts
clean:
	rm -f $(BINARY)

## docker-build: build the Docker image
docker-build:
	docker build -t chatsphere:$(VERSION) -t chatsphere:latest .

## docker-up: start full stack with Docker Compose
docker-up:
	docker compose -f deploy/docker-compose.yml up -d

## docker-down: stop the stack
docker-down:
	docker compose -f deploy/docker-compose.yml down
