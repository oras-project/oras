# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

ORAS (OCI Registry As Storage) is a command-line tool for pushing, pulling, and managing OCI artifacts to/from OCI-compliant registries. It's built in Go and uses the Cobra CLI framework.

## Development Commands

### Building
```bash
# Build for current platform
make

# Build for specific platforms
make build-linux-amd64
make build-mac-arm64
make build-windows-amd64

# Build for all platforms
make build
```

### Testing
```bash
# Run unit tests with coverage
make test

# Run end-to-end tests
make teste2e

# View coverage report
make covhtml
```

### Code Quality
```bash
# Run linter
make lint

# Fix file encoding issues
make fix-encoding

# Tidy and vendor dependencies
make tidy
make vendor
```

### Development Workflow
```bash
# Default development build and test
make

# Clean build artifacts
make clean
```

## Architecture

### CLI Structure
- **Entry point**: `cmd/oras/main.go` - minimal main function that delegates to root command
- **Root command**: `cmd/oras/root/cmd.go` - defines the main `oras` command and subcommands
- **Subcommands**: Located in `cmd/oras/root/` directory:
  - Core commands: `pull.go`, `push.go`, `login.go`, `logout.go`, `tag.go`, `attach.go`
  - Backup/restore: `backup.go`, `restore.go`
  - Utility commands: `discover.go`, `resolve.go`, `cp.go`, `version.go`
  - Nested commands: `blob/`, `manifest/`, `repo/` subdirectories

### Internal Packages
Key internal packages in `internal/`:
- **credential**: Credential store management for registry authentication
- **contentutil**: Content utilities for handling OCI artifacts
- **progress**: Progress reporting and display
- **tree**: Tree structure printing for artifact relationships
- **trace**: Tracing and logging utilities
- **version**: Version information and build metadata
- **io**: I/O utilities and file operations
- **graph**: Dependency graph operations
- **cache**: Caching functionality

### Dependencies
- Uses `oras.land/oras-go/v2` as the core library for OCI operations
- Built with Cobra CLI framework (`github.com/spf13/cobra`)
- Uses `github.com/opencontainers/image-spec` for OCI specification compliance

### Testing
- Unit tests: `*_test.go` files alongside source code
- E2E tests: Located in `test/e2e/` with shell scripts
- Test scripts: `test/e2e/scripts/e2e.sh` for end-to-end testing
- Coverage reporting: Generated in `coverage.txt`

### Build System
- Uses Makefile for build orchestration
- Cross-platform builds for Linux, macOS, Windows, FreeBSD
- Support for multiple architectures: amd64, arm64, armv7, s390x, ppc64le, riscv64, loong64
- GoReleaser configuration in `.goreleaser.yml`
- Build metadata injection via ldflags for version information

## Key Design Patterns

- **Command pattern**: Each CLI command is implemented as a separate function returning `*cobra.Command`
- **Internal packages**: All business logic separated into internal packages to prevent external imports
- **Dependency injection**: Commands accept options/configurations rather than using globals
- **Error handling**: Consistent error handling patterns throughout the codebase
- **Testing**: Comprehensive unit and integration test coverage