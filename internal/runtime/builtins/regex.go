package builtins

import (
	"regexp"
	"strings"
)

func pregMatch(args []Value) (Value, error) {
	if err := checkArgCount("preg_match", len(args), 2, 2); err != nil {
		return NullValue(), err
	}
	re, err := regexp.Compile(regexPattern(args[0].String()))
	if err != nil {
		return NullValue(), err
	}
	if re.MatchString(args[1].String()) {
		return IntValue(1), nil
	}
	return IntValue(0), nil
}

func pregReplace(args []Value) (Value, error) {
	if err := checkArgCount("preg_replace", len(args), 3, 3); err != nil {
		return NullValue(), err
	}
	re, err := regexp.Compile(regexPattern(args[0].String()))
	if err != nil {
		return NullValue(), err
	}
	return StringValue(re.ReplaceAllString(args[2].String(), args[1].String())), nil
}

func regexPattern(pattern string) string {
	if len(pattern) < 2 {
		return pattern
	}
	delimiter := pattern[0]
	if strings.ContainsRune(`/#~!%`, rune(delimiter)) {
		for idx := len(pattern) - 1; idx > 0; idx-- {
			if pattern[idx] == delimiter {
				return pattern[1:idx]
			}
		}
	}
	return pattern
}
