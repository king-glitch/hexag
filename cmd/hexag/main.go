package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
)

const version = "v0.0.23"

func usage() {
	fmt.Println("hexag — hexagonal architecture toolkit and AI agent verifier")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  hexag <command> [arguments]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  verify      Verify code adherence to strict AGENTS.md architecture & naming rules")
	fmt.Println("  bruno       Generate Bruno API collections and Markdown API reference")
	fmt.Println("  mongo       Generate Mongo fields and standalone models")
	fmt.Println("  new         Scaffold a new hexagonal project from the official template")
	fmt.Println("  update      Update AGENTS.md, CLAUDE.md, and configuration in an existing project")
	fmt.Println("  completion  Generate shell autocompletion script (zsh, bash)")
	fmt.Println("  version     Show hexag version")
	fmt.Println()
	fmt.Println("Run 'hexag <command> --help' for details on a specific command.")
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(0)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	var err error
	switch cmd {
	case "verify":
		err = runVerify(args)
	case "bruno":
		err = runBruno(args)
	case "mongo":
		err = runMongo(args)
	case "new":
		err = runNew(args)
	case "update":
		err = runUpdate(args)
	case "completion":
		err = runCompletion(args)
	case "version", "-v", "--version":
		fmt.Printf("hexag %s\n", version)
		return
	case "help", "-h", "--help":
		usage()
		return
	default:
		fmt.Fprintf(os.Stderr, "hexag: unknown command '%s'\n\n", cmd)
		usage()
		os.Exit(1)
	}

	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		fmt.Fprintf(os.Stderr, "hexag error: %v\n", err)
		os.Exit(1)
	}
}
