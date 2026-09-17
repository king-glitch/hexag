package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/king-glitch/hexag/framework/cmd/mongogen/generator"
	"github.com/king-glitch/hexag/framework/cmd/mongogen/parser"
)

func runMongo(args []string) error {
	fs := flag.NewFlagSet("mongo", flag.ContinueOnError)
	typeFlag := fs.String("type", "", "Target struct name(s) to parse, comma-separated, 'all', or empty for *Model")
	fileFlag := fs.String("file", "", "Go source file to parse (defaults to $GOFILE or internal/ports/domain.go)")
	outFlag := fs.String("out", "", "Output directory or file path")
	pkgFlag := fs.String("pkg", "", "Target package name for generated code (default 'models' or directory name)")
	portsImportFlag := fs.String("portsimport", "", "Import path for the project's ports package")
	standaloneFlag := fs.Bool("standalone", false, "Generate self-contained file with embedded Field[T]")

	if err := fs.Parse(args); err != nil {
		return err
	}

	targetFile := *fileFlag
	if targetFile == "" {
		targetFile = os.Getenv("GOFILE")
	}
	if targetFile == "" {
		targetFile = "internal/ports/domain.go"
	}

	targetOut := *outFlag
	if targetOut == "" {
		targetOut = "internal/adapters/database/mongo/models"
	}

	targetPkg := *pkgFlag
	if targetPkg == "" {
		if strings.HasSuffix(targetOut, ".go") {
			targetPkg = filepath.Base(filepath.Clean(filepath.Dir(targetOut)))
		} else {
			targetPkg = filepath.Base(filepath.Clean(targetOut))
		}
	}

	portsImport := *portsImportFlag
	if portsImport == "" {
		mod, err := findModulePath(filepath.Dir(targetFile))
		if err != nil {
			return fmt.Errorf("failed to determine ports import path: %w (pass -portsimport explicitly)", err)
		}
		portsImport = mod + "/internal/ports"
	}

	p := parser.NewParser(targetPkg, portsImport)
	metas, err := p.ParseFile(targetFile, *typeFlag)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", targetFile, err)
	}
	if len(metas) == 0 {
		return fmt.Errorf("no matching structs found in %s", targetFile)
	}

	gen, err := generator.NewGenerator()
	if err != nil {
		return fmt.Errorf("failed to create generator: %w", err)
	}

	if strings.HasSuffix(targetOut, ".go") {
		dir := filepath.Dir(targetOut)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory %s: %w", dir, err)
		}
		if len(metas) > 1 {
			return fmt.Errorf("cannot write multiple structs (%d) to a single file %s", len(metas), targetOut)
		}
		content, err := gen.GenerateModel(metas[0], targetPkg, *standaloneFlag)
		if err != nil {
			return fmt.Errorf("failed to generate model for %s: %w", metas[0].StructName, err)
		}
		if err := os.WriteFile(targetOut, content, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", targetOut, err)
		}
		fmt.Printf("Generated %s for struct %s\n", targetOut, metas[0].StructName)
		return nil
	}

	if err := os.MkdirAll(targetOut, 0755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", targetOut, err)
	}

	if *standaloneFlag {
		fieldContent, err := gen.GenerateFieldsFile(targetPkg)
		if err != nil {
			return fmt.Errorf("failed to generate field.go: %w", err)
		}
		fieldPath := filepath.Join(targetOut, "field.go")
		if err := os.WriteFile(fieldPath, fieldContent, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", fieldPath, err)
		}
		fmt.Printf("Generated %s\n", fieldPath)
	}

	for _, meta := range metas {
		modelContent, err := gen.GenerateModel(meta, targetPkg, *standaloneFlag)
		if err != nil {
			return fmt.Errorf("failed to generate model for %s: %w", meta.StructName, err)
		}
		modelPath := filepath.Join(targetOut, meta.FileName)
		if err := os.WriteFile(modelPath, modelContent, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", modelPath, err)
		}
		fmt.Printf("Generated %s for struct %s\n", modelPath, meta.StructName)
	}

	return nil
}
