package generator

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/king-glitch/hexag/framework/cmd/brunogen/internal/parser"
)

const generatorSignature = "brunogen"

type handwrittenEndpoint struct {
	method string
	path   string
	file   string
}

// Write renders the Bruno collection for routes into outDir.
// It never clobbers existing handwritten .bru files, bruno.json, collection.bru,
// folder.bru, or environments/*.bru. Only files containing the "brunogen" signature
// (or legacy "handle-*.bru" files) are treated as generator-managed and cleaned up
// when no longer present in routes.
//
// Routes whose endpoint (method + normalized path) already exists in a handwritten
// .bru file are skipped automatically so handwritten definitions take precedence.
func Write(routes []parser.Route, basePath, outDir, collectionName string) error {
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	handwritten, existingGenerated, err := scanExistingFiles(outDir)
	if err != nil {
		return err
	}

	newlyGenerated := map[string]bool{}

	for _, route := range routes {
		if hwFile, exists := findHandwrittenMatch(route, basePath, handwritten); exists {
			fullSegments := append(append([]string{}, route.GroupSegments...), route.PathSegments...)
			fmt.Printf("Skipping %s /%s (already defined in handwritten %s)\n", route.Method, strings.Join(fullSegments, "/"), hwFile)
			continue
		}

		segs := kebabSegments(append(append([]string{}, route.GroupSegments...), route.PathSegments...))
		dir := filepath.Join(append([]string{outDir}, segs...)...)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		content := renderRoute(route, basePath)
		action := actionName(route.HandlerMethod)
		file := filepath.Join(dir, action+".bru")
		if err := os.WriteFile(file, []byte(content), 0644); err != nil {
			return err
		}
		newlyGenerated[file] = true
	}

	for _, genFile := range existingGenerated {
		if !newlyGenerated[genFile] {
			_ = os.Remove(genFile)
			pruneEmptyParents(filepath.Dir(genFile), outDir)
		}
	}

	collectionFile := filepath.Join(outDir, "collection.bru")
	if _, err := os.Stat(collectionFile); os.IsNotExist(err) {
		if err := os.WriteFile(collectionFile, []byte(defaultCollectionBru()), 0644); err != nil {
			return err
		}
	}

	brunoJSON := filepath.Join(outDir, "bruno.json")
	if _, err := os.Stat(brunoJSON); os.IsNotExist(err) {
		if err := os.WriteFile(brunoJSON, []byte(defaultBrunoJSON(collectionName)), 0644); err != nil {
			return err
		}
	}

	envDir := filepath.Join(outDir, "environments")
	if err := os.MkdirAll(envDir, 0755); err != nil {
		return err
	}
	envFile := filepath.Join(envDir, "local.bru")
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		if err := os.WriteFile(envFile, []byte(defaultEnvironment(basePath)), 0644); err != nil {
			return err
		}
	}

	return nil
}

func scanExistingFiles(outDir string) ([]handwrittenEndpoint, []string, error) {
	var handwritten []handwrittenEndpoint
	var existingGenerated []string

	err := filepath.Walk(outDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(outDir, path)
		if err != nil {
			return nil
		}

		if isPreservedSpecialFile(rel) {
			return nil
		}

		if !strings.HasSuffix(path, ".bru") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		filename := filepath.Base(path)
		if isGeneratedFile(content, filename) {
			existingGenerated = append(existingGenerated, path)
		} else {
			method, rawURL, ok := parseBruEndpoint(string(content))
			if ok {
				handwritten = append(handwritten, handwrittenEndpoint{
					method: method,
					path:   rawURL,
					file:   rel,
				})
			}
		}

		return nil
	})

	return handwritten, existingGenerated, err
}

func isPreservedSpecialFile(relPath string) bool {
	slashPath := filepath.ToSlash(relPath)
	if slashPath == "bruno.json" || slashPath == "collection.bru" {
		return true
	}
	if filepath.Base(slashPath) == "folder.bru" {
		return true
	}
	parts := strings.Split(slashPath, "/")
	if len(parts) > 0 && parts[0] == "environments" {
		return true
	}
	return false
}

func isGeneratedFile(content []byte, filename string) bool {
	return strings.Contains(string(content), generatorSignature) || strings.HasPrefix(filename, "handle-")
}

func actionName(handlerMethod string) string {
	kebab := parser.ToKebabCase(handlerMethod)
	if stripped := strings.TrimPrefix(kebab, "handle-"); stripped != "" {
		return stripped
	}
	return kebab
}

func parseBruEndpoint(content string) (string, string, bool) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	var currentBlock string
	httpMethods := map[string]bool{
		"get":     true,
		"post":    true,
		"put":     true,
		"delete":  true,
		"patch":   true,
		"options": true,
		"head":    true,
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasSuffix(line, "{") {
			name := strings.ToLower(strings.TrimSpace(strings.TrimSuffix(line, "{")))
			if httpMethods[name] {
				currentBlock = strings.ToUpper(name)
			} else {
				currentBlock = ""
			}
			continue
		}
		if line == "}" {
			currentBlock = ""
			continue
		}
		if currentBlock != "" && strings.HasPrefix(line, "url:") {
			rawURL := strings.TrimSpace(strings.TrimPrefix(line, "url:"))
			return currentBlock, rawURL, true
		}
	}
	return "", "", false
}

func normalizeURLPath(rawURL, basePath string) string {
	rawURL = strings.TrimSpace(rawURL)
	if idx := strings.Index(rawURL, "?"); idx != -1 {
		rawURL = rawURL[:idx]
	}
	if idx := strings.Index(rawURL, "#"); idx != -1 {
		rawURL = rawURL[:idx]
	}
	for {
		start := strings.Index(rawURL, "{{")
		if start == -1 {
			break
		}
		end := strings.Index(rawURL, "}}")
		if end == -1 || end < start {
			break
		}
		rawURL = rawURL[:start] + rawURL[end+2:]
	}
	if u, err := url.Parse(rawURL); err == nil && u.Path != "" {
		rawURL = u.Path
	}
	basePath = "/" + strings.Trim(basePath, "/")
	if basePath != "/" && strings.HasPrefix(rawURL, basePath) {
		rawURL = strings.TrimPrefix(rawURL, basePath)
	}
	rawURL = "/" + strings.Trim(rawURL, "/")
	return rawURL
}

func findHandwrittenMatch(route parser.Route, basePath string, handwritten []handwrittenEndpoint) (string, bool) {
	fullSegments := append(append([]string{}, route.GroupSegments...), route.PathSegments...)
	routeRelPath := "/" + strings.Join(fullSegments, "/")

	for _, hw := range handwritten {
		if matchEndpoint(route.Method, routeRelPath, basePath, hw.method, hw.path) {
			return hw.file, true
		}
	}
	return "", false
}

func matchEndpoint(routeMethod, routeRelPath, basePath, hwMethod, hwRawURL string) bool {
	if !strings.EqualFold(routeMethod, hwMethod) {
		return false
	}
	hwNorm := normalizeURLPath(hwRawURL, basePath)
	rNorm := normalizeURLPath(routeRelPath, basePath)

	if rNorm == hwNorm {
		return true
	}

	rSegs := splitSegments(rNorm)
	hSegs := splitSegments(hwNorm)
	if len(rSegs) != len(hSegs) {
		return false
	}
	for i := range rSegs {
		r := rSegs[i]
		h := hSegs[i]
		if strings.HasPrefix(r, ":") || strings.HasPrefix(h, ":") {
			continue
		}
		if !strings.EqualFold(r, h) {
			return false
		}
	}
	return true
}

func splitSegments(p string) []string {
	parts := strings.Split(strings.Trim(p, "/"), "/")
	var out []string
	for _, part := range parts {
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func pruneEmptyParents(startDir, stopDir string) {
	cur := startDir
	for cur != stopDir && cur != "." && cur != "/" {
		entries, err := os.ReadDir(cur)
		if err != nil || len(entries) > 0 {
			break
		}
		_ = os.Remove(cur)
		cur = filepath.Dir(cur)
	}
}

// MatchesExclude reports whether a route matches any wildcard, regex, or substring pattern.
func MatchesExclude(route parser.Route, basePath string, patterns []string) bool {
	fullSegments := append(append([]string{}, route.GroupSegments...), route.PathSegments...)
	relPath := "/" + strings.Join(fullSegments, "/")
	fullPath := basePath + relPath
	groupPath := strings.Join(route.GroupSegments, "/")
	methodPath := route.Method + " " + fullPath

	targets := []string{fullPath, relPath, groupPath, methodPath}

	for _, pat := range patterns {
		for _, target := range targets {
			if matchPattern(pat, target) {
				return true
			}
		}
	}
	return false
}

func matchPattern(pattern, target string) bool {
	if pattern == target {
		return true
	}
	if strings.Contains(pattern, "*") {
		quoted := regexp.QuoteMeta(pattern)
		regexStr := "^" + strings.ReplaceAll(quoted, `\*`, ".*") + "$"
		if re, err := regexp.Compile(regexStr); err == nil {
			if re.MatchString(target) {
				return true
			}
		}
	}
	if re, err := regexp.Compile(pattern); err == nil {
		if re.MatchString(target) {
			return true
		}
	}
	if strings.Contains(target, pattern) {
		return true
	}
	return false
}

func kebabSegments(segments []string) []string {
	out := make([]string, len(segments))
	for i, s := range segments {
		out[i] = parser.ToKebabCase(strings.TrimPrefix(s, ":"))
	}
	return out
}

func renderRoute(route parser.Route, basePath string) string {
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
	action := actionName(route.HandlerMethod)
	fmt.Fprintf(&b, "meta {\n  name: %s\n  type: http\n  seq: 1\n  tags: [\n    brunogen\n  ]\n}\n\n", action)

	bodyMode := "none"
	if hasBody {
		bodyMode = "json"
	}
	fmt.Fprintf(&b, "%s {\n  url: {{BASE_URL}}%s%s\n  body: %s\n  auth: inherit\n}\n", method, urlPath, queryString, bodyMode)

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
	b.WriteString("Generated by brunogen. DO NOT EDIT.\n\n")

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

func defaultCollectionBru() string {
	return "auth {\n  mode: bearer\n}\n\nauth:bearer {\n  token: {{TOKEN}}\n}\n"
}

