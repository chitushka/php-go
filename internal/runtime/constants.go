package runtime

import (
	"math"
	"os"
	goruntime "runtime"
)

// predefinedConstants holds PHP 5.3 core constants that exist without any
// user code running `define()`. Lookups against this map are case-sensitive,
// matching real PHP constant semantics (unlike the `true`/`false`/`null`
// keywords, which are case-insensitive and handled separately).
var predefinedConstants = buildPredefinedConstants()

func buildPredefinedConstants() map[string]Value {
	eol := "\n"
	osName := "Linux"
	switch goruntime.GOOS {
	case "windows":
		eol = "\r\n"
		osName = "WINNT"
	case "darwin":
		osName = "Darwin"
	case "linux":
		osName = "Linux"
	default:
		osName = goruntime.GOOS
	}

	return map[string]Value{
		"PHP_VERSION":         StringValue("5.3.29"),
		"PHP_MAJOR_VERSION":   IntValue(5),
		"PHP_MINOR_VERSION":   IntValue(3),
		"PHP_RELEASE_VERSION": IntValue(29),
		"PHP_EXTRA_VERSION":   StringValue(""),
		"PHP_VERSION_ID":      IntValue(50329),
		"PHP_OS":              StringValue(osName),
		"PHP_SAPI":            StringValue("cli"),
		"PHP_EOL":             StringValue(eol),
		"PHP_INT_MAX":         IntValue(math.MaxInt64),
		"PHP_INT_SIZE":        IntValue(8),
		"DIRECTORY_SEPARATOR": StringValue(string(os.PathSeparator)),
		"PATH_SEPARATOR":      StringValue(string(os.PathListSeparator)),

		"M_PI":       FloatValue(math.Pi),
		"M_E":        FloatValue(math.E),
		"M_LOG2E":    FloatValue(math.Log2E),
		"M_LOG10E":   FloatValue(math.Log10E),
		"M_LN2":      FloatValue(math.Ln2),
		"M_LN10":     FloatValue(math.Ln10),
		"M_PI_2":     FloatValue(math.Pi / 2),
		"M_PI_4":     FloatValue(math.Pi / 4),
		"M_1_PI":     FloatValue(1 / math.Pi),
		"M_2_PI":     FloatValue(2 / math.Pi),
		"M_SQRTPI":   FloatValue(math.SqrtPi),
		"M_2_SQRTPI": FloatValue(2 / math.SqrtPi),
		"M_SQRT2":    FloatValue(math.Sqrt2),
		"M_SQRT3":    FloatValue(math.Sqrt(3)),
		"M_SQRT1_2":  FloatValue(1 / math.Sqrt2),
		"M_LNPI":     FloatValue(math.Log(math.Pi)),
		"M_EULER":    FloatValue(0.57721566490153286061),
		"NAN":        FloatValue(math.NaN()),
		"INF":        FloatValue(math.Inf(1)),

		"E_ERROR":             IntValue(1),
		"E_WARNING":           IntValue(2),
		"E_PARSE":             IntValue(4),
		"E_NOTICE":            IntValue(8),
		"E_CORE_ERROR":        IntValue(16),
		"E_CORE_WARNING":      IntValue(32),
		"E_COMPILE_ERROR":     IntValue(64),
		"E_COMPILE_WARNING":   IntValue(128),
		"E_USER_ERROR":        IntValue(256),
		"E_USER_WARNING":      IntValue(512),
		"E_USER_NOTICE":       IntValue(1024),
		"E_STRICT":            IntValue(2048),
		"E_RECOVERABLE_ERROR": IntValue(4096),
		"E_DEPRECATED":        IntValue(8192),
		"E_USER_DEPRECATED":   IntValue(16384),
		"E_ALL":               IntValue(32767),
	}
}
