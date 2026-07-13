package phpt

import (
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"php-src-go/internal/lexer"
	"php-src-go/internal/parser"
	"php-src-go/internal/runtime"
)

type Test struct {
	Name        string
	File        string
	SkipIf      string
	Expect      string
	ExpectF     string
	ExpectRegex string
	INI         string
	Clean       string
	Get         string
	Post        string
	Env         string
	ExitCode    int
	HasExitCode bool
}

type sapiContext struct {
	Get  []sapiEntry
	Post []sapiEntry
	Env  []sapiEntry
}

type sapiEntry struct {
	Key   string
	Value string
}

type Result struct {
	Path       string
	Name       string
	Passed     bool
	Skipped    bool
	SkipReason string
	Output     string
	Expected   string
	Err        error
}

var sectionHeader = regexp.MustCompile(`^--([A-Z_]+)--\s*$`)

func Parse(input string) (*Test, error) {
	sections := map[string]string{}
	current := ""
	var b strings.Builder

	lines := strings.SplitAfter(normalizeLineEndings(input), "\n")
	for _, line := range lines {
		trimmed := strings.TrimRight(line, "\n")
		if match := sectionHeader.FindStringSubmatch(trimmed); match != nil {
			if current != "" {
				sections[current] = trimSection(b.String())
				b.Reset()
			}
			current = match[1]
			continue
		}
		if current != "" {
			b.WriteString(line)
		}
	}
	if current != "" {
		sections[current] = trimSection(b.String())
	}

	test := &Test{
		Name:        strings.TrimSpace(sections["TEST"]),
		File:        sections["FILE"],
		SkipIf:      sections["SKIPIF"],
		Expect:      sections["EXPECT"],
		ExpectF:     sections["EXPECTF"],
		ExpectRegex: sections["EXPECTREGEX"],
		INI:         sections["INI"],
		Clean:       sections["CLEAN"],
		Get:         sections["GET"],
		Post:        sections["POST"],
		Env:         sections["ENV"],
	}
	if exitCode, ok := sections["EXIT_CODE"]; ok && strings.TrimSpace(exitCode) != "" {
		parsed, err := strconv.Atoi(strings.TrimSpace(exitCode))
		if err != nil {
			return nil, fmt.Errorf("invalid --EXIT_CODE-- value %q", strings.TrimSpace(exitCode))
		}
		test.ExitCode = parsed
		test.HasExitCode = true
	}
	if test.Name == "" {
		test.Name = "unnamed test"
	}
	if _, ok := sections["FILE"]; !ok {
		return nil, fmt.Errorf("missing --FILE-- section")
	}
	if _, ok := sections["EXPECT"]; !ok {
		if _, ok := sections["EXPECTF"]; !ok {
			if _, ok := sections["EXPECTREGEX"]; !ok {
				return nil, fmt.Errorf("missing --EXPECT--, --EXPECTF-- or --EXPECTREGEX-- section")
			}
		}
	}
	return test, nil
}

func Run(path string, input string) Result {
	test, err := Parse(input)
	if err != nil {
		return Result{Path: path, Name: filepath.Base(path), Err: err}
	}

	if test.SkipIf != "" {
		output, err := runPHP(path, test.SkipIf, test.sapiContext())
		if err != nil {
			return Result{Path: path, Name: test.Name, Err: fmt.Errorf("SKIPIF error: %w", err)}
		}
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(output)), "skip") {
			return Result{Path: path, Name: test.Name, Skipped: true, SkipReason: strings.TrimSpace(output)}
		}
	}

	output, err := runPHP(path, test.File, test.sapiContext())
	exitCode := 0
	if err != nil {
		output += err.Error()
		exitCode = 1
	}
	expected := test.Expect
	passed := false
	if test.ExpectRegex != "" {
		expected = test.ExpectRegex
		passed = expectRegexMatches(output, test.ExpectRegex)
	} else if test.ExpectF != "" {
		expected = test.ExpectF
		passed = expectFMatches(output, test.ExpectF)
	} else {
		passed = comparable(output) == comparable(test.Expect)
	}
	if test.HasExitCode && exitCode != test.ExitCode {
		passed = false
		if err == nil {
			err = fmt.Errorf("exit code mismatch: expected %d, got %d", test.ExitCode, exitCode)
		}
	}
	if test.Clean != "" {
		if _, cleanErr := runPHP(path, test.Clean, test.sapiContext()); cleanErr != nil {
			passed = false
			if err == nil {
				err = fmt.Errorf("CLEAN error: %w", cleanErr)
			}
		}
	}
	return Result{Path: path, Name: test.Name, Passed: passed, Output: output, Expected: expected, Err: err}
}

func (t *Test) sapiContext() sapiContext {
	return sapiContext{
		Get:  parseQuerySection(t.Get),
		Post: parseQuerySection(t.Post),
		Env:  parseEnvSection(t.Env),
	}
}

func runPHP(path string, source string, ctx sapiContext) (string, error) {
	p := parser.New(lexer.New(source))
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		return "", fmt.Errorf("parse errors: %v", p.Errors())
	}

	interp := runtime.NewInterpreterWithBaseDir(filepath.Dir(path))
	installSAPIContext(interp, ctx)
	err := interp.Run(program)
	return interp.Out.String(), err
}

func installSAPIContext(interp *runtime.Interpreter, ctx sapiContext) {
	get := arrayFromEntries(ctx.Get)
	post := arrayFromEntries(ctx.Post)
	env := arrayFromEntries(ctx.Env)
	request := runtime.NewArray()
	copyArrayEntries(request, get)
	copyArrayEntries(request, post)

	server := runtime.NewArray()
	server.Set(runtime.StringKey("REQUEST_METHOD"), runtime.StringValue(requestMethod(ctx)))
	server.Set(runtime.StringKey("QUERY_STRING"), runtime.StringValue(encodeValues(ctx.Get)))

	interp.SetGlobal("_GET", runtime.ArrayValue(get))
	interp.SetGlobal("_POST", runtime.ArrayValue(post))
	interp.SetGlobal("_REQUEST", runtime.ArrayValue(request))
	interp.SetGlobal("_ENV", runtime.ArrayValue(env))
	interp.SetGlobal("_SERVER", runtime.ArrayValue(server))
}

func requestMethod(ctx sapiContext) string {
	if len(ctx.Post) != 0 {
		return "POST"
	}
	return "GET"
}

func arrayFromEntries(values []sapiEntry) *runtime.Array {
	arr := runtime.NewArray()
	for _, entry := range values {
		arr.Set(runtime.StringKey(entry.Key), runtime.StringValue(entry.Value))
	}
	return arr
}

func copyArrayEntries(dst *runtime.Array, src *runtime.Array) {
	for _, entry := range src.Entries {
		dst.Set(entry.Key, entry.Value)
	}
}

func parseQuerySection(section string) []sapiEntry {
	var out []sapiEntry
	section = strings.TrimSpace(section)
	if section == "" {
		return out
	}
	for _, part := range strings.Split(section, "&") {
		if part == "" {
			continue
		}
		key, value, _ := strings.Cut(part, "=")
		decodedKey, err := url.QueryUnescape(key)
		if err != nil {
			decodedKey = key
		}
		decodedValue, err := url.QueryUnescape(value)
		if err != nil {
			decodedValue = value
		}
		out = append(out, sapiEntry{Key: decodedKey, Value: decodedValue})
	}
	return out
}

func parseEnvSection(section string) []sapiEntry {
	var out []sapiEntry
	for _, line := range strings.Split(normalizeLineEndings(section), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			out = append(out, sapiEntry{Key: line})
			continue
		}
		out = append(out, sapiEntry{Key: strings.TrimSpace(key), Value: strings.TrimSpace(value)})
	}
	return out
}

func encodeValues(values []sapiEntry) string {
	query := url.Values{}
	for _, entry := range values {
		query.Set(entry.Key, entry.Value)
	}
	return query.Encode()
}

func comparable(s string) string {
	return strings.TrimRight(normalizeLineEndings(s), "\n")
}

func normalizeLineEndings(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}

func trimSection(s string) string {
	s = normalizeLineEndings(s)
	return strings.TrimPrefix(strings.TrimRight(s, "\n"), "\n")
}

func expectFMatches(output string, pattern string) bool {
	re := regexp.QuoteMeta(comparable(pattern))
	re = strings.ReplaceAll(re, `%s`, `.*`)
	re = strings.ReplaceAll(re, `%a`, `.+`)
	re = strings.ReplaceAll(re, `%d`, `[+-]?\d+`)
	re = strings.ReplaceAll(re, `%i`, `[+-]?\d+`)
	re = strings.ReplaceAll(re, `%f`, `[+-]?(?:\d+\.\d+|\d+)`)
	re = strings.ReplaceAll(re, `%w`, `\s*`)
	re = strings.ReplaceAll(re, `%%`, `%`)
	matched, _ := regexp.MatchString(`(?s)^`+re+`$`, comparable(output))
	return matched
}

func expectRegexMatches(output string, pattern string) bool {
	matched, _ := regexp.MatchString(`(?s)^`+comparable(pattern)+`$`, comparable(output))
	return matched
}
