package builtins

import (
	"encoding/json"
	"math"
)

func jsonEncode(args []Value) (Value, error) {
	if err := checkArgCount("json_encode", len(args), 1, 1); err != nil {
		return NullValue(), err
	}
	data, err := json.Marshal(jsonNative(args[0]))
	if err != nil {
		return BoolValue(false), nil
	}
	return StringValue(string(data)), nil
}

func jsonDecode(args []Value) (Value, error) {
	if err := checkArgCount("json_decode", len(args), 1, 2); err != nil {
		return NullValue(), err
	}
	var decoded any
	if err := json.Unmarshal([]byte(args[0].String()), &decoded); err != nil {
		return NullValue(), nil
	}
	return valueFromJSON(decoded), nil
}

func jsonNative(value Value) any {
	switch value.Kind {
	case NullKind:
		return nil
	case BoolKind:
		return value.B
	case IntKind:
		return value.I
	case FloatKind:
		return value.F
	case StringKind:
		return value.S
	case ArrayKind:
		if value.A == nil {
			return []any{}
		}
		if arrayIsList(value.A) {
			out := make([]any, 0, len(value.A.Entries))
			for _, entry := range value.A.Entries {
				out = append(out, jsonNative(entry.Value))
			}
			return out
		}
		out := map[string]any{}
		for _, entry := range value.A.Entries {
			key := entry.Key.Str
			if entry.Key.IsInt {
				key = formatJSONIntKey(entry.Key.Int)
			}
			out[key] = jsonNative(entry.Value)
		}
		return out
	default:
		return nil
	}
}

func arrayIsList(arr *Array) bool {
	for idx, entry := range arr.Entries {
		if !entry.Key.IsInt || entry.Key.Int != int64(idx) {
			return false
		}
	}
	return true
}

func formatJSONIntKey(key int64) string {
	return IntValue(key).String()
}

func valueFromJSON(value any) Value {
	switch v := value.(type) {
	case nil:
		return NullValue()
	case bool:
		return BoolValue(v)
	case string:
		return StringValue(v)
	case float64:
		if math.Trunc(v) == v {
			return IntValue(int64(v))
		}
		return FloatValue(v)
	case []any:
		arr := NewArray()
		for _, item := range v {
			arr.Append(valueFromJSON(item))
		}
		return ArrayValue(arr)
	case map[string]any:
		arr := NewArray()
		for key, item := range v {
			arr.Set(StringKey(key), valueFromJSON(item))
		}
		return ArrayValue(arr)
	default:
		return NullValue()
	}
}
