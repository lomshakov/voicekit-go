package voicekit

import (
	"bytes"
	"encoding/json"
)

// Object is a decoded JSON object returned by the API. Response shapes evolve
// server-side, so the SDK hands back dynamic objects with typed accessors
// instead of freezing every field into a struct.
type Object map[string]any

// Get returns the raw value at key (nil when absent).
func (o Object) Get(key string) any {
	if o == nil {
		return nil
	}
	return o[key]
}

// Str returns the string value at key ("" when absent or not a string).
func (o Object) Str(key string) string {
	value, _ := o[key].(string)
	return value
}

// Num returns the numeric value at key (0 when absent or not a number).
func (o Object) Num(key string) float64 {
	switch value := o[key].(type) {
	case float64:
		return value
	case float32:
		return float64(value)
	case int:
		return float64(value)
	case int64:
		return float64(value)
	case json.Number:
		parsed, _ := value.Float64()
		return parsed
	default:
		return 0
	}
}

// Int returns the integer value at key (0 when absent).
func (o Object) Int(key string) int { return int(o.Num(key)) }

// Bool returns the boolean value at key (false when absent or not a bool).
func (o Object) Bool(key string) bool {
	value, _ := o[key].(bool)
	return value
}

// Obj returns the nested object at key (nil when absent).
func (o Object) Obj(key string) Object {
	if nested, ok := o[key].(map[string]any); ok {
		return Object(nested)
	}
	return nil
}

// List returns the nested array at key as objects.
func (o Object) List(key string) []Object {
	raw, ok := o[key].([]any)
	if !ok {
		return nil
	}
	out := make([]Object, 0, len(raw))
	for _, item := range raw {
		if nested, ok := item.(map[string]any); ok {
			out = append(out, Object(nested))
		}
	}
	return out
}

// Strings returns the nested array at key as strings.
func (o Object) Strings(key string) []string {
	raw, ok := o[key].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if text, ok := item.(string); ok {
			out = append(out, text)
		}
	}
	return out
}

func decodeInto(data []byte, out any) error {
	if out == nil || len(bytes.TrimSpace(data)) == 0 {
		return nil
	}
	return json.Unmarshal(data, out)
}

// decodeList accepts either a bare JSON array or an object wrapping one under
// a well-known key, so it stays compatible with paginated responses.
func decodeList(data []byte) ([]Object, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, nil
	}
	if trimmed[0] == '[' {
		var direct []Object
		if err := json.Unmarshal(trimmed, &direct); err != nil {
			return nil, err
		}
		return direct, nil
	}
	var wrapper Object
	if err := json.Unmarshal(trimmed, &wrapper); err != nil {
		return nil, err
	}
	for _, key := range []string{"items", "values", "data", "voices", "profiles", "recordings"} {
		if list := wrapper.List(key); list != nil {
			return list, nil
		}
	}
	return nil, nil
}
