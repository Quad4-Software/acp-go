COVERAGE_THRESHOLD ?= 75

all: fmt vet lint test build

fmt:
	gofumpt -l -w . 2>/dev/null || gofmt -l -w .

vet:
	go vet ./...

lint:
	golangci-lint run ./...

test:
	go test -race ./...

fuzz:
	go test -run=Fuzz -fuzz=FuzzUnionUnmarshal -fuzztime=30s .

cover:
	go test -race -coverprofile=coverage.out .

cover-check:
	@total=$$(go tool cover -func=coverage.out | awk '/^total:/ {gsub(/%/,"",$$3); print $$3}'); \
	echo "total coverage: $$total% (gate $(COVERAGE_THRESHOLD)%)"; \
	awk -v t="$$total" -v min="$(COVERAGE_THRESHOLD)" 'BEGIN{exit !(t+0>=min+0)}'

badge:
	go run ./tools/badge -label coverage -coverprofile coverage.out -out docs/coverage.svg

badge-check: cover badge
	git diff --exit-code docs/coverage.svg

build:
	go build ./...

clean:
	rm -f coverage.out
	rm -rf .bin

.PHONY: all fmt vet lint test fuzz cover cover-check badge badge-check build clean
