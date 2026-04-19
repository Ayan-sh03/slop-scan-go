# slop-scan-go

Deterministic CLI for finding AI-associated slop patterns in Go repositories.

## Install

```bash
go install github.com/Ayan-sh03/slop-scan-go@latest
```

Or build from source:

```bash
git clone https://github.com/Ayan-sh03/slop-scan-go.git
cd slop-scan-go
go build -o slop-scan ./cmd/slop-scan
```

## Usage

Scan current directory:
```bash
slop-scan scan .
```

Scan specific path:
```bash
slop-scan scan /path/to/repo
```

Output formats:
```bash
slop-scan scan . --json   # JSON output
slop-scan scan . --lint   # Lint-style output
slop-scan scan .          # Text output (default)
```

## Rules

### Defensive
- `defensive.error-swallowing` - Log-only error handling without proper error propagation
- `defensive.error-obscuring` - Generic error returns that hide context
- `defensive.ignored-error` - Empty error checks that ignore errors

### Structure
- `structure.pass-through-wrappers` - Functions that just delegate to another function
- `comments.placeholder-comments` - TODO, FIXME, HACK and similar placeholder comments

## Configuration

Create `slop-scan.config.json` in the root:

```json
{
  "ignores": ["**/*.generated.go"],
  "rules": {
    "comments.placeholder-comments": { "enabled": false }
  }
}
```

## Project Structure

```
.
├── cmd/slop-scan/        # CLI entry point
├── internal/
│   ├── core/           # Engine, registry, fact store
│   ├── facts/          # Fact providers (AST, functions, etc.)
│   ├── rules/          # Rule implementations
│   ├── reporters/      # Output formatters
│   ├── languages/     # Language plugins
│   └── discovery/     # File discovery
└── internal/types/    # Shared types
```

## Development

Run tests:
```bash
go test ./...
```

Build:
```bash
go build ./...
```

Run locally:
```bash
go run ./cmd/slop-scan scan .
```
