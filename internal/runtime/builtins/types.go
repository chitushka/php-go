package builtins

func isArray(args []Value) (Value, error) {
	if err := checkArgCount("is_array", len(args), 1, 1); err != nil {
		return NullValue(), err
	}
	return BoolValue(args[0].Kind == ArrayKind), nil
}

func isString(args []Value) (Value, error) {
	if err := checkArgCount("is_string", len(args), 1, 1); err != nil {
		return NullValue(), err
	}
	return BoolValue(args[0].Kind == StringKind), nil
}

func isNull(args []Value) (Value, error) {
	if err := checkArgCount("is_null", len(args), 1, 1); err != nil {
		return NullValue(), err
	}
	return BoolValue(args[0].Kind == NullKind), nil
}
