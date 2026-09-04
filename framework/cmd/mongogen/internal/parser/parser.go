package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"reflect"
	"regexp"
	"strings"
	"unicode"
)

type StructMeta struct {
	PackageName string
	StructName  string
	VarName     string
	FileName    string
	Imports     []string
	Fields      []FieldMeta
}

type FieldMeta struct {
	GoName    string
	GoType    string
	BsonKey   string
	FieldType string
	SubFields []FieldMeta
	// FieldPkg is the qualifier prefix for Field[T]/NewField (e.g. "mongo."
	// in shared mode, "" in standalone mode). Set by the generator after
	// parsing, uniformly across the whole field tree, since text/template's
	// `$` resets on every {{template}} call and can't carry it through
	// recursive nested-field rendering.
	FieldPkg string
}

type Parser struct {
	targetPackage string
	// portsImport is the import path substituted for a bare custom-type
	// reference (e.g. a field typed `BankTransactionDirection`) that lives
	// in the caller's own ports package, not in the file being parsed.
	portsImport string
}

func NewParser(targetPackage string, portsImport string) *Parser {
	return &Parser{
		targetPackage: targetPackage,
		portsImport:   portsImport,
	}
}

// ParseFile parses a Go source file and extracts metadata for the specified struct(s).
// If structName is empty, "all", or "*", all structs with bson tags or ending in "Model" are extracted.
func (p *Parser) ParseFile(filePath string, structName string) ([]StructMeta, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file %s: %w", filePath, err)
	}

	packageName := node.Name.Name

	// First pass: collect all struct type definitions in the file to resolve embedded and nested structs
	structMap := make(map[string]*ast.StructType)
	var allStructNames []string

	for _, decl := range node.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			structMap[typeSpec.Name.Name] = structType
			allStructNames = append(allStructNames, typeSpec.Name.Name)
		}
	}

	// Filter target structs
	var targets []string
	if structName == "" || structName == "all" || structName == "*" {
		for _, name := range allStructNames {
			if strings.HasSuffix(name, "Model") && name != "ModelBase" {
				targets = append(targets, name)
			}
		}
		// If no *Model structs found, fallback to all structs
		if len(targets) == 0 {
			targets = allStructNames
		}
	} else if strings.Contains(structName, ",") {
		parts := strings.Split(structName, ",")
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				targets = append(targets, trimmed)
			}
		}
	} else {
		targets = []string{structName}
	}

	var results []StructMeta

	for _, target := range targets {
		structType, exists := structMap[target]
		if !exists {
			return nil, fmt.Errorf("struct %q not found in %s", target, filePath)
		}

		importSet := make(map[string]bool)
		visited := map[string]bool{target: true}
		fields := p.extractFields(structType, structMap, packageName, importSet, "", visited)

		varName := target
		if strings.HasSuffix(target, "Model") && len(target) > 5 {
			varName = strings.TrimSuffix(target, "Model")
		}

		fileName := ToKebabCase(varName) + ".go"

		var imports []string
		for imp := range importSet {
			imports = append(imports, imp)
		}

		results = append(
			results, StructMeta{
				PackageName: packageName,
				StructName:  target,
				VarName:     varName,
				FileName:    fileName,
				Imports:     imports,
				Fields:      fields,
			},
		)
	}

	return results, nil
}

func (p *Parser) extractFields(
	structType *ast.StructType,
	structMap map[string]*ast.StructType,
	sourcePackage string,
	importSet map[string]bool,
	parentKey string,
	visited map[string]bool,
) []FieldMeta {
	var fields []FieldMeta

	if structType.Fields == nil {
		return fields
	}

	for _, field := range structType.Fields.List {
		// Handle embedded structs
		if len(field.Names) == 0 {
			// An embedded field is either a bare identifier (`ModelBase`) or
			// a package-qualified selector (`hexports.ModelBase`) once a
			// project embeds the framework's type directly rather than via a
			// local alias. embeddedName is the identifier to resolve either
			// way; qualified is only used to gate the structMap lookup below,
			// since a qualified name can never resolve to a struct literal
			// in this same file.
			var embeddedName string
			var qualified bool
			switch t := field.Type.(type) {
			case *ast.Ident:
				embeddedName = t.Name
			case *ast.SelectorExpr:
				embeddedName = t.Sel.Name
				qualified = true
			}

			if embeddedName != "" {
				if !qualified {
					if embeddedStruct, exists := structMap[embeddedName]; exists && !visited[embeddedName] {
						visitedChild := make(map[string]bool)
						maps.Copy(visitedChild, visited)
						visitedChild[embeddedName] = true
						embeddedFields := p.extractFields(
							embeddedStruct,
							structMap,
							sourcePackage,
							importSet,
							parentKey,
							visitedChild,
						)
						fields = append(fields, embeddedFields...)
						continue
					}
				}

				// ModelBase/WithUserID are commonly embedded either as a
				// `type X = hexports.X` local alias or directly as
				// `hexports.X`, so they never appear in structMap (which
				// only sees literal *ast.StructType decls in this file).
				// Their shape is framework-fixed, so it is synthesized here
				// instead of resolved via cross-package AST.
				if knownFields, ok := knownEmbedFields(embeddedName, importSet); ok {
					fields = append(fields, knownFields...)
					continue
				}
			}
		}

		// Handle normal fields
		for _, nameIdent := range field.Names {
			goName := nameIdent.Name
			bsonKey, ok := parseBsonKey(field.Tag, goName)
			if !ok {
				continue
			}

			fullBsonKey := bsonKey
			if parentKey != "" {
				fullBsonKey = parentKey + "." + bsonKey
			}

			goType := p.typeToString(field.Type, sourcePackage, importSet)
			fieldType := fmt.Sprintf("Field[%s]", goType)

			// Check for nested struct fields
			var subFields []FieldMeta
			nestedTypeName := p.getNestedStructTypeName(field.Type)
			if nestedTypeName != "" && nestedTypeName != "ModelBase" && !visited[nestedTypeName] {
				if nestedStruct, exists := structMap[nestedTypeName]; exists {
					visitedChild := make(map[string]bool)
					maps.Copy(visitedChild, visited)
					visitedChild[nestedTypeName] = true
					subFields = p.extractFields(
						nestedStruct,
						structMap,
						sourcePackage,
						importSet,
						fullBsonKey,
						visitedChild,
					)
				}
			}

			fields = append(
				fields, FieldMeta{
					GoName:    goName,
					GoType:    goType,
					BsonKey:   fullBsonKey,
					FieldType: fieldType,
					SubFields: subFields,
				},
			)
		}
	}

	return fields
}

func parseBsonKey(tag *ast.BasicLit, fallbackName string) (string, bool) {
	if tag != nil {
		tagValue := reflect.StructTag(strings.Trim(tag.Value, "`")).Get("bson")
		if tagValue == "-" {
			return "", false
		}
		if tagValue != "" {
			parts := strings.Split(tagValue, ",")
			if parts[0] != "" && parts[0] != ",inline" {
				return parts[0], true
			}
		}
	}
	return ToSnakeCase(fallbackName), true
}

func (p *Parser) getNestedStructTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return p.getNestedStructTypeName(t.X)
	default:
		return ""
	}
}

func (p *Parser) typeToString(expr ast.Expr, sourcePackage string, importSet map[string]bool) string {
	switch t := expr.(type) {
	case *ast.Ident:
		name := t.Name
		if isBuiltinType(name) {
			return name
		}
		// If custom type and target package is different from source package
		if p.targetPackage != "" && p.targetPackage != sourcePackage && p.portsImport != "" {
			importSet[p.portsImport] = true
			return "ports." + name
		}
		return name

	case *ast.SelectorExpr:
		pkgIdent, ok := t.X.(*ast.Ident)
		if ok {
			pkg := pkgIdent.Name
			sel := t.Sel.Name
			if pkg == "primitive" || pkg == "bson" {
				importSet["go.mongodb.org/mongo-driver/v2/bson"] = true
				return "bson." + sel
			}
			if pkg == "time" {
				importSet["time"] = true
				return "time." + sel
			}
			return pkg + "." + sel
		}
		return fmt.Sprintf("%v.%s", t.X, t.Sel.Name)

	case *ast.StarExpr:
		return "*" + p.typeToString(t.X, sourcePackage, importSet)

	case *ast.ArrayType:
		lenStr := ""
		if t.Len != nil {
			if basicLit, ok := t.Len.(*ast.BasicLit); ok {
				lenStr = basicLit.Value
			}
		}
		return fmt.Sprintf("[%s]%s", lenStr, p.typeToString(t.Elt, sourcePackage, importSet))

	case *ast.MapType:
		keyType := p.typeToString(t.Key, sourcePackage, importSet)
		valType := p.typeToString(t.Value, sourcePackage, importSet)
		return fmt.Sprintf("map[%s]%s", keyType, valType)

	case *ast.InterfaceType:
		return "any"

	default:
		return "any"
	}
}

func isBuiltinType(name string) bool {
	switch name {
	case "string", "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
		"float32", "float64", "complex64", "complex128",
		"bool", "byte", "rune", "error", "any":
		return true
	default:
		return false
	}
}

var (
	matchFirstCap = regexp.MustCompile("([a-z0-9])([A-Z])")
	matchAllCap   = regexp.MustCompile("([A-Z]+)([A-Z][a-z0-9])")
)

func ToSnakeCase(str string) string {
	snake := matchAllCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchFirstCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

func ToKebabCase(str string) string {
	var result []rune
	runes := []rune(str)

	for i, r := range runes {
		if i > 0 && unicode.IsUpper(r) {
			prev := runes[i-1]
			var next rune
			if i+1 < len(runes) {
				next = runes[i+1]
			}
			if unicode.IsLower(prev) || (unicode.IsUpper(prev) && next != 0 && unicode.IsLower(next)) {
				result = append(result, '-')
			}
		}
		result = append(result, unicode.ToLower(r))
	}

	return string(result)
}

// knownEmbedFields synthesizes the field set for the framework's two fixed
// embeddable types, since they aren't visible as an *ast.StructType once a
// project embeds them via a `type X = hexports.X` alias. Returns ok=false
// for anything else.
func knownEmbedFields(embedName string, importSet map[string]bool) ([]FieldMeta, bool) {
	switch embedName {
	case "ModelBase":
		importSet["go.mongodb.org/mongo-driver/v2/bson"] = true
		importSet["time"] = true
		return []FieldMeta{
			{GoName: "ID", GoType: "bson.ObjectID", BsonKey: "_id", FieldType: "Field[bson.ObjectID]"},
			{GoName: "CreatedAt", GoType: "time.Time", BsonKey: "created_at", FieldType: "Field[time.Time]"},
			{GoName: "UpdatedAt", GoType: "time.Time", BsonKey: "updated_at", FieldType: "Field[time.Time]"},
		}, true

	case "WithUserID":
		importSet["go.mongodb.org/mongo-driver/v2/bson"] = true
		return []FieldMeta{
			{GoName: "UserID", GoType: "bson.ObjectID", BsonKey: "user_id", FieldType: "Field[bson.ObjectID]"},
		}, true

	default:
		return nil, false
	}
}
