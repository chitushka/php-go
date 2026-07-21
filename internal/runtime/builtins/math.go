package builtins

import (
	"math"
	"math/big"
	"strconv"
	"strings"
)

func round(args []Value) (Value, error) {
	if err := checkArgCount("round", len(args), 1, 2); err != nil {
		return NullValue(), err
	}
	n, _ := args[0].Number()
	precision := 0
	if len(args) == 2 {
		p, _ := args[1].Number()
		precision = int(p)
	}
	return FloatValue(phpRound(n, precision)), nil
}

// phpRound replicates PHP's round($val, $precision): half-away-from-zero
// rounding at the given number of decimal places (negative rounds into the
// integer part), always returning a float regardless of the result's
// "wholeness" — unlike PHP's round(), it must never collapse to int.
//
// It works from the shortest round-trip decimal string of the value (via
// strconv) rather than value*10^precision directly, because the latter
// reintroduces the classic binary floating point error (e.g. round(1.005, 2)
// naively yields 1.0 instead of PHP's 1.01, since 1.005 is actually stored as
// 1.00499999999999989...). The shortest round-trip string recovers the
// decimal PHP's own pre-rounding correction is designed to preserve, and
// math/big then performs exact decimal rounding on it.
func phpRound(value float64, precision int) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) || value == 0 {
		return value
	}
	if precision > 323 {
		precision = 323
	}
	if precision < -323 {
		precision = -323
	}

	neg := value < 0
	s := strconv.FormatFloat(math.Abs(value), 'f', -1, 64)
	intPart, fracPart, _ := strings.Cut(s, ".")

	digits := new(big.Int)
	digits.SetString(intPart+fracPart, 10)
	fracLen := len(fracPart)

	drop := fracLen - precision
	switch {
	case drop > 0:
		divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(drop)), nil)
		quo, rem := new(big.Int), new(big.Int)
		quo.QuoRem(digits, divisor, rem)
		rem.Abs(rem)
		rem.Lsh(rem, 1)
		if rem.Cmp(divisor) >= 0 {
			quo.Add(quo, big.NewInt(1))
		}
		digits = quo
	case drop < 0:
		multiplier := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-drop)), nil)
		digits = new(big.Int).Mul(digits, multiplier)
	}

	digitStr := digits.String()
	var resultStr string
	if precision > 0 {
		for len(digitStr) <= precision {
			digitStr = "0" + digitStr
		}
		splitAt := len(digitStr) - precision
		resultStr = digitStr[:splitAt] + "." + digitStr[splitAt:]
	} else {
		resultStr = digitStr + strings.Repeat("0", -precision)
	}

	out, err := strconv.ParseFloat(resultStr, 64)
	if err != nil {
		return value
	}
	if neg {
		out = -out
	}
	return out
}

func floor(args []Value) (Value, error) {
	if err := checkArgCount("floor", len(args), 1, 1); err != nil {
		return NullValue(), err
	}
	n, _ := args[0].Number()
	return FloatValue(math.Floor(n)), nil
}

func ceil(args []Value) (Value, error) {
	if err := checkArgCount("ceil", len(args), 1, 1); err != nil {
		return NullValue(), err
	}
	n, _ := args[0].Number()
	return FloatValue(math.Ceil(n)), nil
}

func abs(args []Value) (Value, error) {
	if err := checkArgCount("abs", len(args), 1, 1); err != nil {
		return NullValue(), err
	}
	if args[0].Kind == IntKind {
		n := args[0].I
		if n == math.MinInt64 {
			return FloatValue(math.Abs(float64(n))), nil
		}
		if n < 0 {
			n = -n
		}
		return IntValue(n), nil
	}
	n, _ := args[0].Number()
	return FloatValue(math.Abs(n)), nil
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
