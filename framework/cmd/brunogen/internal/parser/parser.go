package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var httpMethods = map[string]bool{
	"Get": true, "Post": true, "Put": true, "Patch": true,
	"Delete": true, "Options": true, "Head": true,
}

// Most handlers follow AGENTS.md's `<MethodName>Request`/`<MethodName>Response`
// convention exactly, but a few (e.g. property.go's Create/List) name the
// type after the resource instead of the literal Go method name. These
// patterns pull the real type name straight out of the handler body as a
// fallback when the naming convention doesn't hold.
var (
	reVarReq      = regexp.MustCompile(`\bvar\s+req\s+(\w+)\b`)
	reSuccessWrap = regexp.MustCompile(`NewSuccessResponse\(\s*(\w+)\(`)
	reResLiteral  = regexp.MustCompile(`\bres\s*:=\s*(\w+)\{`)
	reResVarDecl  = regexp.MustCompile(`\bvar\s+res\s+(\w+)\b`)
)

// Parse discovers every route wired into routesDir's Register/Register*
// methods that is actually mounted per compositionFile, resolving each
// route's request/response example fields against the project's ports
// package (portsDir).
func Parse(routesDir, compositionFile, portsDir, basePath string) ([]Route, error) {
	prefixes, err := parseComposition(compositionFile)
	if err != nil {
		return nil, err
	}

	external, err := buildRegistry(portsDir)
	if err != nil {
		return nil, fmt.Errorf("build ports registry: %w", err)
	}

	// All route files share one `package routes` scope — a type declared in
	// identity.go (e.g. PaginationResponse) is visible from property.go too —
	// so field resolution needs one registry built across the whole
	// directory, not a fresh one per file.
	pkgTypes, err := buildRegistry(routesDir)
	if err != nil {
		return nil, fmt.Errorf("build routes package registry: %w", err)
	}

	entries, err := os.ReadDir(routesDir)
	if err != nil {
		return nil, err
	}

	var routes []Route
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		fileRoutes, err := parseRouteFile(filepath.Join(routesDir, name), name, prefixes, pkgTypes, external)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", name, err)
		}
		routes = append(routes, fileRoutes...)
	}

	sort.Slice(routes, func(i, j int) bool {
		return strings.Join(append(routes[i].GroupSegments, routes[i].PathSegments...), "/") <
			strings.Join(append(routes[j].GroupSegments, routes[j].PathSegments...), "/")
	})

	_ = basePath // basePath is applied by the generator, kept here for signature symmetry
	return routes, nil
}

type groupCtx struct {
	segments  []string
	needsAuth bool
}

func parseRouteFile(path, baseName string, prefixes registerPrefixes, pkgTypes, external typeRegistry) ([]Route, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	// fileTypes resolves a handler's own Request/Response entry-point type
	// name (scoped to this file, so e.g. "ListResponse" in notification.go
	// can never accidentally satisfy property.go's List handler); nested
	// resolves struct fields, which legitimately span the whole package via
	// ordinary Go scoping (e.g. PaginationResponse, declared once).
	fileTypes := registryFromFile(node)
	nested := registryChain{fileTypes, pkgTypes}

	funcsByName := map[string]*ast.FuncDecl{}
	var registerFuncs []*ast.FuncDecl
	for _, decl := range node.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || len(fn.Recv.List) == 0 {
			continue
		}
		funcsByName[fn.Name.Name] = fn
		if strings.HasPrefix(fn.Name.Name, "Register") {
			registerFuncs = append(registerFuncs, fn)
		}
	}

	var routes []Route
	for _, fn := range registerFuncs {
		handlerType := identOrSelectorName(fn.Recv.List[0].Type)
		prefixStr, ok := prefixes[handlerType][fn.Name.Name]
		if !ok {
			continue // Register* method exists but isn't wired into the app; nothing to document
		}
		if fn.Type.Params == nil || len(fn.Type.Params.List) == 0 || len(fn.Type.Params.List[0].Names) == 0 {
			continue
		}
		routerParam := fn.Type.Params.List[0].Names[0].Name

		groups := map[string]groupCtx{
			routerParam: {segments: splitPath(prefixStr)},
		}

		for _, stmt := range fn.Body.List {
			collected := walkRegisterStmt(stmt, groups, handlerType, fn.Name.Name, baseName, funcsByName, fileTypes, nested, external, fset, src)
			routes = append(routes, collected...)
		}
	}

	return routes, nil
}

// walkRegisterStmt handles one top-level statement of a Register* method
// body: either a `x := y.Group(...)` group declaration, or a
// `x.Method(path, ...mw, handler)` route registration.
func walkRegisterStmt(
	stmt ast.Stmt,
	groups map[string]groupCtx,
	handlerType, registerMethod, handlerFile string,
	funcsByName map[string]*ast.FuncDecl,
	fileTypes typeRegistry, nested registryChain, external typeRegistry,
	fset *token.FileSet, src []byte,
) []Route {
	switch s := stmt.(type) {
	case *ast.AssignStmt:
		if len(s.Lhs) != 1 || len(s.Rhs) != 1 {
			return nil
		}
		lhs, ok := s.Lhs[0].(*ast.Ident)
		if !ok {
			return nil
		}
		call, ok := s.Rhs[0].(*ast.CallExpr)
		if !ok {
			return nil
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Group" {
			return nil
		}
		recv, ok := sel.X.(*ast.Ident)
		if !ok {
			return nil
		}
		parent, ok := groups[recv.Name]
		if !ok || len(call.Args) == 0 {
			return nil
		}
		lit, ok := stringLit(call.Args[0])
		if !ok {
			return nil
		}
		groups[lhs.Name] = groupCtx{
			segments:  append(append([]string{}, parent.segments...), splitPath(lit)...),
			needsAuth: parent.needsAuth || len(call.Args) > 1,
		}
		return nil

	case *ast.ExprStmt:
		call, ok := s.X.(*ast.CallExpr)
		if !ok {
			return nil
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || !httpMethods[sel.Sel.Name] {
			return nil
		}
		recv, ok := sel.X.(*ast.Ident)
		if !ok {
			return nil
		}
		parent, ok := groups[recv.Name]
		if !ok || len(call.Args) < 2 {
			return nil
		}

		pathLit, ok := stringLit(call.Args[0])
		if !ok {
			return nil
		}
		handlerSel, ok := call.Args[len(call.Args)-1].(*ast.SelectorExpr)
		if !ok {
			return nil // inline func literal or similar — not a documentable named handler
		}
		handlerMethod := handlerSel.Sel.Name
		extraMiddleware := len(call.Args) - 2 // args between path and handler

		fd, hasDoc := funcsByName[handlerMethod]
		bindKind := "none"
		var doc, body string
		if hasDoc {
			if fd.Doc != nil {
				doc = strings.TrimSpace(fd.Doc.Text())
			}
			body = funcBodyText(fset, src, fd)
			bindKind = detectBindKind(body)
		}

		reqTypeName := handlerMethod + "Request"
		if _, ok := fileTypes.resolveStruct(reqTypeName); !ok {
			if m := reVarReq.FindStringSubmatch(body); m != nil {
				reqTypeName = m[1]
			}
		}
		reqFields := resolveNamed(fileTypes, nested, external, reqTypeName, bindTagKey(bindKind))
		if bindKind == "pagination" && reqFields == nil {
			reqFields = []Field{{Key: "page", Kind: "int"}, {Key: "amount", Kind: "int"}}
		}

		resTypeName := handlerMethod + "Response"
		if _, ok := fileTypes.resolveStruct(resTypeName); !ok {
			resTypeName = fallbackResponseType(body)
		}
		resFields := resolveNamed(fileTypes, nested, external, resTypeName, "json")

		route := Route{
			Method:         strings.ToUpper(sel.Sel.Name),
			GroupSegments:  parent.segments,
			PathSegments:   splitPath(pathLit),
			HandlerFile:    handlerFile,
			HandlerType:    handlerType,
			RegisterMethod: registerMethod,
			HandlerMethod:  handlerMethod,
			Doc:            doc,
			RequiresAuth:   parent.needsAuth || extraMiddleware > 0,
			RequestBind:    bindKind,
			RequestFields:  reqFields,
			ResponseFields: resFields,
		}
		return []Route{route}
	}
	return nil
}

// resolveNamed looks up typeName strictly within fileTypes (so a generic
// entry-point name can't accidentally match an unrelated file), but expands
// its fields via nested (this file + the whole routes package), since
// nested struct fields legitimately reference types declared elsewhere in
// the same Go package.
//
// A handful of responses are defined as a bare slice rather than a struct
// (`type ListRoomTypesResponse []RoomTypeResponse`) — the whole body is an
// array, not an object. That's represented as a single Field with an empty
// Key (never a legitimate JSON key on its own), which the generator treats
// as "this Field IS the response body" instead of one of several fields.
func resolveNamed(fileTypes typeRegistry, nested registryChain, external typeRegistry, typeName, tagKey string) []Field {
	if arr, ok := fileTypes[typeName].(*ast.ArrayType); ok {
		item := Field{}
		resolveType(arr.Elt, nested, external, tagKey, 1, &item)
		return []Field{{Kind: "array", SubFields: []Field{item}}}
	}

	st, ok := fileTypes.resolveStruct(typeName)
	if !ok {
		return nil
	}
	return resolveFields(st, nested, external, tagKey, 0)
}

func bindTagKey(bindKind string) string {
	if bindKind == "query" {
		return "query"
	}
	return "json"
}

// funcBodyText slices the handler method's source text — used both to
// detect how its request struct is bound and, when the naming convention
// doesn't hold, to recover the actual request/response type names.
func funcBodyText(fset *token.FileSet, src []byte, fd *ast.FuncDecl) string {
	if fd.Body == nil {
		return ""
	}
	start := fset.Position(fd.Body.Pos()).Offset
	end := fset.Position(fd.Body.End()).Offset
	if start < 0 || end > len(src) || start >= end {
		return ""
	}
	return string(src[start:end])
}

// detectBindKind spots the well-known transport calls in a handler body —
// simplest robust signal for how its request struct (if any) is populated.
func detectBindKind(body string) string {
	switch {
	case strings.Contains(body, ".Body(&req)"):
		return "body"
	case strings.Contains(body, "BindQuery(c, &req)") || strings.Contains(body, "BindQuery(c, &req"):
		return "query"
	case strings.Contains(body, "ParsePaginationParamsContext("):
		return "pagination"
	default:
		return "none"
	}
}

// fallbackResponseType recovers the real response struct name when a
// handler doesn't follow the `<MethodName>Response` convention: either a
// type conversion wrapping the success response (`NewSuccessResponse(XResponse(...))`)
// or a `res := XResponse{...}` / `var res XResponse` local variable.
func fallbackResponseType(body string) string {
	if m := reSuccessWrap.FindStringSubmatch(body); m != nil {
		return m[1]
	}
	if m := reResLiteral.FindStringSubmatch(body); m != nil {
		return m[1]
	}
	if m := reResVarDecl.FindStringSubmatch(body); m != nil {
		return m[1]
	}
	return ""
}

func stringLit(e ast.Expr) (string, bool) {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	return strings.Trim(lit.Value, "\"`"), true
}

func splitPath(p string) []string {
	p = strings.Trim(p, "/")
	if p == "" {
		return nil
	}
	return strings.Split(p, "/")
}
