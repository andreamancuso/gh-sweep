# Development reference

This file keeps maintainer-facing commands out of the README. The README should stay focused on what the tool does and how to use it.

## Requirements

- Go 1.25.0 or newer
- Git
- GitHub CLI (`gh`) for live GitHub API usage

Authenticate before running commands that call GitHub:

```bash
gh auth login
```

For package inspection or deletion, refresh package scopes:

```bash
gh auth refresh -s read:packages -s delete:packages
```

Automation can also use `GH_TOKEN` or `GITHUB_TOKEN`.

## Build

Build all packages:

```bash
go build ./...
```

Build the local binary:

```bash
go build -o gh-sweep
```

Run the CLI from source:

```bash
go run . --help
go run . storage --repo owner/repo --list
```

## Tests

Run the full test suite:

```bash
go test ./...
```

Run tests for one package:

```bash
go test ./internal/github/...
go test ./cmd/...
```

Run one test by name:

```bash
go test ./internal/github -run TestStorage
```

Run with coverage:

```bash
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Static checks

Run Go vet:

```bash
go vet ./...
```

Run vulnerability scanning:

```bash
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

Run security linting:

```bash
go run github.com/securego/gosec/v2/cmd/gosec@latest ./...
```

Run whitespace checks before committing:

```bash
git diff --check
```

## Dependency maintenance

List outdated direct and transitive dependencies:

```bash
go list -m -u all
```

Tidy module files after dependency changes:

```bash
go mod tidy
```

## Suggested pre-commit check

For code changes, run:

```bash
go test ./...
go vet ./...
go build ./...
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
go run github.com/securego/gosec/v2/cmd/gosec@latest ./...
git diff --check
```

For documentation-only changes, `git diff --check` is usually enough.
