package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/king-glitch/hexag/framework/cmd/verify/rules"
)

func runVerify(args []string) error {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	verbose := fs.Bool("v", false, "verbose output")
	dir := fs.String("dir", "", "target directory to verify (defaults to current directory)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	targetDir := *dir
	if targetDir == "" {
		if fs.NArg() > 0 {
			targetDir = fs.Arg(0)
		} else {
			targetDir = "."
		}
	}

	targetDir = strings.TrimSuffix(targetDir, "/...")
	targetDir = strings.TrimSuffix(targetDir, "...")

	absDir, err := filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("failed to resolve directory: %w", err)
	}

	verifier := rules.NewVerifier()

	internalDir := filepath.Join(absDir, "internal")
	templateInternalDir := filepath.Join(absDir, "template", "internal")

	var dirsToVerify []string
	if info, err := os.Stat(internalDir); err == nil && info.IsDir() {
		dirsToVerify = append(dirsToVerify, internalDir)
	}
	if info, err := os.Stat(templateInternalDir); err == nil && info.IsDir() {
		dirsToVerify = append(dirsToVerify, templateInternalDir)
	}

	if len(dirsToVerify) == 0 {
		dirsToVerify = append(dirsToVerify, absDir)
	}

	for _, d := range dirsToVerify {
		if *verbose {
			fmt.Printf("Verifying directory: %s\n", d)
		}
		if err := verifier.VerifyPath(d); err != nil {
			return err
		}
	}

	violations := verifier.Violations()
	if len(violations) > 0 {
		fileSet := make(map[string]struct{})
		for _, v := range violations {
			if v.Pos.Filename != "" {
				fileSet[v.Pos.Filename] = struct{}{}
			}
		}

		fmt.Fprintf(os.Stderr, "\n❌ Verification failed: %d violation(s) found across %d file(s).\n\n", len(violations), len(fileSet))
		for i, v := range violations {
			fmt.Fprintln(os.Stderr, v.Format(i+1, len(violations)))
		}
		fmt.Fprintf(os.Stderr, "💡 Summary: %d violation(s) found across %d file(s).\n", len(violations), len(fileSet))
		fmt.Fprintln(os.Stderr, "Please fix the violations above and run 'make verify' (or 'hexag verify') again.")
		os.Exit(1)
	}

	fmt.Printf("✓ hexag verify: all architecture and naming rules passed (%d Go files checked)\n", verifier.FileCount())
	return nil
}
