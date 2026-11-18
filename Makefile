.PHONY: all build test test-golden clean run inspect serve validate

BINARY_NAME=4d-openapi

all: test build

build:
	go build -o $(BINARY_NAME) ./cmd/4d-openapi

test:
	go test -v ./...

test-golden:
	go test -v ./cmd/4d-openapi -run TestGoldenDeterminism

run: build
	./$(BINARY_NAME) generate --project testdata/fixtures/TestApp --output dist/openapi.yaml --format yaml

inspect: build
	./$(BINARY_NAME) inspect --project testdata/fixtures/TestApp

serve: build
	./$(BINARY_NAME) serve --project testdata/fixtures/TestApp --listen 127.0.0.1:8088

validate: build
	./$(BINARY_NAME) validate --file testdata/fixtures/TestApp/openapi.yaml

clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME).exe
	rm -rf dist/
