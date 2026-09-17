package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/king-glitch/hexag/framework/cmd/verify/internal/rules"
)

func main() {
	var (
		verbose = flag.Bool("v", false, "verbose output")
		dir     = flag.String("dir", "", "target directory to verify (defaults to current directory)")
	)
	flag.Parse()

	targetDir := *dir
	if targetDir == "" {
		if flag.NArg() > 0 {
			targetDir = flag.Arg(0)
		} else {
			targetDir = "."
		}
	}

	// If argument ends with /..., strip it
	targetDir = strings.TrimSuffix(targetDir, "/...")
	targetDir = strings.TrimSuffix(targetDir, "...")

	absDir, err := filepath.Abs(targetDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to resolve directory: %v\n", err)
		os.Exit(1)
	}

	verifier := rules.NewVerifier()

	// Check whether to verify internal/ or template/internal/ or whole directory
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
		// Fall back to scanning the provided directory directly
		dirsToVerify = append(dirsToVerify, absDir)
	}

	for _, d := range dirsToVerify {
		if *verbose {
			fmt.Printf("Verifying directory: %s\n", d)
		}
		if err := verifier.VerifyPath(d); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	}

	violations := verifier.Violations()
	if len(violations) > 0 {
		fmt.Fprintf(os.Stderr, "\n❌ Verification failed with %d violation(s):\n\n", len(violations))
		for i, v := range violations {
			fmt.Fprintf(os.Stderr, "%d) %s\n", i+1, v.String())
		}
		fmt.Fprintln(os.Stderr, "\nRun 'make verify' or 'hexag verify' again after fixing violations.")
		os.Exit(1)
	}

	fmt.Printf("✓ hexag verify: all architecture and naming rules passed (%d Go files checked)\n", verifier.FileCount())
}
