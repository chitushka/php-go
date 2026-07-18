package runtime

type ErrorSeverity string

const (
	ParseError       ErrorSeverity = "parse"
	FatalError       ErrorSeverity = "fatal"
	RecoverableError ErrorSeverity = "error"
	Warning          ErrorSeverity = "warning"
	Notice           ErrorSeverity = "notice"
)

type ErrorMask uint32

const (
	ErrorMaskParse ErrorMask = 1 << iota
	ErrorMaskFatal
	ErrorMaskError
	ErrorMaskWarning
	ErrorMaskNotice

	ErrorMaskAll = ErrorMaskParse | ErrorMaskFatal | ErrorMaskError | ErrorMaskWarning | ErrorMaskNotice
)

type ErrorPolicy struct {
	Report ErrorMask
	Halt   ErrorMask
}

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

func DefaultErrorPolicy() ErrorPolicy {
	return ErrorPolicy{
		Report: ErrorMaskAll,
		Halt:   ErrorMaskParse | ErrorMaskFatal | ErrorMaskError,
	}
}

func StrictErrorPolicy() ErrorPolicy {
	return ErrorPolicy{
		Report: ErrorMaskAll,
		Halt:   ErrorMaskAll,
	}
}

func (p ErrorPolicy) Ignore(severities ...ErrorSeverity) ErrorPolicy {
	for _, severity := range severities {
		mask := SeverityErrorMask(severity)
		p.Report &^= mask
		p.Halt &^= mask
	}
	return p
}

func (p ErrorPolicy) DoNotHalt(severities ...ErrorSeverity) ErrorPolicy {
	for _, severity := range severities {
		p.Halt &^= SeverityErrorMask(severity)
	}
	return p
}

func (p ErrorPolicy) HaltOn(severities ...ErrorSeverity) ErrorPolicy {
	for _, severity := range severities {
		mask := SeverityErrorMask(severity)
		p.Report |= mask
		p.Halt |= mask
	}
	return p
}

func (p ErrorPolicy) Reports(severity ErrorSeverity) bool {
	return p.Report&SeverityErrorMask(severity) != 0
}

func (p ErrorPolicy) Halts(severity ErrorSeverity) bool {
	return p.Halt&SeverityErrorMask(severity) != 0
}

func SeverityErrorMask(severity ErrorSeverity) ErrorMask {
	switch severity {
	case ParseError:
		return ErrorMaskParse
	case FatalError:
		return ErrorMaskFatal
	case RecoverableError:
		return ErrorMaskError
	case Warning:
		return ErrorMaskWarning
	case Notice:
		return ErrorMaskNotice
	default:
		return 0
	}
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
