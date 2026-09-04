package generator

import (
	"strings"

	"github.com/king-glitch/hexag/framework/cmd/brunogen/internal/parser"
)

// ExampleFields builds a JSON-marshalable example value from resolved
// fields (order preserved via a slice of key/value pairs rendered by the
// caller, not a map, so output is deterministic).
func ExampleObject(fields []parser.Field) []KV {
	kvs := make([]KV, 0, len(fields))
	for _, f := range fields {
		kvs = append(kvs, KV{Key: f.Key, Value: exampleValue(f)})
	}
	return kvs
}

// DataValue builds the example value for the response envelope's "data"
// key. Almost always an object built from fields, except the rare
// bare-slice response (`type XResponse []Y`), which parser.resolveNamed
// represents as a single Field with an empty Key — there, the whole body
// IS that one field's value (a JSON array), not an object wrapping it.
func DataValue(fields []parser.Field) any {
	if len(fields) == 1 && fields[0].Key == "" {
		return exampleValue(fields[0])
	}
	return orderedObject(ExampleObject(fields))
}

// KV is an ordered key/value pair — plain map[string]any would shuffle key
// order on every marshal, which makes regenerated diffs noisy for no reason.
type KV struct {
	Key   string
	Value any
}

func exampleValue(f parser.Field) any {
	if len(f.OneOf) > 0 {
		return f.OneOf[0]
	}
	if f.Email {
		return "user@example.com"
	}

	switch f.Kind {
	case "string":
		return exampleString(f.Key)
	case "int":
		if strings.HasSuffix(f.Key, "_amount") || strings.HasSuffix(f.Key, "_price") || strings.Contains(f.Key, "satang") {
			return 150000
		}
		return 1
	case "float":
		return 1.5
	case "bool":
		return true
	case "objectid":
		return "65f1a2b3c4d5e6f7a8b9c0d1"
	case "time":
		return "2026-09-03T12:00:00Z"
	case "object":
		return orderedObject(ExampleObject(f.SubFields))
	case "collection":
		return orderedObject(ExampleObject(f.SubFields))
	case "array":
		if len(f.SubFields) == 0 {
			return []any{}
		}
		item := f.SubFields[0]
		if item.Kind == "object" {
			return []any{orderedObject(ExampleObject(item.SubFields))}
		}
		return []any{exampleValue(item)}
	case "map":
		if len(f.SubFields) == 0 {
			return orderedObject(nil)
		}
		return orderedObject([]KV{{Key: "example_key", Value: exampleValue(f.SubFields[0])}})
	default:
		return exampleString(f.Key)
	}
}

func exampleString(key string) string {
	if key == "" {
		return "example value"
	}
	return "example " + strings.ReplaceAll(key, "_", " ")
}
