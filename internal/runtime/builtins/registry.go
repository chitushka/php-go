package builtins

import "strings"

type Func func([]Value) (Value, error)

var funcs = map[string]Func{
	"isset":            isset,
	"empty":            empty,
	"count":            count,
	"strlen":           strlen,
	"substr":           substr,
	"strpos":           strpos,
	"strtolower":       strtolower,
	"strtoupper":       strtoupper,
	"array_key_exists": arrayKeyExists,
	"in_array":         inArray,
	"implode":          implode,
	"explode":          explode,
	"is_array":         isArray,
	"is_string":        isString,
	"is_null":          isNull,
}

func Call(name string, args []Value) (Value, bool, error) {
	fn, ok := funcs[strings.ToLower(name)]
	if !ok {
		return NullValue(), false, nil
	}
	value, err := fn(args)
	return value, true, err
}
