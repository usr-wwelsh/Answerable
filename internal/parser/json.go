package parser

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
)

// ParseJSON reads a JSON object (or an array of objects, taking the first)
// into a Record. Nested objects are flattened with dot-joined keys; scalar
// values are stringified.
func ParseJSON(r io.Reader) (Record, error) {
	var v interface{}
	if err := json.NewDecoder(r).Decode(&v); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}

	if arr, ok := v.([]interface{}); ok {
		if len(arr) == 0 {
			return nil, fmt.Errorf("no objects found")
		}
		v = arr[0]
	}

	obj, ok := v.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("expected a JSON object (or array of objects)")
	}

	rec := Record{}
	flattenJSON("", obj, rec)
	return rec, nil
}

func flattenJSON(prefix string, obj map[string]interface{}, rec Record) {
	for k, v := range obj {
		key := normalizeKey(k)
		if prefix != "" {
			key = prefix + "." + key
		}
		switch val := v.(type) {
		case map[string]interface{}:
			flattenJSON(key, val, rec)
		case nil:
			continue
		case string:
			rec[key] = val
		case float64:
			rec[key] = strconv.FormatFloat(val, 'f', -1, 64)
		case bool:
			rec[key] = strconv.FormatBool(val)
		default:
			rec[key] = fmt.Sprint(val)
		}
	}
}
