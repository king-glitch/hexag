package parser

import (
	"go/ast"
	"reflect"
	"strings"
)

// resolveFields walks a struct's fields recursively into example-able Field
// metadata. tagKey is "json" or "query" depending on how the request is
// bound; local is the current file's type registry (for nested/embedded
// sibling types), external is the project's ports package registry (for
// fields typed `ports.X`). Depth-limited against accidental cycles.
func resolveFields(st *ast.StructType, local, external resolver, tagKey string, depth int) []Field {
	if st.Fields == nil || depth > 6 {
		return nil
	}

	var fields []Field
	for _, f := range st.Fields.List {
		if len(f.Names) == 0 {
			fields = append(fields, resolveEmbedded(f, local, external, tagKey, depth)...)
			continue
		}

		for _, nameIdent := range f.Names {
			key, ok := tagKeyValue(f.Tag, tagKey, nameIdent.Name)
			if !ok {
				continue
			}

			field := Field{Key: key}
			applyValidate(f.Tag, &field)
			resolveType(f.Type, local, external, tagKey, depth, &field)
			fields = append(fields, field)
		}
	}
	return fields
}

// resolveEmbedded handles an embedded (anonymous) field: a local struct
// (flattened in, matching mongogen's embed handling), a resolvable
// `ports.X` struct (flattened in), a `hexports.Collection[T]` (special-cased
// via resolveCollection), or anything else unresolvable (silently skipped —
// the example is best-effort, not a strict contract).
func resolveEmbedded(f *ast.Field, local, external resolver, tagKey string, depth int) []Field {
	switch t := f.Type.(type) {
	case *ast.Ident:
		if st, ok := local.resolveStruct(t.Name); ok {
			return resolveFields(st, local, external, tagKey, depth+1)
		}
	case *ast.SelectorExpr:
		if st, ok := external.resolveStruct(t.Sel.Name); ok {
			return resolveFields(st, external, external, tagKey, depth+1)
		}
	case *ast.IndexExpr, *ast.IndexListExpr:
		// Embedded (anonymous) `hexports.Collection[T]` flattens its own
		// collection/count/meta keys straight into the parent, same as any
		// other embed — unlike a *named* Collection[T] field, which keeps
		// its own key (handled in resolveType instead).
		if collection, ok := resolveCollection(f.Type, external); ok {
			return collection.SubFields
		}
	}
	return nil
}

// resolveType fills in field.Kind (and SubFields for objects/arrays) from a
// Go type expression.
func resolveType(expr ast.Expr, local, external resolver, tagKey string, depth int, field *Field) {
	switch t := expr.(type) {
	case *ast.StarExpr:
		field.Required = false
		resolveType(t.X, local, external, tagKey, depth, field)

	case *ast.Ident:
		if kind, ok := builtinKind(t.Name); ok {
			field.Kind = kind
			return
		}
		if st, ok := local.resolveStruct(t.Name); ok {
			field.Kind = "object"
			field.SubFields = resolveFields(st, local, external, tagKey, depth+1)
			return
		}
		field.Kind = "string" // unresolved local named type (e.g. a DTO never uses real enum types directly)

	case *ast.SelectorExpr:
		pkg, _ := t.X.(*ast.Ident)
		pkgName := ""
		if pkg != nil {
			pkgName = pkg.Name
		}
		switch {
		case (pkgName == "bson" || pkgName == "primitive") && t.Sel.Name == "ObjectID":
			field.Kind = "objectid"
		case pkgName == "time" && t.Sel.Name == "Time":
			field.Kind = "time"
		case pkgName == "ports":
			if st, ok := external.resolveStruct(t.Sel.Name); ok {
				field.Kind = "object"
				field.SubFields = resolveFields(st, external, external, tagKey, depth+1)
				return
			}
			field.Kind = "string" // Satang, typed enums, or anything else unresolved
		default:
			field.Kind = "string"
		}

	case *ast.ArrayType:
		field.Kind = "array"
		item := Field{}
		resolveType(t.Elt, local, external, tagKey, depth+1, &item)
		field.SubFields = []Field{item}

	case *ast.MapType:
		field.Kind = "map"
		item := Field{Key: "key"}
		resolveType(t.Value, local, external, tagKey, depth+1, &item)
		field.SubFields = []Field{item}

	case *ast.IndexExpr, *ast.IndexListExpr:
		if collection, ok := resolveCollection(expr, external); ok {
			*field = collection
			return
		}
		field.Kind = "object"

	default:
		field.Kind = "string"
	}
}

// resolveCollection special-cases `hexports.Collection[T]`, whose shape is
// framework-fixed ({collection: [T], count, meta: {total, total_pages}}),
// rather than trying to resolve arbitrary Go generic instantiations.
func resolveCollection(expr ast.Expr, external resolver) (Field, bool) {
	var indexExpr ast.Expr
	var base ast.Expr
	switch t := expr.(type) {
	case *ast.IndexExpr:
		base, indexExpr = t.X, t.Index
	case *ast.IndexListExpr:
		if len(t.Indices) != 1 {
			return Field{}, false
		}
		base, indexExpr = t.X, t.Indices[0]
	default:
		return Field{}, false
	}

	sel, ok := base.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Collection" {
		return Field{}, false
	}

	item := Field{}
	resolveType(indexExpr, external, external, "json", 1, &item)

	return Field{
		Kind: "collection",
		SubFields: []Field{
			{Key: "collection", Kind: "array", SubFields: []Field{item}},
			{Key: "count", Kind: "int"},
			{Key: "meta", Kind: "object", SubFields: []Field{
				{Key: "total", Kind: "int"},
				{Key: "total_pages", Kind: "int"},
			}},
		},
	}, true
}

func builtinKind(name string) (string, bool) {
	switch name {
	case "string":
		return "string", true
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64":
		return "int", true
	case "float32", "float64":
		return "float", true
	case "bool":
		return "bool", true
	default:
		return "", false
	}
}

func tagKeyValue(tag *ast.BasicLit, tagKey, fallbackName string) (string, bool) {
	if tag == nil {
		return ToSnakeCase(fallbackName), true
	}
	tagValue := reflect.StructTag(strings.Trim(tag.Value, "`")).Get(tagKey)
	if tagValue == "-" {
		return "", false
	}
	if tagValue == "" {
		return ToSnakeCase(fallbackName), true
	}
	parts := strings.Split(tagValue, ",")
	if parts[0] == "" {
		return ToSnakeCase(fallbackName), true
	}
	return parts[0], true
}

func applyValidate(tag *ast.BasicLit, field *Field) {
	if tag == nil {
		return
	}
	v := reflect.StructTag(strings.Trim(tag.Value, "`")).Get("validate")
	if v == "" {
		return
	}
	for _, rule := range strings.Split(v, ",") {
		switch {
		case rule == "required":
			field.Required = true
		case rule == "email":
			field.Email = true
		case strings.HasPrefix(rule, "oneof="):
			field.OneOf = strings.Fields(strings.TrimPrefix(rule, "oneof="))
		}
	}
}
