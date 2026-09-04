package generator

import (
	"bytes"
	"fmt"
	"go/format"
	"sort"
	"text/template"

	"github.com/king-glitch/hexag/framework/cmd/mongogen/internal/parser"
)

// FieldImportPath is the framework package generated model files import for
// Field[T] in non-standalone mode. Its package name is "mongo".
const FieldImportPath = "github.com/king-glitch/hexag/framework/mongo"

type Generator struct {
	fieldTmpl      *template.Template
	modelTmpl      *template.Template
	standaloneTmpl *template.Template
}

func NewGenerator() (*Generator, error) {
	fieldTmpl, err := template.New("field").Parse(FieldTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse field template: %w", err)
	}

	modelTmpl, err := template.New("model").Parse(ModelTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse model template: %w", err)
	}

	standaloneTmpl, err := template.New("standalone").Parse(StandaloneModelTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse standalone template: %w", err)
	}

	return &Generator{
		fieldTmpl:      fieldTmpl,
		modelTmpl:      modelTmpl,
		standaloneTmpl: standaloneTmpl,
	}, nil
}

type ModelViewData struct {
	PackageName string
	StructName  string
	VarName     string
	HasAlias    bool
	Imports     []string
	Fields      []parser.FieldMeta
	// FieldPkg is the qualifier prefix for Field[T]/NewField, e.g. "mongo."
	// in shared mode or "" in standalone mode (Field[T] is embedded).
	FieldPkg string
}

func (g *Generator) GenerateFieldsFile(packageName string) ([]byte, error) {
	data := struct {
		PackageName string
	}{
		PackageName: packageName,
	}

	var buf bytes.Buffer
	if err := g.fieldTmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute field template: %w", err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to format field file: %w\nSource:\n%s", err, buf.String())
	}

	return formatted, nil
}

// GenerateModel renders one model's Field-struct file. In standalone mode,
// Field[T] is embedded in the output and FieldPkg stays empty. Otherwise
// FieldImportPath is added to the file's imports and every Field[T]/
// NewField reference is qualified with "mongo.".
func (g *Generator) GenerateModel(
	meta parser.StructMeta,
	packageName string,
	standalone bool,
) ([]byte, error) {
	if packageName == "" {
		packageName = meta.PackageName
	}

	importSet := make(map[string]bool)
	importSet["go.mongodb.org/mongo-driver/v2/bson"] = true

	fieldPkg := ""
	if !standalone {
		importSet[FieldImportPath] = true
		fieldPkg = "mongo."
	}

	for _, imp := range meta.Imports {
		importSet[imp] = true
	}

	var imports []string
	for imp := range importSet {
		imports = append(imports, imp)
	}
	sort.Strings(imports)

	hasAlias := meta.VarName != meta.StructName

	data := ModelViewData{
		PackageName: packageName,
		StructName:  meta.StructName,
		VarName:     meta.VarName,
		HasAlias:    hasAlias,
		Imports:     imports,
		Fields:      withFieldPkg(meta.Fields, fieldPkg),
		FieldPkg:    fieldPkg,
	}

	var buf bytes.Buffer
	tmpl := g.modelTmpl
	if standalone {
		tmpl = g.standaloneTmpl
	}

	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute model template for %s: %w", meta.StructName, err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf(
			"failed to format model file for %s: %w\nSource:\n%s",
			meta.StructName,
			err,
			buf.String(),
		)
	}

	return formatted, nil
}

// withFieldPkg stamps FieldPkg onto every field in the tree, including
// nested SubFields, so the template can read it off each node directly
// instead of relying on `$` across a {{template}} recursion boundary.
func withFieldPkg(fields []parser.FieldMeta, pkg string) []parser.FieldMeta {
	out := make([]parser.FieldMeta, len(fields))
	for i, f := range fields {
		f.FieldPkg = pkg
		f.SubFields = withFieldPkg(f.SubFields, pkg)
		out[i] = f
	}

	return out
}
