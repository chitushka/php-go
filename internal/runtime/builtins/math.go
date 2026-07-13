package builtins

import "math"

func round(args []Value) (Value, error) {
	if err := checkArgCount("round", len(args), 1, 1); err != nil {
		return NullValue(), err
	}
	n, _ := args[0].Number()
	return numericResult(math.Round(n)), nil
}

func floor(args []Value) (Value, error) {
	if err := checkArgCount("floor", len(args), 1, 1); err != nil {
		return NullValue(), err
	}
	n, _ := args[0].Number()
	return numericResult(math.Floor(n)), nil
}

func ceil(args []Value) (Value, error) {
	if err := checkArgCount("ceil", len(args), 1, 1); err != nil {
		return NullValue(), err
	}
	n, _ := args[0].Number()
	return numericResult(math.Ceil(n)), nil
}

func abs(args []Value) (Value, error) {
	if err := checkArgCount("abs", len(args), 1, 1); err != nil {
		return NullValue(), err
	}
	n, _ := args[0].Number()
	return numericResult(math.Abs(n)), nil
}

func min(args []Value) (Value, error) {
	if len(args) == 0 {
		return NullValue(), checkArgCount("min", len(args), 1, 1)
	}
	values := flattenComparableArgs(args)
	if len(values) == 0 {
		return NullValue(), nil
	}
	best := values[0]
	bestN, _ := best.Number()
	for _, value := range values[1:] {
		n, _ := value.Number()
		if n < bestN {
			best = value
			bestN = n
		}
	}
	return best, nil
}

func max(args []Value) (Value, error) {
	if len(args) == 0 {
		return NullValue(), checkArgCount("max", len(args), 1, 1)
	}
	values := flattenComparableArgs(args)
	if len(values) == 0 {
		return NullValue(), nil
	}
	best := values[0]
	bestN, _ := best.Number()
	for _, value := range values[1:] {
		n, _ := value.Number()
		if n > bestN {
			best = value
			bestN = n
		}
	}
	return best, nil
}

func flattenComparableArgs(args []Value) []Value {
	if len(args) == 1 && args[0].Kind == ArrayKind && args[0].A != nil {
		out := make([]Value, 0, len(args[0].A.Entries))
		for _, entry := range args[0].A.Entries {
			out = append(out, entry.Value)
		}
		return out
	}
	return args
}

func numericResult(v float64) Value {
	if math.Trunc(v) == v {
		return IntValue(int64(v))
	}
	return FloatValue(v)
}
