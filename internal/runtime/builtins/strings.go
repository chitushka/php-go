package builtins

import (
	"strconv"
	"strings"
)

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

func strRepeat(args []Value) (Value, error) {
	if err := checkArgCount("str_repeat", len(args), 2, 2); err != nil {
		return NullValue(), err
	}
	times := IntValueOf(args[1])
	if times <= 0 {
		return StringValue(""), nil
	}
	return StringValue(strings.Repeat(args[0].String(), times)), nil
}

// numberFormat mirrors PHP's number_format(): rounds to $decimals places
// (reusing phpRound's binary-float pre-rounding correction) then inserts
// $thousands_sep every 3 digits of the integer part.
func numberFormat(args []Value) (Value, error) {
	if err := checkArgCount("number_format", len(args), 1, 4); err != nil {
		return NullValue(), err
	}
	n, _ := args[0].Number()
	decimals := 0
	if len(args) >= 2 {
		decimals = IntValueOf(args[1])
	}
	decPoint := "."
	if len(args) >= 3 {
		decPoint = args[2].String()
	}
	thousandsSep := ","
	if len(args) >= 4 {
		thousandsSep = args[3].String()
	}
	if decimals < 0 {
		decimals = 0
	}
	rounded := phpRound(n, decimals)
	neg := rounded < 0
	if neg {
		rounded = -rounded
	}
	raw := strconv.FormatFloat(rounded, 'f', decimals, 64)
	intPart, fracPart, _ := strings.Cut(raw, ".")

	var b strings.Builder
	for idx, ch := range []byte(intPart) {
		if idx > 0 && (len(intPart)-idx)%3 == 0 {
			b.WriteString(thousandsSep)
		}
		b.WriteByte(ch)
	}
	result := b.String()
	if decimals > 0 {
		result += decPoint + fracPart
	}
	if neg {
		result = "-" + result
	}
	return StringValue(result), nil
}

// strPad mirrors PHP's str_pad(): pads $input with (repeats of) $pad_string
// up to $pad_length, on the left/right/both sides per $pad_type
// (STR_PAD_RIGHT is the default, matching PHP).
func strPad(args []Value) (Value, error) {
	if err := checkArgCount("str_pad", len(args), 2, 4); err != nil {
		return NullValue(), err
	}
	s := args[0].String()
	padLength := IntValueOf(args[1])
	padString := " "
	if len(args) >= 3 && args[2].String() != "" {
		padString = args[2].String()
	}
	padType := int64(1) // STR_PAD_RIGHT
	if len(args) == 4 {
		n, _ := args[3].Number()
		padType = int64(n)
	}
	need := padLength - len(s)
	if need <= 0 {
		return StringValue(s), nil
	}
	buildPad := func(length int) string {
		if length <= 0 {
			return ""
		}
		var b strings.Builder
		for b.Len() < length {
			b.WriteString(padString)
		}
		return b.String()[:length]
	}
	switch padType {
	case 0: // STR_PAD_LEFT
		return StringValue(buildPad(need) + s), nil
	case 2: // STR_PAD_BOTH
		left := need / 2
		right := need - left
		return StringValue(buildPad(left) + s + buildPad(right)), nil
	default: // STR_PAD_RIGHT
		return StringValue(s + buildPad(need)), nil
	}
}

// wordwrap mirrors PHP's wordwrap(): breaks $str into lines of at most
// $width chars at word boundaries (spaces), joined by $break; with $cut,
// single words longer than $width are force-split.
func wordwrap(args []Value) (Value, error) {
	if err := checkArgCount("wordwrap", len(args), 1, 4); err != nil {
		return NullValue(), err
	}
	s := args[0].String()
	width := 75
	if len(args) >= 2 {
		width = IntValueOf(args[1])
	}
	brk := "\n"
	if len(args) >= 3 {
		brk = args[2].String()
	}
	cut := len(args) == 4 && args[3].Truthy()
	if width <= 0 {
		width = 1
	}

	var lines []string
	var current strings.Builder
	for _, word := range strings.Split(s, " ") {
		for cut && len(word) > width {
			if current.Len() > 0 {
				lines = append(lines, current.String())
				current.Reset()
			}
			lines = append(lines, word[:width])
			word = word[width:]
		}
		switch {
		case current.Len() == 0:
			current.WriteString(word)
		case current.Len()+1+len(word) <= width:
			current.WriteString(" ")
			current.WriteString(word)
		default:
			lines = append(lines, current.String())
			current.Reset()
			current.WriteString(word)
		}
	}
	lines = append(lines, current.String())
	return StringValue(strings.Join(lines, brk)), nil
}
