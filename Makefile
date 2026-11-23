.PHONY: all build test test-golden clean run inspect serve validate mock export-ts export-dart export-swift

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

mock: build
	./$(BINARY_NAME) mock --project testdata/fixtures/TestApp --listen 127.0.0.1:8088

validate: build
	./$(BINARY_NAME) validate --file testdata/fixtures/TestApp/openapi.yaml

export-ts: build
	./$(BINARY_NAME) export --project testdata/fixtures/TestApp --target typescript

export-dart: build
	./$(BINARY_NAME) export --project testdata/fixtures/TestApp --target dart

export-swift: build
	./$(BINARY_NAME) export --project testdata/fixtures/TestApp --target swift

clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME).exe
	rm -rf dist/
