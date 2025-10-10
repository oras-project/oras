# GitHub Copilot Instructions for ORAS

## Project Overview

ORAS (OCI Registry As Storage) is a CLI tool for working with OCI registries. It enables users to push, pull, copy, discover, and manage OCI artifacts and images.

## Technology Stack

- **Language**: Go 1.25+
- **CLI Framework**: Cobra (github.com/spf13/cobra)
- **Testing**: Ginkgo for e2e tests, standard Go testing for unit tests
- **Package**: oras.land/oras-go/v2 for core functionality

## Code Style and Standards

### General Go Coding

1. **Follow Go conventions**: Use standard Go idioms and patterns
2. **Error handling**: Always handle errors explicitly, never ignore them
3. **Context**: Use context.Context for cancellation and timeout handling
4. **Imports**: Group imports in three sections: standard library, external packages, internal packages

### Error Messages

Follow the [Error Handling and Message Guideline](../docs/proposals/error-handling-guideline.md):

1. **Structure**: `{Error|Error response from registry}: {Error description (HTTP status code if from server)}`
2. **Be descriptive**: Provide full description if user input doesn't match expectations
3. **Be actionable**: Include suggestions, command usage, or link to documentation
4. **Be user-friendly**:
   - Use capital letters at the start of error messages
   - Print human-readable messages, not programming expressions
   - When server errors vary, indicate the error is from the server
   - Guide users to use `--verbose` for detailed logs when needed
   - Avoid formula-like or programming expressions (e.g., no "json: cannot unmarshal...")
   - Avoid ambiguous phrases like "Something unexpected happens"

### CLI Design

1. **Flags**: Use consistent flag naming across commands
2. **Help text**: Provide clear examples in command help (see `cmd/oras/root/repo/ls.go` for examples)
3. **Output formats**: Support text, JSON, and go-template formats where applicable
4. **Compatibility**: Consider backward compatibility with different registry versions

### Testing

1. **E2E tests**: Use Ginkgo/Gomega framework (see `test/e2e/suite/command/`)
2. **Test structure**: Follow existing patterns with Describe/When/It blocks
3. **Test data**: Use test fixtures from `test/e2e/internal/testdata/`
4. **Test utilities**: Use helper functions from `test/e2e/internal/utils/`
5. **Assertions**: Use Gomega matchers (Expect/Should/To)

### Code Organization

1. **Commands**: Place in `cmd/oras/root/` or appropriate subdirectory
2. **Internal packages**: Use `internal/` for non-exported functionality
3. **Options**: Group related flags in option structs (see `cmd/oras/internal/option/`)
4. **Display**: Separate presentation logic in display handlers

## Building and Testing

### Build Commands

```bash
# Build for current platform
make build-<os>-<arch>

# Run unit tests
make test

# Run e2e tests
make teste2e

# Lint code
make lint

# Check encoding
make check-encoding
```

### Development Workflow

1. Run `make tidy vendor` to update dependencies
2. Run `make check-encoding` to ensure proper line endings
3. Run `make test` for unit tests
4. Run `make teste2e` for end-to-end tests
5. Run `make lint` to check code quality

## Feature Considerations

### Distribution Spec Compatibility

- Support OCI Distribution Spec v1.0 and v1.1
- Use `--distribution-spec` flag for registry compatibility modes
- Handle both Referrers API and referrers tag schema

### Multi-arch Support

- Handle multi-architecture images properly
- Support platform-specific operations
- Consider index manifests and per-platform manifests

### Debug and Verbose Output

- Provide meaningful output with `--verbose` flag
- Format debug logs with timestamps and clear request/response separation
- Include relevant metadata (digest, size, mediaType, annotations)
- Print user environment info when helpful for troubleshooting

## Documentation

- Update relevant documentation in `docs/` when adding features
- Follow the proposal template in `docs/proposals/proposal-doc-template.md`
- Include versioned permanent links when referencing external docs
- Provide examples in command help text

## Security

- Never commit secrets or credentials
- Handle authentication securely
- Validate user input to prevent injection attacks
- Follow security best practices for HTTP clients

## Common Patterns

### Command Structure

```go
type commandOptions struct {
    option.Common
    option.Remote
    // ... other option groups
    customField string
}

func commandCmd() *cobra.Command {
    var opts commandOptions
    cmd := &cobra.Command{
        Use:   "command [flags] <args>",
        Short: "Brief description",
        Long:  `Detailed description with examples`,
        Args:  oerrors.CheckArgs(argument.Exactly(1), "description"),
        PreRunE: func(cmd *cobra.Command, args []string) error {
            return option.Parse(cmd, &opts)
        },
        RunE: func(cmd *cobra.Command, args []string) error {
            return runCommand(cmd, &opts)
        },
    }
    // Add flags
    option.ApplyFlags(&opts, cmd.Flags())
    return oerrors.Command(cmd, &opts.Remote)
}
```

### Test Structure

```go
var _ = Describe("Feature description:", func() {
    When("running command", func() {
        It("should behavior", func() {
            // Setup
            // Execute
            // Assert
            Expect(result).To(Equal(expected))
        })
    })
})
```

## Best Practices

1. **Minimal changes**: Make surgical, focused changes to accomplish tasks
2. **Backward compatibility**: Don't break existing functionality
3. **Test coverage**: Add tests for new functionality
4. **Error clarity**: Ensure errors guide users to solutions
5. **Performance**: Consider efficiency, especially for large artifact operations
6. **Logging**: Use appropriate log levels (debug, verbose, standard output)
7. **User experience**: Prioritize ease of use and clear feedback
