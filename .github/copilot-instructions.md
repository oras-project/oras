# ORAS CLI 

ORAS (OCI Registry As Storage) is a CLI tool for working with OCI artifacts and registries. It's a Go application that provides commands for pushing, pulling, copying, and managing OCI artifacts.

Always reference these instructions first and fallback to search or bash commands only when you encounter unexpected information that does not match the info here.

## Working Effectively

### Bootstrap and Build
- Install Go 1.25.0 or later (the project requires Go 1.25.0 as specified in go.mod)
- Prepare the environment:
  ```bash
  make tidy    # Download dependencies - takes ~3 seconds
  make vendor  # Vendor dependencies - takes ~1 second  
  ```
- Build the CLI:
  ```bash
  make build-linux-amd64  # Build for Linux AMD64 - takes ~15 seconds
  ```
  - NEVER CANCEL: Build takes 15 seconds. Set timeout to 60+ seconds.
  - Binary will be created at `bin/linux/amd64/oras`

### Testing
- Run unit tests:
  ```bash
  make test  # Run all unit tests - takes ~40 seconds. NEVER CANCEL. Set timeout to 120+ seconds.
  ```
  - NEVER CANCEL: Unit tests take 40 seconds. Set timeout to 120+ seconds.
  - **EXPECTED BEHAVIOR**: Tests may show warnings about missing "covdata" tool and make will exit with error code 1, but the actual tests pass successfully
  - This is due to Go 1.25 compatibility issues with coverage tooling
  - Coverage reports are generated in `coverage.txt` despite the error
  - Individual test results will show "PASS" for all test packages

- Run E2E tests (requires Docker):
  ```bash
  # Install ginkgo first if not available
  go install github.com/onsi/ginkgo/v2/ginkgo@latest
  export PATH=$(go env GOPATH)/bin:$PATH
  
  make teste2e  # Run end-to-end tests - takes 15-20+ minutes. NEVER CANCEL. Set timeout to 1800+ seconds.
  ```
  - NEVER CANCEL: E2E tests take 15-20+ minutes and require downloading Docker images. Set timeout to 1800+ seconds.
  - E2E tests start multiple registry containers (oras-distribution, upstream distribution, zot)
  - Tests are designed for Linux platform only
  - Initial run will download large Docker images (~100MB+ each) which takes additional time

### Lint and Code Quality  
- Check code formatting:
  ```bash
  make check-encoding  # Check CR/LF encoding - takes ~2 seconds
  ```
- Run linting:
  ```bash
  make lint  # Run golangci-lint 
  ```
  - **KNOWN ISSUE**: golangci-lint may fail with Go 1.25 due to version compatibility: "the Go language version (go1.23) used to build golangci-lint is lower than the targeted Go version (1.25.0)"
  - The GitHub Actions workflow uses a compatible version of golangci-lint
  - If linting fails locally, rely on the CI pipeline for lint validation

## Validation Scenarios

ALWAYS test CLI functionality after making changes by running through these validation scenarios:

### Basic CLI Validation
1. **Version Check**: `./bin/linux/amd64/oras version` - should show version, Go version, OS/Arch, git commit
2. **Help Command**: `./bin/linux/amd64/oras --help` - should show all available commands  
3. **Command Help**: `./bin/linux/amd64/oras push --help` - should show detailed command usage

### Core Functionality Testing
1. **Basic Commands Test**:
   ```bash
   # Test version and help commands  
   ./bin/linux/amd64/oras version
   ./bin/linux/amd64/oras --help
   ./bin/linux/amd64/oras push --help
   ./bin/linux/amd64/oras pull --help
   ```

2. **Argument Validation Test**:
   ```bash
   # Create test file
   mkdir -p /tmp/oras-test
   echo "test content for oras validation" > /tmp/oras-test/sample.txt
   
   # Test path validation (should show helpful error about absolute paths)
   ./bin/linux/amd64/oras push localhost:5000/test:v1 /tmp/oras-test/sample.txt
   # Expected output: "Error: absolute file path detected. If it's intentional, use --disable-path-validation flag..."
   
   # Test with relative path
   cd /tmp/oras-test
   /home/runner/work/oras/oras/bin/linux/amd64/oras push localhost:5000/test:v1 sample.txt
   # Expected: connection error since no registry is running - this confirms CLI parsing works
   ```

3. **Command Discovery Test**:
   ```bash
   # Verify all main commands are available
   ./bin/linux/amd64/oras --help | grep -E "(push|pull|cp|attach|backup|restore|login|logout)"
   # Should show all core commands
   ```

## Build System Details

### Makefile Targets
- `make default` - Runs `test build-$(OS)-$(ARCH)` (default target)
- `make build` - Build for all platforms (Linux, Mac, Windows)
- `make build-linux-amd64` - Build Linux AMD64 binary only  
- `make test` - Run unit tests with race detection and coverage
- `make teste2e` - Run end-to-end tests
- `make clean` - Clean build artifacts
- `make help` - Show all available targets

### Build Timing Expectations
- **Dependency setup**: `make tidy` + `make vendor` = ~4 seconds total
- **Single platform build**: `make build-linux-amd64` = ~15 seconds. NEVER CANCEL.
- **Unit tests**: `make test` = ~40 seconds. NEVER CANCEL.
- **E2E tests**: `make teste2e` = 15-20+ minutes. NEVER CANCEL.
- **Encoding check**: `make check-encoding` = ~2 seconds

### Project Structure
- `cmd/oras/` - CLI entry point and command implementations
- `internal/` - Internal packages (cache, credential, docker, crypto, etc.)
- `test/e2e/` - End-to-end tests using Ginkgo framework
- `Makefile` - Build system with cross-platform targets
- `go.mod` - Go module with dependencies (requires Go 1.25.0)

### Key Codebase Locations
- `cmd/oras/main.go` - CLI application entry point
- `cmd/oras/root/` - Root command and all subcommands (push, pull, etc.)
- `internal/version/` - Version information and build metadata
- `internal/credential/` - Registry authentication logic
- `internal/docker/` - Docker credential helper integration
- `test/e2e/suite/` - E2E test suites (command, auth, scenario)
- `.github/workflows/build.yml` - Main CI/CD pipeline
- `test/e2e/README.md` - Comprehensive E2E testing guide

### Key Dependencies
- `oras.land/oras-go/v2` - Core ORAS Go library
- `github.com/spf13/cobra` - CLI framework
- `github.com/opencontainers/image-spec` - OCI image spec
- `github.com/sirupsen/logrus` - Logging
- Testing: Ginkgo/Gomega for E2E tests

### CI/CD Integration
Always run these commands before submitting changes:
1. `make check-encoding` - Verify file encoding
2. `make test` - Unit tests must pass  
3. `make build-linux-amd64` - Verify build succeeds
4. Run validation scenarios above to test CLI functionality
5. `make lint` - Code linting (may fail locally with Go 1.25 but will work in CI)

The GitHub Actions workflow (`.github/workflows/build.yml`) runs these same checks on Go 1.25.

## Important Notes

- **Platform Support**: E2E tests only run on Linux platform
- **Docker Requirement**: E2E tests require Docker to run registry containers
- **Go Version**: Project strictly requires Go 1.25.0 as specified in go.mod
- **Binary Location**: Built binaries are placed in `bin/{os}/{arch}/oras` (Linux) or `bin/{os}/{arch}/oras.exe` (Windows)
- **Path Validation**: CLI requires `--disable-path-validation` flag when using absolute file paths

## Common Commands Reference

### Quick Start Workflow
```bash
# Setup
make tidy && make vendor

# Build and test  
make build-linux-amd64  # 15 seconds
make test                # 40 seconds  

# Validate functionality
./bin/linux/amd64/oras version
./bin/linux/amd64/oras --help
```

### Repository Root Structure
```
/home/runner/work/oras/oras/
├── .github/          # GitHub workflows and templates
├── cmd/              # CLI commands and entry point
├── internal/         # Internal Go packages
├── test/            # Test suites
├── docs/            # Documentation
├── Makefile         # Build system
├── go.mod           # Go module definition
└── bin/             # Built binaries (created after build)
```

## Summary for Quick Reference

**Essential Commands** (copy-paste ready):
```bash
# Complete setup from fresh clone
make tidy && make vendor                    # ~4 seconds
make build-linux-amd64                     # ~15 seconds, NEVER CANCEL
make test                                   # ~40 seconds, NEVER CANCEL (expect error code 1 but tests pass)
./bin/linux/amd64/oras version             # Validate build works
```

**Key Validation After Changes**:
```bash
make check-encoding                         # File encoding check
make test                                   # Unit tests (ignore final error, check individual PASS results)
./bin/linux/amd64/oras version && ./bin/linux/amd64/oras --help  # CLI validation
```

**Important Notes**:
- Go 1.25.0 required
- Unit tests pass but `make test` exits with error code 1 due to covdata tool compatibility
- golangci-lint may fail locally with Go 1.25 but works in CI
- E2E tests take 15-20+ minutes and require Docker
- All timeout values include buffer time - NEVER CANCEL long operations

Always build and exercise your changes using the validation scenarios above. The CLI provides comprehensive help via `--help` flags on any command.