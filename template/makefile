.PHONY: run test generate mockery bruno verify build

run:
	go run ./commands

test:
	go test ./...

generate:
	go generate ./...

mockery:
	mockery

bruno:
	@if which hexag >/dev/null 2>&1; then hexag bruno; else go run github.com/king-glitch/hexag/cmd/hexag bruno -api-md docs/API.md; fi

verify:
	@if which hexag >/dev/null 2>&1; then hexag verify; else go run github.com/king-glitch/hexag/cmd/hexag verify; fi

build:
	CGO_ENABLED=0 go build -o bin/api ./commands
