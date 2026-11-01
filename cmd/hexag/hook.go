package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Hook adapters keep agent config files thin: every agent calls `hexag hook`, so
// rule reminders and verify feedback update with the binary, not per project.
const (
	hookStopAttempts   = 3
	hookViolationLimit = 10
	hookModeReminder   = "Keep /caveman ultra and /ponytail ultra active for every response this session. Follow AGENTS.md exactly; it overrides habits from earlier turns."
)

const hookUsage = "usage: hexag hook antigravity <pre|stop> | hexag hook claude <session|prompt|edit>"

func runHook(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf(hookUsage)
	}

	switch args[0] + " " + args[1] {
	case "antigravity pre", "antigravity stop":
		return runAntigravityHook(args[1])
	case "claude session", "claude prompt", "claude edit":
		return runClaudeHook(args[1])
	default:
		return fmt.Errorf(hookUsage)
	}
}

// Antigravity drops PostToolUse output, so verify results reach the model only
// through PreInvocation injectSteps and Stop continue decisions.
func runAntigravityHook(event string) error {
	var input struct {
		ConversationID string   `json:"conversationId"`
		WorkspacePaths []string `json:"workspacePaths"`
	}
	if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
		return fmt.Errorf("failed to decode hook input: %w", err)
	}

	dir := "."
	if len(input.WorkspacePaths) > 0 {
		dir = input.WorkspacePaths[0]
	}
	report, err := hookVerifyReport(dir)
	if err != nil {
		return err
	}

	if event == "pre" {
		message := hookModeReminder
		if report != "" {
			message += "\n\n" + report
		}
		return writeHookOutput(map[string]any{
			"injectSteps": []map[string]string{{"ephemeralMessage": message}},
		})
	}

	statePath := filepath.Join(os.TempDir(), "hexag-verify-stop-"+input.ConversationID)
	if report == "" {
		if err := os.Remove(statePath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to reset stop counter: %w", err)
		}
		return writeHookOutput(map[string]string{"decision": "allow"})
	}

	attempts := 1
	if raw, err := os.ReadFile(statePath); err == nil {
		if n, err := strconv.Atoi(strings.TrimSpace(string(raw))); err == nil {
			attempts = n + 1
		}
	}
	if err := os.WriteFile(statePath, []byte(strconv.Itoa(attempts)), 0644); err != nil {
		return fmt.Errorf("failed to write stop counter: %w", err)
	}
	// ponytail: capped forced continues per failing streak so an unfixable violation cannot loop forever.
	if attempts > hookStopAttempts {
		return writeHookOutput(map[string]string{"decision": "allow"})
	}
	return writeHookOutput(map[string]string{"decision": "continue", "reason": report})
}

// Claude Code adds SessionStart/UserPromptSubmit stdout to context and feeds
// PostToolUse stderr back to the model on exit code 2.
func runClaudeHook(event string) error {
	var input struct {
		Cwd       string `json:"cwd"`
		ToolInput struct {
			FilePath string `json:"file_path"`
		} `json:"tool_input"`
	}
	if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
		return fmt.Errorf("failed to decode hook input: %w", err)
	}

	dir := os.Getenv("CLAUDE_PROJECT_DIR")
	if dir == "" {
		dir = input.Cwd
	}
	if dir == "" {
		dir = "."
	}

	switch event {
	case "session":
		// No CLAUDE.md is generated, so AGENTS.md reaches Claude through this hook.
		for _, name := range []string{"AGENTS.md", "MEMORY.md"} {
			content, err := os.ReadFile(filepath.Join(dir, name))
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				return fmt.Errorf("failed to read %s: %w", name, err)
			}
			fmt.Printf("%s\n\n", content)
		}
		fmt.Println(hookModeReminder)
		return nil

	case "prompt":
		fmt.Println(hookModeReminder)
		return nil
	}

	if !strings.HasSuffix(input.ToolInput.FilePath, ".go") {
		return nil
	}
	report, err := hookVerifyReport(dir)
	if err != nil {
		return err
	}
	if report != "" {
		fmt.Fprintln(os.Stderr, report)
		os.Exit(2)
	}
	return nil
}

// hookVerifyReport returns "" when the project passes, else a capped violation report.
func hookVerifyReport(dir string) (string, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve workspace: %w", err)
	}

	verifier, err := verifyProject(absDir, false)
	if err != nil {
		return "", err
	}

	violations := verifier.Violations()
	if len(violations) == 0 {
		return "", nil
	}

	var report strings.Builder
	fmt.Fprintf(&report, "hexag verify is failing with %d violation(s). Fix every one before the next edit (AGENTS.md Operating Loop):\n\n", len(violations))
	for i, v := range violations {
		if i == hookViolationLimit {
			fmt.Fprintf(&report, "... %d more; run make verify for the full list.\n", len(violations)-hookViolationLimit)
			break
		}
		fmt.Fprintln(&report, v.Format(i+1, len(violations)))
	}
	return report.String(), nil
}

func writeHookOutput(v any) error {
	if err := json.NewEncoder(os.Stdout).Encode(v); err != nil {
		return fmt.Errorf("failed to write hook output: %w", err)
	}
	return nil
}
