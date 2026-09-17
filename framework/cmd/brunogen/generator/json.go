package generator

import (
	"bytes"
	"encoding/json"
)

// orderedMap marshals a slice of KV pairs as a JSON object, preserving
// field order — plain map[string]any shuffles keys, which would make every
// regeneration produce a spurious diff.
type orderedMap []KV

func orderedObject(kvs []KV) orderedMap { return orderedMap(kvs) }

func (m orderedMap) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, kv := range m {
		if i > 0 {
			buf.WriteByte(',')
		}
		key, err := json.Marshal(kv.Key)
		if err != nil {
			return nil, err
		}
		buf.Write(key)
		buf.WriteByte(':')
		val, err := json.Marshal(kv.Value)
		if err != nil {
			return nil, err
		}
		buf.Write(val)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// marshalIndent renders v (typically an orderedMap from ExampleObject) as
// pretty JSON, re-indenting through a decode/encode pass so nested
// orderedMap values also come out indented (json.MarshalIndent alone does
// not recurse MarshalJSON output).
func marshalIndent(v any) (string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, raw, "", "  "); err != nil {
		return "", err
	}
	return buf.String(), nil
}
