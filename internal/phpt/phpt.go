package phpt

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"php-src-go/internal/lexer"
	"php-src-go/internal/parser"
	"php-src-go/internal/runtime"
)

type Test struct {
	Name   string
	File   string
	Expect string
}

type Result struct {
	Path     string
	Name     string
	Passed   bool
	Output   string
	Expected string
	Err      error
}

var sectionHeader = regexp.MustCompile(`^--([A-Z]+)--\s*$`)

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

	test := &Test{Name: strings.TrimSpace(sections["TEST"]), File: sections["FILE"], Expect: sections["EXPECT"]}
	if test.Name == "" {
		test.Name = "unnamed test"
	}
	if _, ok := sections["FILE"]; !ok {
		return nil, fmt.Errorf("missing --FILE-- section")
	}
	if _, ok := sections["EXPECT"]; !ok {
		return nil, fmt.Errorf("missing --EXPECT-- section")
	}
	return test, nil
}

func Run(path string, input string) Result {
	test, err := Parse(input)
	if err != nil {
		return Result{Path: path, Name: filepath.Base(path), Err: err}
	}

	p := parser.New(lexer.New(test.File))
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		return Result{Path: path, Name: test.Name, Expected: test.Expect, Err: fmt.Errorf("parse errors: %v", p.Errors())}
	}

	interp := runtime.NewInterpreterWithBaseDir(filepath.Dir(path))
	err = interp.Run(program)
	output := interp.Out.String()
	passed := err == nil && comparable(output) == comparable(test.Expect)
	return Result{Path: path, Name: test.Name, Passed: passed, Output: output, Expected: test.Expect, Err: err}
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
