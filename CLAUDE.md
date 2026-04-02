# CLAUDE.md

## Build and Test

- `just check` -- run all checks (test + lint + fmt-check). Used by CI and pre-commit hook.
- `just test` -- run tests only. Pass args: `just test -v -run TestFoo`
- `just lint` -- run golangci-lint
- `just fmt` -- format all Go files with gofmt -s
- `just build` -- build the binary to ./ganda
- `just install` -- install to GOPATH/bin
- `just clean` -- remove build artifacts (use this, never bare rm -rf)
- `just tidy` -- run go mod tidy
- `just update-deps` -- update all dependencies
- `just bench-go` -- run Go micro-benchmarks (parser, responses)
- `just bench` -- run end-to-end throughput benchmark via hyperfine

Red/green testing: write a failing test before implementing, then make it pass.
All commits should pass `just check`.

## Release

- `just bump 1.0.4` -- create annotated tag with release notes and push to trigger release workflow
- `just retag 1.0.4` -- delete and re-tag to re-trigger a failed release

Version is injected via ldflags at build time (no version file). GoReleaser handles
this for releases. Tag convention: v-prefixed semver (e.g., v1.0.3).

## Architecture

ganda pipes URLs (or JSON request specs) from stdin through parallel HTTP workers
and emits responses to stdout, with status logging to stderr.

Packages:
- **cli/** -- urfave/cli v3 command setup, flag parsing, request/response orchestration
- **config/** -- configuration struct with defaults, flag value types
- **parser/** -- reads stdin (URL list or JSON lines), creates http.Request objects
- **requests/** -- HTTP client with retry/backoff, request worker goroutines
- **responses/** -- response processing (raw, base64, sha256, escaped, JSON envelope), file saving
- **execcontext/** -- bridges config to runtime context (timeouts, logger, IO handles)
- **logger/** -- leveled logger with optional color output
- **echoserver/** -- embedded echo server for testing/demos (labstack/echo v4)

## Conventions

- Tests use httptest.Server stubs and assert with stretchr/testify.
- All packages have test coverage.
- JSON-line input supports up to 1MB per line (scanner buffer limit).
