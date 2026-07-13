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

func fileGetContents(args []Value) (Value, error) {
	if err := checkArgCount("file_get_contents", len(args), 1, 1); err != nil {
		return NullValue(), err
	}
	data, err := os.ReadFile(args[0].String())
	if err != nil {
		return BoolValue(false), nil
	}
	return StringValue(string(data)), nil
}

func filePutContents(args []Value) (Value, error) {
	if err := checkArgCount("file_put_contents", len(args), 2, 3); err != nil {
		return NullValue(), err
	}
	data := []byte(args[1].String())
	if err := os.WriteFile(args[0].String(), data, 0644); err != nil {
		return BoolValue(false), nil
	}
	return IntValue(int64(len(data))), nil
}

func unlinkFn(args []Value) (Value, error) {
	if err := checkArgCount("unlink", len(args), 1, 1); err != nil {
		return NullValue(), err
	}
	if err := os.Remove(args[0].String()); err != nil {
		return BoolValue(false), nil
	}
	return BoolValue(true), nil
}
