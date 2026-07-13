package builtins

import (
	"strconv"
	"strings"
	"time"
)

func timeFn(args []Value) (Value, error) {
	if err := checkArgCount("time", len(args), 0, 0); err != nil {
		return NullValue(), err
	}
	return IntValue(time.Now().Unix()), nil
}

func dateFn(args []Value) (Value, error) {
	if err := checkArgCount("date", len(args), 1, 2); err != nil {
		return NullValue(), err
	}
	ts := time.Now()
	if len(args) == 2 {
		n, _ := args[1].Number()
		ts = time.Unix(int64(n), 0)
	}
	return StringValue(formatDate(args[0].String(), ts)), nil
}

func formatDate(format string, ts time.Time) string {
	replacements := map[rune]string{
		'Y': ts.Format("2006"),
		'y': ts.Format("06"),
		'm': ts.Format("01"),
		'n': ts.Format("1"),
		'd': ts.Format("02"),
		'j': ts.Format("2"),
		'H': ts.Format("15"),
		'i': ts.Format("04"),
		's': ts.Format("05"),
		'U': strconv.FormatInt(ts.Unix(), 10),
	}
	var b strings.Builder
	escaped := false
	for _, r := range format {
		if escaped {
			b.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		if replacement, ok := replacements[r]; ok {
			b.WriteString(replacement)
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
