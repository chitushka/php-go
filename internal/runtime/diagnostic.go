package runtime

type ErrorSeverity string

const (
	ParseError ErrorSeverity = "parse"
	FatalError ErrorSeverity = "fatal"
	Warning    ErrorSeverity = "warning"
	Notice     ErrorSeverity = "notice"
)

type RuntimeError struct {
	Severity ErrorSeverity
	Message  string
}

func (e *RuntimeError) Error() string {
	return e.Message
}

type Diagnostic struct {
	Severity ErrorSeverity
	Message  string
}

func newRuntimeError(severity ErrorSeverity, message string) error {
	return &RuntimeError{Severity: severity, Message: message}
}

func errorSeverity(err error) (ErrorSeverity, bool) {
	if typed, ok := err.(*RuntimeError); ok {
		return typed.Severity, true
	}
	return "", false
}
