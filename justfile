default: run

run *args:
    go run ./cmd/lhtui {{args}}

build:
    go build -o lhtui ./cmd/lhtui

test:
    go vet ./...
    go test ./...

lint:
    golangci-lint run

fmt:
    gofmt -w .

# Copy the golden fixtures from a local LearnHarness checkout.
golden path:
    rm -rf testdata/golden
    mkdir -p testdata/golden
    cp -R {{path}}/tests/fixtures/golden-data/{chats,graph,jobs,settings.json} testdata/golden/

nix-build:
    nix build --print-build-logs

vendor:
    go mod tidy
    go mod vendor
