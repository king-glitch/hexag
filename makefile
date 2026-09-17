.PHONY: test verify bruno build install package-template

test:
	go test ./...

verify:
	go run ./cmd/hexag verify

bruno:
	go run ./cmd/hexag bruno -api-md docs/API.md

package-template:
	tar -czf cmd/hexag/template.tar.gz -C template .

install: package-template
	go install ./cmd/hexag

build: package-template
	go build -o bin/hexag ./cmd/hexag
	go build -o bin/verify ./framework/cmd/verify
	go build -o bin/brunogen ./framework/cmd/brunogen
	go build -o bin/mongogen ./framework/cmd/mongogen
