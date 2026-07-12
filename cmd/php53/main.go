package main

import (
	"fmt"
	"os"
	"strings"

	"php-src-go/internal/lexer"
	"php-src-go/internal/parser"
	"php-src-go/internal/runtime"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: php53 <file.php>")
		os.Exit(2)
	}
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	p := parser.New(lexer.New(string(data)))
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		fmt.Fprintln(os.Stderr, strings.Join(p.Errors(), "\n"))
		os.Exit(1)
	}
	interp := runtime.NewInterpreter()
	if err := interp.Run(program); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Print(interp.Out.String())
}
