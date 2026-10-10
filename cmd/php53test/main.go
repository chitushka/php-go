package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/chitushka/php-go/internal/phpt"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: php53test <file.phpt|directory> [...]")
		os.Exit(2)
	}

	paths, err := collect(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if len(paths) == 0 {
		fmt.Fprintln(os.Stderr, "no .phpt files found")
		os.Exit(1)
	}

	failed := 0
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			failed++
			fmt.Printf("FAIL %s\n  %v\n", path, err)
			continue
		}
		result := phpt.Run(path, string(data))
		if result.Skipped {
			fmt.Printf("SKIP %s - %s", path, result.Name)
			if result.SkipReason != "" {
				fmt.Printf(" (%s)", result.SkipReason)
			}
			fmt.Println()
			continue
		}
		if result.Passed {
			fmt.Printf("PASS %s - %s\n", path, result.Name)
			continue
		}
		failed++
		fmt.Printf("FAIL %s - %s\n", path, result.Name)
		if result.Err != nil {
			fmt.Printf("  error: %v\n", result.Err)
		}
		fmt.Printf("  expected: %q\n", result.Expected)
		fmt.Printf("  got:      %q\n", result.Output)
	}

	if failed != 0 {
		os.Exit(1)
	}
}

func collect(args []string) ([]string, error) {
	var out []string
	for _, arg := range args {
		info, err := os.Stat(arg)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			out = append(out, arg)
			continue
		}
		err = filepath.WalkDir(arg, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if filepath.Ext(path) == ".phpt" {
				out = append(out, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}
