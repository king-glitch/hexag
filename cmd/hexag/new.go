package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func runNew(args []string) error {
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	localPath := fs.String("local", "", "Local hexag replacement path (optional for framework developers)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	positional := fs.Args()
	if len(positional) < 2 {
		return fmt.Errorf("usage: hexag new <module-path> <dest-dir> [db-name]")
	}

	modulePath := positional[0]
	destDir := positional[1]
	dbName := ""
	if len(positional) >= 3 {
		dbName = positional[2]
	} else {
		base := filepath.Base(modulePath)
		dbName = strings.ReplaceAll(base, "-", "_")
	}

	absDest, err := filepath.Abs(destDir)
	if err != nil {
		return fmt.Errorf("failed to resolve destination directory: %w", err)
	}

	// Check if directory exists and has non-ignored contents
	ignoreMap := map[string]bool{
		".idea":     true,
		".vscode":   true,
		".DS_Store": true,
		".git":      true,
		".fleet":    true,
	}

	if entries, err := os.ReadDir(absDest); err == nil && len(entries) > 0 {
		var realEntries []string
		for _, e := range entries {
			if !ignoreMap[e.Name()] {
				realEntries = append(realEntries, e.Name())
			}
		}
		if len(realEntries) > 0 {
			return fmt.Errorf("destination directory already has project content: %v", realEntries)
		}
	}

	if err := os.MkdirAll(absDest, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	files, err := loadTemplateFiles()
	if err != nil {
		return fmt.Errorf("failed to load template files: %w", err)
	}

	for _, f := range files {
		targetPath := filepath.Join(absDest, f.Path)
		if f.IsDir {
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}

		content := f.Content
		contentStr := string(content)
		contentStr = strings.ReplaceAll(contentStr, "{{MODULE_PATH}}", modulePath)
		contentStr = strings.ReplaceAll(contentStr, "{{DB_NAME}}", dbName)

		if f.Path == "go.mod" {
			if *localPath != "" {
				contentStr = strings.ReplaceAll(contentStr, "{{HEXAG_PATH}}", *localPath)
			} else {
				// Remove local replace line for standard consumers
				var cleanLines []string
				for _, line := range strings.Split(contentStr, "\n") {
					if strings.HasPrefix(strings.TrimSpace(line), "replace github.com/king-glitch/hexag =>") {
						continue
					}
					cleanLines = append(cleanLines, line)
				}
				contentStr = strings.Join(cleanLines, "\n")
			}
		}

		mode := f.Mode
		if mode == 0 {
			mode = 0644
		}
		if err := os.WriteFile(targetPath, []byte(contentStr), mode); err != nil {
			return fmt.Errorf("failed to write %s: %w", targetPath, err)
		}
	}

	// Create CLAUDE.md
	claudePath := filepath.Join(absDest, "CLAUDE.md")
	claudeContent := "@AGENTS.md\n\nRun `make verify` before updating `MEMORY.md`. If it fails, fix the code immediately.\n"
	if err := os.WriteFile(claudePath, []byte(claudeContent), 0644); err != nil {
		return fmt.Errorf("failed to write CLAUDE.md: %w", err)
	}

	// Copy .env.example to .env
	envExPath := filepath.Join(absDest, ".env.example")
	envPath := filepath.Join(absDest, ".env")
	if envBytes, err := os.ReadFile(envExPath); err == nil {
		_ = os.WriteFile(envPath, envBytes, 0644)
	}

	fmt.Printf("Scaffolded %s for %s\n", absDest, modulePath)

	// Generate initial mongo models directly so internal/adapters/database/mongo/models exists
	domainFile := filepath.Join(absDest, "internal", "ports", "domain.go")
	modelsOut := filepath.Join(absDest, "internal", "adapters", "database", "mongo", "models")
	if err := runMongo([]string{"-file", domainFile, "-out", modelsOut, "-pkg", "models"}); err != nil {
		fmt.Fprintf(os.Stderr, "warning: initial model generation failed: %v\n", err)
	}

	// Run go mod download & go mod tidy
	cmds := [][]string{
		{"go", "mod", "download"},
		{"go", "mod", "tidy"},
	}

	for _, cmdArgs := range cmds {
		cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		cmd.Dir = absDest
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: %s failed: %v (%s)\n", strings.Join(cmdArgs, " "), err, strings.TrimSpace(stderr.String()))
		}
	}

	fmt.Printf("\nDone! Next steps:\n  cd %s\n  go build ./...\n  # edit internal/ports/domain.go\n  # make generate\n  # make verify\n", destDir)
	return nil
}
