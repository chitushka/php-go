package builtins

import "strings"

type Func func([]Value) (Value, error)
type OutputFunc func([]Value) (Value, string, error)

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
	"trim":             trim,
	"ltrim":            ltrim,
	"rtrim":            rtrim,
	"str_replace":      strReplace,
	"round":            round,
	"floor":            floor,
	"ceil":             ceil,
	"abs":              abs,
	"min":              min,
	"max":              max,
	"time":             timeFn,
	"date":             dateFn,
	"file_exists":      fileExists,
	"is_file":          isFile,
	"is_dir":           isDir,
	"basename":         basename,
	"dirname":          dirname,
	"preg_match":       pregMatch,
	"preg_replace":     pregReplace,
	"serialize":        serializeFn,
	"unserialize":      unserializeFn,
}

var outputFuncs = map[string]OutputFunc{
	"print_r":  printR,
	"var_dump": varDump,
}

func Call(name string, args []Value) (Value, bool, error) {
	fn, ok := funcs[strings.ToLower(name)]
	if !ok {
		return NullValue(), false, nil
	}
	value, err := fn(args)
	return value, true, err
}

func CallWithOutput(name string, args []Value) (Value, string, bool, error) {
	fn, ok := outputFuncs[strings.ToLower(name)]
	if !ok {
		return NullValue(), "", false, nil
	}
	value, output, err := fn(args)
	return value, output, true, err
}
