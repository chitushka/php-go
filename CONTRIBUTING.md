# Contributing

Thank you for helping improve php-go. Keep changes focused and preserve PHP
5.3 behavior rather than adding features from later PHP releases.

## Development workflow

1. Add or update a focused test. Language behavior belongs in `tests/phpt`.
2. Implement the smallest complete vertical slice.
3. Format edited Go files with `gofmt -w`.
4. Run the release checks:

   ```sh
   go vet ./...
   go test ./...
   go run ./cmd/php53test tests/phpt
   go build ./cmd/php53 ./cmd/php53test
   ```

For language compatibility questions, compare behavior with the PHP 5.3
scanner/parser sources and document intentional differences. Pull requests
should explain user-visible behavior, tests, and any security implications.
