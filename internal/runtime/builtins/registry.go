package builtins

import "strings"

type Func func([]Value) (Value, error)
type OutputFunc func([]Value) (Value, string, error)

var funcs = map[string]Func{
	"isset":             isset,
	"empty":             empty,
	"count":             count,
	"strlen":            strlen,
	"substr":            substr,
	"strpos":            strpos,
	"strtolower":        strtolower,
	"strtoupper":        strtoupper,
	"array_key_exists":  arrayKeyExists,
	"in_array":          inArray,
	"implode":           implode,
	"explode":           explode,
	"array_sum":         arraySum,
	"array_product":     arrayProduct,
	"array_reverse":     arrayReverse,
	"array_slice":       arraySlice,
	"array_search":      arraySearch,
	"array_merge":       arrayMerge,
	"array_keys":        arrayKeys,
	"is_array":          isArray,
	"is_string":         isString,
	"is_null":           isNull,
	"is_object":         isObject,
	"trim":              trim,
	"ltrim":             ltrim,
	"rtrim":             rtrim,
	"str_replace":       strReplace,
	"str_repeat":        strRepeat,
	"number_format":     numberFormat,
	"str_pad":           strPad,
	"wordwrap":          wordwrap,
	"sprintf":           sprintfFn,
	"round":             round,
	"floor":             floor,
	"ceil":              ceil,
	"abs":               abs,
	"min":               min,
	"max":               max,
	"time":              timeFn,
	"date":              dateFn,
	"file_exists":       fileExists,
	"is_file":           isFile,
	"is_dir":            isDir,
	"basename":          basename,
	"dirname":           dirname,
	"file_get_contents": fileGetContents,
	"file_put_contents": filePutContents,
	"unlink":            unlinkFn,
	"preg_match":        pregMatch,
	"preg_replace":      pregReplace,
	"json_encode":       jsonEncode,
	"json_decode":       jsonDecode,
	"md5":               md5Fn,
	"sha1":              sha1Fn,
	"hash":              hashFn,
	"serialize":         serializeFn,
	"unserialize":       unserializeFn,
}

var outputFuncs = map[string]OutputFunc{
	"print_r":  printR,
	"var_dump": varDump,
	"printf":   printfFn,
}

func Call(name string, args []Value) (Value, bool, error) {
	fn, ok := funcs[strings.ToLower(name)]
	if !ok {
		return NullValue(), false, nil
	}
	value, err := fn(args)
	return value, true, err
}

// Exists reports whether name is a registered builtin function (used by
// function_exists()). It does not know about the additional builtins that
// are special-cased directly in the interpreter's call dispatch (isset,
// array_map, sort, ...) — callers should check those separately.
func Exists(name string) bool {
	key := strings.ToLower(name)
	if _, ok := funcs[key]; ok {
		return true
	}
	_, ok := outputFuncs[key]
	return ok
}

func CallWithOutput(name string, args []Value) (Value, string, bool, error) {
	fn, ok := outputFuncs[strings.ToLower(name)]
	if !ok {
		return NullValue(), "", false, nil
	}
	value, output, err := fn(args)
	return value, output, true, err
}
