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
	go run github.com/king-glitch/hexag/framework/cmd/brunogen -api-md docs/API.md

verify:
	go run github.com/king-glitch/hexag/framework/cmd/verify

build:
	CGO_ENABLED=0 go build -o bin/api ./commands
