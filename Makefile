.PHONY: build run docker tidy clean

BINARY=monitor-server

build:
	go build -ldflags="-s -w" -o $(BINARY) ./cmd/server

tidy:
	go mod tidy

run: build
	./$(BINARY)

docker:
	docker compose build

docker-run:
	docker compose up

docker-down:
	docker compose down

clean:
	rm -f $(BINARY)

install-browsers:
	go run github.com/mxschmitt/playwright-go/cmd/playwright@v0.6100.0 install chromium
