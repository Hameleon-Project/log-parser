.PHONY: fmt test vet lint run

fmt:
	gofmt -w cmd internal

test:
	go test ./...

vet:
	go vet ./...

lint: fmt vet
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.61.0 run ./...

run:
	go run ./cmd/app
