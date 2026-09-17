package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/king-glitch/hexag/framework/cmd/brunogen/internal/parser"
)

func TestActionName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"HandleStart", "start"},
		{"HandleGetActive", "get-active"},
		{"HandleCheckDrops", "check-drops"},
		{"HandleClaimDrop", "claim-drop"},
		{"HandleOpenCache", "open-cache"},
		{"HandleGetMe", "get-me"},
		{"Handle", "handle"},
		{"GetMe", "get-me"},
	}

	for _, tt := range tests {
		got := actionName(tt.input)
		if got != tt.expected {
			t.Errorf("actionName(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}

func TestMatchesExclude(t *testing.T) {
	route := parser.Route{
		Method:        "POST",
		GroupSegments: []string{"authentication"},
		PathSegments:  []string{"login"},
	}

	tests := []struct {
		pattern string
		matches bool
	}{
		{"/api/v1/authentication/*", true},
		{"*authentication*", true},
		{"authentication", true},
		{"/authentication/login", true},
		{"POST /api/v1/authentication/login", true},
		{"/api/v1/expedition/*", false},
		{"*expedition*", false},
	}

	for _, tt := range tests {
		got := MatchesExclude(route, "/api/v1", []string{tt.pattern})
		if got != tt.matches {
			t.Errorf("MatchesExclude(pattern=%q) = %v; want %v", tt.pattern, got, tt.matches)
		}
	}
}

func TestNormalizeURLPath(t *testing.T) {
	tests := []struct {
		rawURL   string
		basePath string
		expected string
	}{
		{"{{BASE_URL}}/archive/manifest", "/api/v1", "/archive/manifest"},
		{"{{BASE_URL}}/archive?type=stat", "/api/v1", "/archive"},
		{"http://localhost:8000/api/v1/archive/releases", "/api/v1", "/archive/releases"},
		{"/api/v1/authentication/login", "/api/v1", "/authentication/login"},
		{"{{BASE_URL}}/archive/", "/api/v1", "/archive"},
	}

	for _, tt := range tests {
		got := normalizeURLPath(tt.rawURL, tt.basePath)
		if got != tt.expected {
			t.Errorf("normalizeURLPath(%q, %q) = %q; want %q", tt.rawURL, tt.basePath, got, tt.expected)
		}
	}
}

func TestWrite_PreservesHandwrittenAndDeletesStale(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Create a handwritten .bru file for POST /authentication/login
	authDir := filepath.Join(tmpDir, "authentication")
	if err := os.MkdirAll(authDir, 0755); err != nil {
		t.Fatal(err)
	}
	hwLoginFile := filepath.Join(authDir, "login.bru")
	hwContent := `meta {
  name: Login
  type: http
  seq: 1
}

post {
  url: {{BASE_URL}}/authentication/login
  body: json
  auth: none
}
`
	if err := os.WriteFile(hwLoginFile, []byte(hwContent), 0644); err != nil {
		t.Fatal(err)
	}

	// 2. Create collection.bru and environments/dev.bru
	if err := os.WriteFile(filepath.Join(tmpDir, "collection.bru"), []byte("auth: bearer\n"), 0644); err != nil {
		t.Fatal(err)
	}
	envDir := filepath.Join(tmpDir, "environments")
	if err := os.MkdirAll(envDir, 0755); err != nil {
		t.Fatal(err)
	}
	devEnvFile := filepath.Join(envDir, "dev.bru")
	if err := os.WriteFile(devEnvFile, []byte("vars { BASE_URL: https://dev.api }\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// 3. Create a stale legacy generated file: handle-old.bru with handle- prefix
	oldDir := filepath.Join(tmpDir, "old")
	if err := os.MkdirAll(oldDir, 0755); err != nil {
		t.Fatal(err)
	}
	oldFile := filepath.Join(oldDir, "handle-old.bru")
	if err := os.WriteFile(oldFile, []byte("meta {\n  name: handle-old\n}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Routes to generate:
	// - POST /authentication/login (should be skipped because handwritten file exists!)
	// - GET /expedition/entries/active (should be generated as get-active.bru)
	routes := []parser.Route{
		{
			Method:        "POST",
			GroupSegments: []string{"authentication"},
			PathSegments:  []string{"login"},
			HandlerMethod: "HandleLogin",
		},
		{
			Method:        "GET",
			GroupSegments: []string{"expedition", "entries"},
			PathSegments:  []string{"active"},
			HandlerMethod: "HandleGetActive",
		},
		{
			Method:        "POST",
			GroupSegments: []string{"character"},
			PathSegments:  []string{"characters", ":character_id", "skills", ":skill_id", "equip"},
			HandlerMethod: "HandleEquipSkill",
		},
		{
			Method:        "POST",
			GroupSegments: []string{"character"},
			PathSegments:  []string{"characters", ":character_id", "skills", ":skill_id", "unequip"},
			HandlerMethod: "HandleUnequipSkill",
		},
	}

	if err := Write(routes, "/api/v1", tmpDir, "test-api"); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Verify handwritten login.bru was untouched
	data, err := os.ReadFile(hwLoginFile)
	if err != nil {
		t.Fatalf("handwritten file was deleted: %v", err)
	}
	if string(data) != hwContent {
		t.Fatalf("handwritten file was overwritten")
	}

	// Verify collection.bru and dev.bru were preserved
	if _, err := os.Stat(filepath.Join(tmpDir, "collection.bru")); err != nil {
		t.Errorf("collection.bru was deleted: %v", err)
	}
	if _, err := os.Stat(devEnvFile); err != nil {
		t.Errorf("dev.bru was deleted: %v", err)
	}

	// Verify stale handle-old.bru was removed
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Errorf("stale file handle-old.bru was NOT deleted")
	}

	// Verify generated file is in parent folder: expedition/entries/get-active.bru (not entries/active/...)
	genFile := filepath.Join(tmpDir, "expedition", "entries", "get-active.bru")
	genContent, err := os.ReadFile(genFile)
	if err != nil {
		t.Fatalf("expected generated file %s, err: %v", genFile, err)
	}
	genStr := string(genContent)
	if !strings.Contains(genStr, "name: get-active") {
		t.Errorf("expected meta name 'get-active', got:\n%s", genStr)
	}
	if !strings.Contains(genStr, "brunogen") {
		t.Errorf("expected brunogen signature in file, got:\n%s", genStr)
	}
	if !strings.Contains(genStr, "auth: inherit") {
		t.Errorf("expected 'auth: inherit' in file, got:\n%s", genStr)
	}
	if strings.Contains(genStr, "auth:bearer") {
		t.Errorf("expected no per-request auth:bearer block in file, got:\n%s", genStr)
	}

	// Verify equip and unequip share the same folder under :skill_id
	skillDir := filepath.Join(tmpDir, "character", "characters", "character_id", "skills", "skill_id")
	equipFile := filepath.Join(skillDir, "equip-skill.bru")
	unequipFile := filepath.Join(skillDir, "unequip-skill.bru")
	if _, err := os.Stat(equipFile); err != nil {
		t.Errorf("expected equip-skill.bru in %s, err: %v", skillDir, err)
	}
	if _, err := os.Stat(unequipFile); err != nil {
		t.Errorf("expected unequip-skill.bru in %s, err: %v", skillDir, err)
	}
	// Verify no redundant equip/ and unequip/ folders exist
	if _, err := os.Stat(filepath.Join(skillDir, "equip")); !os.IsNotExist(err) {
		t.Errorf("redundant folder equip/ should not exist under %s", skillDir)
	}
	if _, err := os.Stat(filepath.Join(skillDir, "unequip")); !os.IsNotExist(err) {
		t.Errorf("redundant folder unequip/ should not exist under %s", skillDir)
	}
}
