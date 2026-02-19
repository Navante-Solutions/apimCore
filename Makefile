.PHONY: build test lint docker run clean

BINARY := apimcore
MAIN   := ./cmd/apim

build:
	go build -ldflags="-s -w" -o $(BINARY) $(MAIN)

test:
	go test -v -race -count=1 ./...

lint:
	golangci-lint run --timeout=5m

docker:
	docker build -t apimcore:local .

run: build
	./$(BINARY)

clean:
	rm -f $(BINARY) $(BINARY).exe
