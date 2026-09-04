package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/king-glitch/hexag/framework/cmd/mongogen/internal/generator"
	"github.com/king-glitch/hexag/framework/cmd/mongogen/internal/parser"
)

func main() {
	var (
		typeFlag        string
		fileFlag        string
		outFlag         string
		pkgFlag         string
		portsImportFlag string
		standaloneFlag  bool
	)

	flag.StringVar(&typeFlag, "type", "", "Target struct name(s) to parse, comma-separated, 'all', or empty for *Model")
	flag.StringVar(&fileFlag, "file", "", "Go source file to parse (defaults to $GOFILE)")
	flag.StringVar(&outFlag, "out", "", "Output directory or file path")
	flag.StringVar(&pkgFlag, "pkg", "", "Target package name for generated code (default 'models' or directory name)")
	flag.StringVar(
		&portsImportFlag,
		"portsimport",
		"",
		"Import path for the project's ports package (default: <module from go.mod>/internal/ports)",
	)
	flag.BoolVar(
		&standaloneFlag,
		"standalone",
		false,
		"Generate self-contained file with embedded Field[T] definitions instead of importing the framework's",
	)
	flag.Parse()

	if fileFlag == "" {
		fileFlag = os.Getenv("GOFILE")
	}
	if fileFlag == "" {
		fileFlag = "internal/ports/domain.go"
	}

	if outFlag == "" {
		outFlag = "internal/adapters/database/mongo/models"
	}

	if pkgFlag == "" {
		if strings.HasSuffix(outFlag, ".go") {
			dir := filepath.Dir(outFlag)
			pkgFlag = filepath.Base(filepath.Clean(dir))
		} else {
			pkgFlag = filepath.Base(filepath.Clean(outFlag))
		}
		if pkgFlag == "." || pkgFlag == "/" || pkgFlag == "" {
			pkgFlag = "models"
		}
	}

	if portsImportFlag == "" {
		module, err := findModulePath(".")
		if err != nil {
			log.Fatalf("Failed to determine ports import path: %v (pass -portsimport explicitly)", err)
		}
		portsImportFlag = module + "/internal/ports"
	}

	p := parser.NewParser(pkgFlag, portsImportFlag)
	metas, err := p.ParseFile(fileFlag, typeFlag)
	if err != nil {
		log.Fatalf("Failed to parse %s: %v", fileFlag, err)
	}

	if len(metas) == 0 {
		log.Fatalf("No matching structs found in %s", fileFlag)
	}

	gen, err := generator.NewGenerator()
	if err != nil {
		log.Fatalf("Failed to create generator: %v", err)
	}

	// Check if outFlag is a single .go file
	if strings.HasSuffix(outFlag, ".go") {
		dir := filepath.Dir(outFlag)
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("Failed to create output directory %s: %v", dir, err)
		}

		if len(metas) > 1 {
			log.Fatalf("Cannot write multiple structs (%d) to a single file %s", len(metas), outFlag)
		}

		content, err := gen.GenerateModel(metas[0], pkgFlag, standaloneFlag)
		if err != nil {
			log.Fatalf("Failed to generate model for %s: %v", metas[0].StructName, err)
		}

		if err := os.WriteFile(outFlag, content, 0644); err != nil {
			log.Fatalf("Failed to write %s: %v", outFlag, err)
		}

		fmt.Printf("Generated %s for struct %s\n", outFlag, metas[0].StructName)
		return
	}

	// Out directory mode
	if err := os.MkdirAll(outFlag, 0755); err != nil {
		log.Fatalf("Failed to create output directory %s: %v", outFlag, err)
	}

	// Generate field.go only in standalone mode; shared mode imports Field[T]
	// from the framework instead.
	if standaloneFlag {
		fieldContent, err := gen.GenerateFieldsFile(pkgFlag)
		if err != nil {
			log.Fatalf("Failed to generate field.go: %v", err)
		}

		fieldPath := filepath.Join(outFlag, "field.go")
		if err := os.WriteFile(fieldPath, fieldContent, 0644); err != nil {
			log.Fatalf("Failed to write %s: %v", fieldPath, err)
		}
		fmt.Printf("Generated %s\n", fieldPath)
	}

	for _, meta := range metas {
		modelContent, err := gen.GenerateModel(meta, pkgFlag, standaloneFlag)
		if err != nil {
			log.Fatalf("Failed to generate model for %s: %v", meta.StructName, err)
		}

		modelPath := filepath.Join(outFlag, meta.FileName)
		if err := os.WriteFile(modelPath, modelContent, 0644); err != nil {
			log.Fatalf("Failed to write %s: %v", modelPath, err)
		}
		fmt.Printf("Generated %s for %s (var: %s)\n", modelPath, meta.StructName, meta.VarName)
	}
}

// findModulePath walks upward from dir looking for go.mod and returns its
// module path, so a project's go:generate line does not have to spell out
// its own module path as a flag.
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
