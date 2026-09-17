package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/king-glitch/hexag/framework/cmd/brunogen/generator"
	"github.com/king-glitch/hexag/framework/cmd/brunogen/parser"
)

func runBruno(args []string) error {
	fs := flag.NewFlagSet("bruno", flag.ContinueOnError)
	routesFlag := fs.String("routes", "internal/adapters/endpoint/fiber/routes", "Directory of route files to parse")
	compositionFlag := fs.String("composition", "internal/adapters/endpoint/fiber/handler.go", "File wiring handler.Register(api.Group(\"/x\")) calls")
	portsFlag := fs.String("ports", "internal/ports", "Directory of the project's ports package")
	basePathFlag := fs.String("base-path", "/api/v1", "API base path prefix")
	outFlag := fs.String("out", "docs/bruno", "Output directory for the generated Bruno collection")
	apiMDFlag := fs.String("api-md", "docs/API.md", "Path to write Markdown API contract reference")
	collectionNameFlag := fs.String("collection-name", "", "Bruno collection name (default: module name)")
	excludeFlag := fs.String("exclude", "", "Comma-separated wildcard or regex patterns of routes to exclude")

	if err := fs.Parse(args); err != nil {
		return err
	}

	module, err := findModulePath(".")
	if err != nil && *collectionNameFlag == "" {
		return fmt.Errorf("failed to determine module name: %w (pass -collection-name explicitly)", err)
	}
	colName := *collectionNameFlag
	if colName == "" {
		parts := strings.Split(module, "/")
		colName = parts[len(parts)-1]
	}

	routes, err := parser.Parse(*routesFlag, *compositionFlag, *portsFlag, *basePathFlag)
	if err != nil {
		return fmt.Errorf("failed to parse routes: %w", err)
	}
	if len(routes) == 0 {
		return fmt.Errorf("no routes discovered under %s (check -composition wiring)", *routesFlag)
	}

	var excludePatterns []string
	if *excludeFlag != "" {
		for _, p := range strings.Split(*excludeFlag, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				excludePatterns = append(excludePatterns, p)
			}
		}
	}
	if len(excludePatterns) > 0 {
		var filtered []parser.Route
		for _, r := range routes {
			if generator.MatchesExclude(r, *basePathFlag, excludePatterns) {
				continue
			}
			filtered = append(filtered, r)
		}
		routes = filtered
	}

	if err := os.MkdirAll(filepath.Dir(*outFlag), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	if err := generator.Write(routes, *basePathFlag, *outFlag, colName); err != nil {
		return fmt.Errorf("failed to write Bruno collection: %w", err)
	}
	fmt.Printf("Generated %d requests into %s\n", len(routes), *outFlag)

	if *apiMDFlag != "" {
		if err := os.MkdirAll(filepath.Dir(*apiMDFlag), 0755); err != nil {
			return fmt.Errorf("failed to create API.md directory: %w", err)
		}
		if err := generator.WriteMarkdown(routes, *basePathFlag, *apiMDFlag, colName+" API"); err != nil {
			return fmt.Errorf("failed to write API.md: %w", err)
		}
		fmt.Printf("Generated %s\n", *apiMDFlag)
	}

	return nil
}

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
