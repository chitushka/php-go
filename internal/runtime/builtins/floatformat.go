package builtins

import (
	"math"
	"strconv"
	"strings"
)

// PHPFloatPrecision mirrors PHP's default `precision` ini setting.
const PHPFloatPrecision = 14

// FormatPHPFloat renders a float the way PHP's zend_gcvt (php_gcvt) does for
// echo/string casts/var_dump/serialize: `precision` (default 14) significant
// digits, switching to "1.0E+20" style scientific notation when the decimal
// exponent falls outside [-4, precision], and NAN/INF/-INF for non-finite
// values instead of Go's NaN/+Inf/-Inf spelling.
func FormatPHPFloat(f float64) string {
	return FormatPHPFloatPrecision(f, PHPFloatPrecision)
}

func FormatPHPFloatPrecision(f float64, precision int) string {
	if math.IsNaN(f) {
		return "NAN"
	}
	if math.IsInf(f, 1) {
		return "INF"
	}
	if math.IsInf(f, -1) {
		return "-INF"
	}

	sign := ""
	if math.Signbit(f) {
		sign = "-"
		f = -f
	}

	if f == 0 {
		return sign + "0"
	}

	// 'e' with precision-1 fractional digits yields exactly `precision`
	// significant digits, correctly rounded.
	s := strconv.FormatFloat(f, 'e', precision-1, 64)
	eIdx := strings.IndexByte(s, 'e')
	mantissa := s[:eIdx]
	exp, err := strconv.Atoi(s[eIdx+1:])
	if err != nil {
		return sign + s
	}

	digits := strings.Replace(mantissa, ".", "", 1)
	digits = strings.TrimRight(digits, "0")
	if digits == "" {
		digits = "0"
	}

	// Standard %g-style switch: scientific notation once the exponent of
	// the leading digit (exp) drops below -4 or reaches precision.
	if exp < -4 || exp >= precision {
		m := digits[:1]
		if len(digits) > 1 {
			m += "." + digits[1:]
		} else {
			m += ".0"
		}
		e := exp
		expSign := "+"
		if e < 0 {
			expSign = "-"
			e = -e
		}
		return sign + m + "E" + expSign + strconv.Itoa(e)
	}

	decpt := exp + 1

	if decpt <= 0 {
		return sign + "0." + strings.Repeat("0", -decpt) + digits
	}

	if len(digits) <= decpt {
		return sign + digits + strings.Repeat("0", decpt-len(digits))
	}

	return sign + digits[:decpt] + "." + digits[decpt:]
}
