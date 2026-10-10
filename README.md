# php-go

[![CI](https://github.com/chitushka/php-go/actions/workflows/ci.yml/badge.svg)](https://github.com/chitushka/php-go/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A PHP 5.3 compatibility interpreter written in Go. The project provides a
lexer, recursive-descent parser, AST, runtime, command-line interpreter, and a
PHPT-compatible acceptance-test runner.

php-go is useful for compatibility research, controlled legacy-code execution,
and experimentation. It implements a substantial, tested PHP 5.3 subset; it is
not a drop-in replacement for the native PHP runtime or its extension library.

## Requirements

- Go 1.25.6 or newer
- Windows, Linux, or macOS

The project has no third-party Go dependencies.

## Install and run

Run directly from a clone:

```sh
go run ./cmd/php53 examples/hello.php
```

Or install both commands:

```sh
go install github.com/chitushka/php-go/cmd/php53@latest
go install github.com/chitushka/php-go/cmd/php53test@latest
php53 path/to/program.php
```

Build local binaries with:

```sh
go build -o bin/php53 ./cmd/php53
go build -o bin/php53test ./cmd/php53test
```

On Windows, the produced files have an `.exe` suffix.

## Supported compatibility surface

The tested subset includes:

- PHP source modes, scalar literals, interpolation, heredoc/nowdoc, and
  backticks;
- variables, arrays, references, expressions, casts, and PHP 5.x-style type
  juggling;
- control flow and alternative control-structure syntax;
- functions, closures, namespaces, constants, includes, and error policy;
- classes, interfaces, inheritance, visibility, late static binding, magic
  methods, and exceptions;
- common string, array, math, date, filesystem, regex, JSON, hash, and
  serialization builtins;
- basic SAPI state, headers, superglobals, and multipart upload handling.

The executable compatibility contract is the focused suite in `tests/phpt`.

## Known limitations

- Only the documented and tested PHP 5.3 subset is supported.
- `declare` and `eval` are not implemented; `eval` is intentionally deferred
  because of its security model.
- Extension parity is out of scope. Builtins are implemented selectively.
- Diagnostics and edge-case behavior can differ from the Zend Engine outside
  the PHPT contract.
- `tests/custom` is an exploratory corpus and includes unsupported constructs
  such as `goto`; it is not the release gate.

## Security

The interpreter is **not a sandbox**. PHP programs can access the filesystem
and backticks execute commands through the host shell. Run only trusted source,
or isolate the process using operating-system or container controls. See
[`SECURITY.md`](SECURITY.md) for reporting and deployment guidance.

## Development and verification

```sh
gofmt -w path/to/edited.go
go vet ./...
go test ./...
go run ./cmd/php53test tests/phpt
go build ./cmd/php53 ./cmd/php53test
```

CI runs these checks on Linux and Windows and runs the Go race detector on
Linux. Every language change should include one focused PHPT test documenting
the expected PHP 5.3 behavior.

See [`CONTRIBUTING.md`](CONTRIBUTING.md) before opening a pull request and
[`CHANGELOG.md`](CHANGELOG.md) for release notes.

## License

Licensed under the [MIT License](LICENSE).
