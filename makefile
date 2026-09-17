.PHONY: test verify bruno build

test:
	go test ./...

verify:
	go run ./framework/cmd/verify

bruno:
	go run ./framework/cmd/brunogen -api-md docs/API.md

build:
	go build -o bin/verify ./framework/cmd/verify
	go build -o bin/brunogen ./framework/cmd/brunogen
	go build -o bin/mongogen ./framework/cmd/mongogen
