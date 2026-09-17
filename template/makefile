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
	@which hexag >/dev/null 2>&1 && hexag bruno || go run github.com/king-glitch/hexag/cmd/hexag bruno -api-md docs/API.md

verify:
	@which hexag >/dev/null 2>&1 && hexag verify || go run github.com/king-glitch/hexag/cmd/hexag verify

build:
	CGO_ENABLED=0 go build -o bin/api ./commands
