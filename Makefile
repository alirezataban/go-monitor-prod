.PHONY: build run docker tidy clean

BINARY=monitor-server

build:
	go build -ldflags="-s -w" -o $(BINARY) ./cmd/server

tidy:
	go mod tidy

run: build
	./$(BINARY)

docker:
	docker build -f Dockerfile.simple -t monitor-engine-go:latest .

docker-run:
	docker run --rm -p 8080:8080 --shm-size=256m monitor-engine-go:latest

clean:
	rm -f $(BINARY)

install-browsers:
	go run github.com/mxschmitt/playwright-go/cmd/playwright@v0.6100.0 install chromium
