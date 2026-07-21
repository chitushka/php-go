package builtins

import (
	"fmt"
	"strconv"
	"strings"
)

func sprintfFn(args []Value) (Value, error) {
	if len(args) < 1 {
		return NullValue(), fmt.Errorf("sprintf expects at least 1 argument, got %d", len(args))
	}
	result, err := formatSprintf(args[0].String(), args[1:])
	if err != nil {
		return NullValue(), err
	}
	return StringValue(result), nil
}

func printfFn(args []Value) (Value, string, error) {
	if len(args) < 1 {
		return NullValue(), "", fmt.Errorf("printf expects at least 1 argument, got %d", len(args))
	}
	result, err := formatSprintf(args[0].String(), args[1:])
	if err != nil {
		return NullValue(), "", err
	}
	return IntValue(int64(len(result))), result, nil
}

// formatSprintf implements PHP's sprintf()/printf() format-string dialect:
// %[flags][width][.precision]specifier, with flags `-` (left-justify), `0`
// (zero-pad), `+` (force sign) and `'X` (custom pad char X). Positional
// (`%2$s`) arguments are not supported — not needed by any caller yet.
func formatSprintf(format string, args []Value) (string, error) {
	var b strings.Builder
	argIdx := 0
	i := 0
	n := len(format)
	for i < n {
		if format[i] != '%' {
			b.WriteByte(format[i])
			i++
			continue
		}
		i++
		if i >= n {
			break
		}
		if format[i] == '%' {
			b.WriteByte('%')
			i++
			continue
		}

		leftJustify := false
		showSign := false
		padChar := byte(' ')
	flags:
		for i < n {
			switch format[i] {
			case '-':
				leftJustify = true
				i++
			case '+':
				showSign = true
				i++
			case '0':
				padChar = '0'
				i++
			case ' ':
				i++
			case '\'':
				i++
				if i < n {
					padChar = format[i]
					i++
				}
			default:
				break flags
			}
		}

		width := 0
		for i < n && format[i] >= '0' && format[i] <= '9' {
			width = width*10 + int(format[i]-'0')
			i++
		}

		precision := -1
		if i < n && format[i] == '.' {
			i++
			precision = 0
			for i < n && format[i] >= '0' && format[i] <= '9' {
				precision = precision*10 + int(format[i]-'0')
				i++
			}
		}

		if i >= n {
			return "", fmt.Errorf("sprintf(): unterminated format specifier")
		}
		verb := format[i]
		i++

		if argIdx >= len(args) {
			return "", fmt.Errorf("sprintf(): too few arguments")
		}
		arg := args[argIdx]
		argIdx++

		piece, err := formatSprintfVerb(verb, arg, precision, showSign)
		if err != nil {
			return "", err
		}

		if len(piece) < width {
			padCount := width - len(piece)
			if leftJustify {
				piece += strings.Repeat(" ", padCount)
			} else if padChar == '0' && len(piece) > 0 && (piece[0] == '-' || piece[0] == '+') {
				piece = piece[:1] + strings.Repeat("0", padCount) + piece[1:]
			} else {
				piece = strings.Repeat(string(padChar), padCount) + piece
			}
		}
		b.WriteString(piece)
	}
	return b.String(), nil
}

func formatSprintfVerb(verb byte, arg Value, precision int, showSign bool) (string, error) {
	switch verb {
	case 's':
		s := arg.String()
		if precision >= 0 && precision < len(s) {
			s = s[:precision]
		}
		return s, nil
	case 'd':
		n, _ := arg.Number()
		iv := int64(n)
		if showSign && iv >= 0 {
			return "+" + strconv.FormatInt(iv, 10), nil
		}
		return strconv.FormatInt(iv, 10), nil
	case 'u':
		n, _ := arg.Number()
		return strconv.FormatUint(uint64(int64(n)), 10), nil
	case 'f', 'F':
		n, _ := arg.Number()
		if precision < 0 {
			precision = 6
		}
		s := strconv.FormatFloat(n, 'f', precision, 64)
		if showSign && n >= 0 {
			s = "+" + s
		}
		return s, nil
	case 'e', 'E':
		n, _ := arg.Number()
		if precision < 0 {
			precision = 6
		}
		return strconv.FormatFloat(n, byte(verb), precision, 64), nil
	case 'x':
		n, _ := arg.Number()
		return strconv.FormatInt(int64(n), 16), nil
	case 'X':
		n, _ := arg.Number()
		return strings.ToUpper(strconv.FormatInt(int64(n), 16)), nil
	case 'o':
		n, _ := arg.Number()
		return strconv.FormatInt(int64(n), 8), nil
	case 'b':
		n, _ := arg.Number()
		return strconv.FormatInt(int64(n), 2), nil
	case 'c':
		n, _ := arg.Number()
		return string([]byte{byte(int64(n))}), nil
	default:
		return "", fmt.Errorf("sprintf(): unknown format specifier %%%c", verb)
	}
}
