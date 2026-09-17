package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func runUpdate(args []string) error {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	updateMockery := fs.Bool("mockery", false, "Also update .mockery.yml")
	updateMakefile := fs.Bool("makefile", false, "Also update makefile")
	updateAll := fs.Bool("all", false, "Update all framework config files (.mockery.yml, makefile)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	destDir := "."
	if fs.NArg() > 0 {
		destDir = fs.Arg(0)
	}

	absDest, err := filepath.Abs(destDir)
	if err != nil {
		return fmt.Errorf("failed to resolve directory %s: %w", destDir, err)
	}

	goModPath := filepath.Join(absDest, "go.mod")
	modulePath, ok := readModuleLine(goModPath)
	if !ok || modulePath == "" {
		return fmt.Errorf("no valid go.mod found in %s", absDest)
	}

	fmt.Printf("Target project: %s\n", absDest)
	fmt.Printf("Module path:    %s\n", modulePath)

	// 1. Update AGENTS.md
	agentsFile, err := getTemplateFile("AGENTS.md")
	if err != nil {
		return fmt.Errorf("failed to load template AGENTS.md: %w", err)
	}
	agentsContent := strings.ReplaceAll(string(agentsFile.Content), "{{MODULE_PATH}}", modulePath)
	if err := os.WriteFile(filepath.Join(absDest, "AGENTS.md"), []byte(agentsContent), 0644); err != nil {
		return fmt.Errorf("failed to write AGENTS.md: %w", err)
	}
	fmt.Println("✓ updated AGENTS.md")

	// 2. Ensure CLAUDE.md has @AGENTS.md
	claudePath := filepath.Join(absDest, "CLAUDE.md")
	contractClause := "@AGENTS.md\n\nRun `make verify` before updating `MEMORY.md`. If it fails, fix the code immediately.\n"
	if existing, err := os.ReadFile(claudePath); err != nil {
		if err := os.WriteFile(claudePath, []byte(contractClause), 0644); err != nil {
			return fmt.Errorf("failed to create CLAUDE.md: %w", err)
		}
		fmt.Println("✓ created CLAUDE.md")
	} else if !strings.Contains(string(existing), "@AGENTS.md") {
		newContent := contractClause + "\n" + string(existing)
		if err := os.WriteFile(claudePath, []byte(newContent), 0644); err != nil {
			return fmt.Errorf("failed to update CLAUDE.md: %w", err)
		}
		fmt.Println("✓ prepended @AGENTS.md to CLAUDE.md")
	} else {
		fmt.Println("✓ CLAUDE.md already references @AGENTS.md")
	}

	// 3. Optional: .mockery.yml
	if *updateMockery || *updateAll {
		if mockeryFile, err := getTemplateFile(".mockery.yml"); err == nil {
			content := strings.ReplaceAll(string(mockeryFile.Content), "{{MODULE_PATH}}", modulePath)
			if err := os.WriteFile(filepath.Join(absDest, ".mockery.yml"), []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to write .mockery.yml: %w", err)
			}
			fmt.Println("✓ updated .mockery.yml")
		}
	}

	// 4. Optional: makefile
	if *updateMakefile || *updateAll {
		if makefileFile, err := getTemplateFile("makefile"); err == nil {
			if err := os.WriteFile(filepath.Join(absDest, "makefile"), makefileFile.Content, 0644); err != nil {
				return fmt.Errorf("failed to write makefile: %w", err)
			}
			fmt.Println("✓ updated makefile")
		}
	}

	// 5. MEMORY.md
	memoryPath := filepath.Join(absDest, "MEMORY.md")
	if _, err := os.Stat(memoryPath); os.IsNotExist(err) {
		if memoryFile, err := getTemplateFile("MEMORY.md"); err == nil {
			_ = os.WriteFile(memoryPath, memoryFile.Content, 0644)
			fmt.Println("✓ created MEMORY.md")
		}
	}

	return nil
}
