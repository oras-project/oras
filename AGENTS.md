# GitHub Copilot Instructions

## Project
ORAS (OCI Registry As Storage) - Go CLI tool for OCI artifact management using Cobra framework.

## Build & Test
```bash
make              # Build for current platform
make test         # Unit tests with coverage
make teste2e      # E2E tests
make lint         # Run linter
```

## Repository Structure
```
cmd/oras/           # CLI entry and commands
├── main.go         # Minimal entry point
└── root/           # Root command and subcommands
    ├── *.go        # Core commands (pull, push, login, tag, etc.)
    ├── blob/       # Blob operations (push, fetch, delete)
    ├── manifest/   # Manifest operations (push, fetch, delete)
    └── repo/       # Repository operations (ls, tags)

internal/           # Internal packages (non-importable)
├── credential/     # Registry authentication
├── contentutil/    # Artifact content handling
├── progress/       # Progress display
├── tree/           # Artifact tree printing
├── cache/          # Caching layer
├── graph/          # Dependency graphs
└── io/             # I/O utilities

test/e2e/           # End-to-end tests
docs/               # Documentation
vendor/             # Vendored dependencies
```

## Architecture
- **Entry**: `cmd/oras/main.go` → `cmd/oras/root/cmd.go`
- **Commands**: `cmd/oras/root/*.go` and `cmd/oras/root/{blob,manifest,repo}/`
- **Internal**: Business logic in `internal/` (credential, contentutil, progress, tree, trace)
- **Core lib**: `oras.land/oras-go/v2`

## Patterns
- Command pattern with `*cobra.Command`
- Internal packages only (no external imports)
- Dependency injection over globals
- Tests alongside source (`*_test.go`)
- E2E tests in `test/e2e/`

## Key Files
- `Makefile`: Build orchestration
- `.goreleaser.yml`: Release configuration
- Version injection via ldflags
