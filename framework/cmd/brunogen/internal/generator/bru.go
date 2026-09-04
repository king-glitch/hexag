package generator

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/king-glitch/hexag/framework/cmd/brunogen/internal/parser"
)

// Write renders the full Bruno collection (one .bru per route, mirroring
// the URL as a folder tree) plus bruno.json and environments/local.bru,
// into outDir. Regeneration wipes and rebuilds every route folder so a
// removed/renamed route's stale .bru file actually disappears, but never
// clobbers an existing bruno.json or environments/local.bru (a developer's
// real token lives there).
func Write(routes []parser.Route, basePath, outDir, collectionName string) error {
	brunoJSON := filepath.Join(outDir, "bruno.json")
	envFile := filepath.Join(outDir, "environments", "local.bru")

	preserved := map[string][]byte{}
	for _, f := range []string{brunoJSON, envFile} {
		if b, err := os.ReadFile(f); err == nil {
			preserved[f] = b
		}
	}

	if err := os.RemoveAll(outDir); err != nil {
		return err
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	for _, route := range routes {
		segs := kebabSegments(append(append([]string{}, route.GroupSegments...), route.PathSegments...))
		dir := filepath.Join(append([]string{outDir}, segs...)...)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		content := renderRoute(route, basePath)
		file := filepath.Join(dir, parser.ToKebabCase(route.HandlerMethod)+".bru")
		if err := os.WriteFile(file, []byte(content), 0644); err != nil {
			return err
		}
	}

	if b, ok := preserved[brunoJSON]; ok {
		if err := os.WriteFile(brunoJSON, b, 0644); err != nil {
			return err
		}
	} else if err := os.WriteFile(brunoJSON, []byte(defaultBrunoJSON(collectionName)), 0644); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Join(outDir, "environments"), 0755); err != nil {
		return err
	}
	if b, ok := preserved[envFile]; ok {
		if err := os.WriteFile(envFile, b, 0644); err != nil {
			return err
		}
	} else if err := os.WriteFile(envFile, []byte(defaultEnvironment(basePath)), 0644); err != nil {
		return err
	}

	return nil
}

func kebabSegments(segments []string) []string {
	out := make([]string, len(segments))
	for i, s := range segments {
		out[i] = parser.ToKebabCase(strings.TrimPrefix(s, ":"))
	}
	return out
}

func renderRoute(route parser.Route, basePath string) string {
	// basePath is not repeated here: {{BASE_URL}} already carries it (see
	// defaultEnvironment) — prefixing both would double it at request time.
	fullSegments := append(append([]string{}, route.GroupSegments...), route.PathSegments...)
	urlPath := "/" + strings.Join(fullSegments, "/")

	method := strings.ToLower(route.Method)
	hasBody := route.RequestBind == "body" && len(route.RequestFields) > 0
	hasQuery := (route.RequestBind == "query" || route.RequestBind == "pagination") && len(route.RequestFields) > 0
	pathParams := pathParamNames(route.PathSegments)

	queryString := ""
	if hasQuery {
		var parts []string
		for _, kv := range ExampleObject(route.RequestFields) {
			parts = append(parts, fmt.Sprintf("%s=%s", kv.Key, url.QueryEscape(fmt.Sprintf("%v", kv.Value))))
		}
		queryString = "?" + strings.Join(parts, "&")
	}

	var b strings.Builder
	fmt.Fprintf(&b, "meta {\n  name: %s\n  type: http\n  seq: 1\n}\n\n", parser.ToKebabCase(route.HandlerMethod))

	bodyMode := "none"
	if hasBody {
		bodyMode = "json"
	}
	authMode := "none"
	if route.RequiresAuth {
		authMode = "bearer"
	}
	fmt.Fprintf(&b, "%s {\n  url: {{BASE_URL}}%s%s\n  body: %s\n  auth: %s\n}\n", method, urlPath, queryString, bodyMode, authMode)

	if hasBody {
		b.WriteString("\nheaders {\n  Content-Type: application/json\n}\n")
	}

	if len(pathParams) > 0 {
		b.WriteString("\nparams:path {\n")
		for _, p := range pathParams {
			fmt.Fprintf(&b, "  %s: %s\n", p, pathParamExample(p))
		}
		b.WriteString("}\n")
	}

	if hasQuery {
		b.WriteString("\nparams:query {\n")
		for _, kv := range ExampleObject(route.RequestFields) {
			fmt.Fprintf(&b, "  %s: %v\n", kv.Key, kv.Value)
		}
		b.WriteString("}\n")
	}

	if route.RequiresAuth {
		b.WriteString("\nauth:bearer {\n  token: {{TOKEN}}\n}\n")
	}

	if hasBody {
		reqJSON, err := marshalIndent(orderedObject(ExampleObject(route.RequestFields)))
		if err == nil {
			b.WriteString("\nbody:json {\n")
			b.WriteString(indent(reqJSON, "  "))
			b.WriteString("\n}\n")
		}
	}

	docs := renderDocs(route)
	if docs != "" {
		b.WriteString("\ndocs {\n")
		b.WriteString(indent(docs, "  "))
		b.WriteString("\n}\n")
	}

	return b.String()
}

func renderDocs(route parser.Route) string {
	var b strings.Builder
	if route.Doc != "" {
		b.WriteString(strings.TrimSpace(route.Doc))
		b.WriteString("\n\n")
	}

	if len(route.ResponseFields) > 0 {
		envelope := orderedObject([]KV{{Key: "data", Value: DataValue(route.ResponseFields)}})
		resJSON, err := marshalIndent(envelope)
		if err == nil {
			b.WriteString("Response example:\n```json\n")
			b.WriteString(resJSON)
			b.WriteString("\n```")
		}
	} else {
		b.WriteString("Response example:\n```json\n{\n  \"data\": null\n}\n```")
	}

	return strings.TrimSpace(b.String())
}

func pathParamNames(segments []string) []string {
	var out []string
	for _, s := range segments {
		if strings.HasPrefix(s, ":") {
			out = append(out, strings.TrimPrefix(s, ":"))
		}
	}
	return out
}

func pathParamExample(name string) string {
	if strings.HasSuffix(strings.ToLower(name), "id") {
		return "65f1a2b3c4d5e6f7a8b9c0d1"
	}
	return "example-" + parser.ToKebabCase(name)
}

func indent(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = prefix + l
	}
	return strings.Join(lines, "\n")
}

func defaultBrunoJSON(name string) string {
	return fmt.Sprintf(`{
  "version": "1",
  "name": %q,
  "type": "collection",
  "ignore": ["node_modules", ".git"]
}
`, name)
}

func defaultEnvironment(basePath string) string {
	return fmt.Sprintf("vars {\n  BASE_URL: http://localhost:8000%s\n  TOKEN: \n}\n", basePath)
}
