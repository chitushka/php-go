package builtins

import (
	"math"
	"testing"
)

func TestFormatPHPFloat(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{0.1 + 0.2, "0.3"},
		{0.3, "0.3"},
		{3.0, "3"},
		{-3.0, "-3"},
		{0.0, "0"},
		{123.45, "123.45"},
		{12300.0, "12300"},
		{0.0001, "0.0001"},
		{0.00001, "1.0E-5"},
		{0.000123, "0.000123"},
		{1e13, "10000000000000"},
		{1e14, "1.0E+14"},
		{1e20, "1.0E+20"},
		{math.Pi, "3.1415926535898"},
		{math.NaN(), "NAN"},
		{math.Inf(1), "INF"},
		{math.Inf(-1), "-INF"},
	}
	for _, c := range cases {
		if got := FormatPHPFloat(c.in); got != c.want {
			t.Errorf("FormatPHPFloat(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}
