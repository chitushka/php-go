package builtins

func isset(args []Value) (Value, error) {
	if len(args) == 0 {
		return NullValue(), checkArgCount("isset", len(args), 1, 1)
	}
	for _, arg := range args {
		if arg.Kind == NullKind {
			return BoolValue(false), nil
		}
	}
	return BoolValue(true), nil
}

func empty(args []Value) (Value, error) {
	if err := checkArgCount("empty", len(args), 1, 1); err != nil {
		return NullValue(), err
	}
	return BoolValue(!args[0].Truthy()), nil
}
