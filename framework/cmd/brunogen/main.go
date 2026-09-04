// Command brunogen turns a hexag project's Fiber routes into a Bruno API
// client collection (.bru files, mirroring the URL tree) plus an optional
// API.md contract reference, by parsing the routes/*.go files with go/ast —
// the same approach as cmd/mongogen, no compilation or reflection involved.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/king-glitch/hexag/framework/cmd/brunogen/internal/generator"
	"github.com/king-glitch/hexag/framework/cmd/brunogen/internal/parser"
)

func main() {
	var (
		routesFlag         string
		compositionFlag    string
		portsFlag          string
		basePathFlag       string
		outFlag            string
		apiMDFlag          string
		collectionNameFlag string
	)

	flag.StringVar(&routesFlag, "routes", "internal/adapters/endpoint/fiber/routes", "Directory of route files to parse")
	flag.StringVar(&compositionFlag, "composition", "internal/adapters/endpoint/fiber/handler.go", "File wiring handler.Register(api.Group(\"/x\")) calls")
	flag.StringVar(&portsFlag, "ports", "internal/ports", "Directory of the project's ports package (for resolving field types)")
	flag.StringVar(&basePathFlag, "base-path", "/api/v1", "API base path prefix")
	flag.StringVar(&outFlag, "out", "docs/bruno", "Output directory for the generated Bruno collection")
	flag.StringVar(&apiMDFlag, "api-md", "", "Optional path to also write a Markdown API contract reference")
	flag.StringVar(&collectionNameFlag, "collection-name", "", "Bruno collection name (default: module name)")
	flag.Parse()

	module, err := findModulePath(".")
	if err != nil && collectionNameFlag == "" {
		log.Fatalf("Failed to determine module name: %v (pass -collection-name explicitly)", err)
	}
	if collectionNameFlag == "" {
		parts := strings.Split(module, "/")
		collectionNameFlag = parts[len(parts)-1]
	}

	routes, err := parser.Parse(routesFlag, compositionFlag, portsFlag, basePathFlag)
	if err != nil {
		log.Fatalf("Failed to parse routes: %v", err)
	}
	if len(routes) == 0 {
		log.Fatalf("No routes discovered under %s (check -composition wiring)", routesFlag)
	}

	if err := os.MkdirAll(filepath.Dir(outFlag), 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}
	if err := generator.Write(routes, basePathFlag, outFlag, collectionNameFlag); err != nil {
		log.Fatalf("Failed to write Bruno collection: %v", err)
	}
	fmt.Printf("Generated %d requests into %s\n", len(routes), outFlag)

	if apiMDFlag != "" {
		if err := os.MkdirAll(filepath.Dir(apiMDFlag), 0755); err != nil {
			log.Fatalf("Failed to create API.md directory: %v", err)
		}
		if err := generator.WriteMarkdown(routes, basePathFlag, apiMDFlag, collectionNameFlag+" API"); err != nil {
			log.Fatalf("Failed to write API.md: %v", err)
		}
		fmt.Printf("Generated %s\n", apiMDFlag)
	}
}

// findModulePath walks upward from dir looking for go.mod and returns its
// module path — copied from cmd/mongogen, generic to any project.
func findModulePath(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}

	for {
		goModPath := filepath.Join(abs, "go.mod")
		if module, ok := readModuleLine(goModPath); ok {
			return module, nil
		}

		parent := filepath.Dir(abs)
		if parent == abs {
			return "", fmt.Errorf("no go.mod found above %s", dir)
		}
		abs = parent
	}
}

func readModuleLine(goModPath string) (string, bool) {
	f, err := os.Open(goModPath)
	if err != nil {
		return "", false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if module, ok := strings.CutPrefix(line, "module "); ok {
			return strings.TrimSpace(module), true
		}
	}

	return "", false
}
