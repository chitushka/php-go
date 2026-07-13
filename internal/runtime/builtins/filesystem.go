package builtins

import (
	"os"
	"path/filepath"
)

func fileExists(args []Value) (Value, error) {
	if err := checkArgCount("file_exists", len(args), 1, 1); err != nil {
		return NullValue(), err
	}
	_, err := os.Stat(args[0].String())
	return BoolValue(err == nil), nil
}

func isFile(args []Value) (Value, error) {
	if err := checkArgCount("is_file", len(args), 1, 1); err != nil {
		return NullValue(), err
	}
	info, err := os.Stat(args[0].String())
	return BoolValue(err == nil && !info.IsDir()), nil
}

func isDir(args []Value) (Value, error) {
	if err := checkArgCount("is_dir", len(args), 1, 1); err != nil {
		return NullValue(), err
	}
	info, err := os.Stat(args[0].String())
	return BoolValue(err == nil && info.IsDir()), nil
}

func basename(args []Value) (Value, error) {
	if err := checkArgCount("basename", len(args), 1, 1); err != nil {
		return NullValue(), err
	}
	return StringValue(filepath.Base(args[0].String())), nil
}

func dirname(args []Value) (Value, error) {
	if err := checkArgCount("dirname", len(args), 1, 1); err != nil {
		return NullValue(), err
	}
	return StringValue(filepath.Dir(args[0].String())), nil
}
