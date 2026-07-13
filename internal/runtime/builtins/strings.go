package builtins

import "strings"

func strlen(args []Value) (Value, error) {
	if err := checkArgCount("strlen", len(args), 1, 1); err != nil {
		return NullValue(), err
	}
	return IntValue(int64(len(args[0].String()))), nil
}

func substr(args []Value) (Value, error) {
	if err := checkArgCount("substr", len(args), 2, 3); err != nil {
		return NullValue(), err
	}
	s := args[0].String()
	start := IntValueOf(args[1])
	if start < 0 {
		start = len(s) + start
	}
	if start < 0 {
		start = 0
	}
	if start > len(s) {
		return StringValue(""), nil
	}
	end := len(s)
	if len(args) == 3 {
		length := IntValueOf(args[2])
		if length < 0 {
			end = len(s) + length
		} else {
			end = start + length
		}
	}
	if end < start {
		end = start
	}
	if end > len(s) {
		end = len(s)
	}
	return StringValue(s[start:end]), nil
}

func strpos(args []Value) (Value, error) {
	if err := checkArgCount("strpos", len(args), 2, 3); err != nil {
		return NullValue(), err
	}
	haystack := args[0].String()
	needle := args[1].String()
	offset := 0
	if len(args) == 3 {
		offset = IntValueOf(args[2])
		if offset < 0 {
			offset = 0
		}
	}
	if offset > len(haystack) {
		return BoolValue(false), nil
	}
	pos := strings.Index(haystack[offset:], needle)
	if pos < 0 {
		return BoolValue(false), nil
	}
	return IntValue(int64(offset + pos)), nil
}

func strtolower(args []Value) (Value, error) {
	if err := checkArgCount("strtolower", len(args), 1, 1); err != nil {
		return NullValue(), err
	}
	return StringValue(strings.ToLower(args[0].String())), nil
}

func strtoupper(args []Value) (Value, error) {
	if err := checkArgCount("strtoupper", len(args), 1, 1); err != nil {
		return NullValue(), err
	}
	return StringValue(strings.ToUpper(args[0].String())), nil
}

func trim(args []Value) (Value, error) {
	if err := checkArgCount("trim", len(args), 1, 2); err != nil {
		return NullValue(), err
	}
	cutset := " \n\r\t\v\x00"
	if len(args) == 2 {
		cutset = args[1].String()
	}
	return StringValue(strings.Trim(args[0].String(), cutset)), nil
}

func ltrim(args []Value) (Value, error) {
	if err := checkArgCount("ltrim", len(args), 1, 2); err != nil {
		return NullValue(), err
	}
	cutset := " \n\r\t\v\x00"
	if len(args) == 2 {
		cutset = args[1].String()
	}
	return StringValue(strings.TrimLeft(args[0].String(), cutset)), nil
}

func rtrim(args []Value) (Value, error) {
	if err := checkArgCount("rtrim", len(args), 1, 2); err != nil {
		return NullValue(), err
	}
	cutset := " \n\r\t\v\x00"
	if len(args) == 2 {
		cutset = args[1].String()
	}
	return StringValue(strings.TrimRight(args[0].String(), cutset)), nil
}

func strReplace(args []Value) (Value, error) {
	if err := checkArgCount("str_replace", len(args), 3, 3); err != nil {
		return NullValue(), err
	}
	return StringValue(strings.ReplaceAll(args[2].String(), args[0].String(), args[1].String())), nil
}
