package parser

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// typeRegistry is every top-level `type X ...` declaration seen across one or
// more files, keyed by bare type name. Used to resolve local aliases
// (`type Foo Bar`) and nested struct fields without a full go/packages
// compile.
type typeRegistry map[string]ast.Expr

// buildRegistry parses every non-test .go file directly inside dir (no
// recursion) and records each type declaration's RHS expression.
func buildRegistry(dir string) (typeRegistry, error) {
	reg := typeRegistry{}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		node, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, 0)
		if err != nil {
			return nil, err
		}

		for _, decl := range node.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				continue
			}
			for _, spec := range genDecl.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				reg[ts.Name.Name] = ts.Type
			}
		}
	}

	return reg, nil
}

// resolveStruct follows `type Foo Bar` alias chains (bare identifiers only,
// never package-qualified — those point outside this registry) until it
// finds a literal struct type, or gives up after a few hops.
func (r typeRegistry) resolveStruct(name string) (*ast.StructType, bool) {
	return registryChain{r}.resolveStruct(name)
}

// resolver is implemented by both a single typeRegistry and a
// registryChain, so field resolution can be handed either a flat map (the
// ports package) or a prioritized fallback chain (a route file's own types
// first, then the whole routes package) without caring which.
type resolver interface {
	resolveStruct(name string) (*ast.StructType, bool)
}

// registryChain resolves a name against each registry in order, so a
// generic entry-point type name (e.g. "ListResponse") is matched against
// the current file before it can accidentally collide with an unrelated
// file's type of the same name — but a nested field type declared in
// another file of the same package (ordinary Go package scoping) still
// resolves via the later registries in the chain.
type registryChain []typeRegistry

// registryFromFile builds a typeRegistry from an already-parsed file's type
// declarations, without touching disk again — used as the highest-priority
// (most specific) member of a registryChain.
func registryFromFile(node *ast.File) typeRegistry {
	reg := typeRegistry{}
	for _, decl := range node.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}
		for _, spec := range genDecl.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			reg[ts.Name.Name] = ts.Type
		}
	}
	return reg
}

func (c registryChain) resolveStruct(name string) (*ast.StructType, bool) {
	seen := map[string]bool{}
	for i := 0; i < 8; i++ {
		if seen[name] {
			return nil, false
		}
		seen[name] = true

		var expr ast.Expr
		found := false
		for _, r := range c {
			if e, ok := r[name]; ok {
				expr, found = e, true
				break
			}
		}
		if !found {
			return nil, false
		}

		switch t := expr.(type) {
		case *ast.StructType:
			return t, true
		case *ast.Ident:
			name = t.Name
			continue
		default:
			return nil, false
		}
	}
	return nil, false
}
