# AGENTS.md - Helmper Coding Guidelines

## Build Commands

```bash
# Build the application
cd cmd/helmper && go build

# Run all tests
go test -v ./...

# Run a specific test
go test -v ./pkg/helm -run TestPull
go test -v ./internal -run TestProgram

# Run tests with coverage
go test -cover ./...

# Lint (requires golangci-lint)
golangci-lint run

# Clean and tidy dependencies
go mod tidy
go mod verify
```

## Code Style Guidelines

### Project Structure
- Follow [golang-standards/project-layout](https://github.com/golang-standards/project-layout)
- `cmd/`: Main application entry points
- `pkg/`: Public library code (helm, image, registry, etc.)
- `internal/`: Private application code
- `docs/`: Documentation
- `example/`: Example configurations

### Imports
- Group imports: stdlib, third-party, local project
- Use blank line between groups
- Local imports use full module path: `github.com/ChristofferNissen/helmper/pkg/helm`
- Alias imports when necessary (e.g., `helm_registry "helm.sh/helm/v3/pkg/registry"`)

### Formatting
- Use `gofmt` / `goimports` for formatting
- Line length: keep under 120 characters when possible
- Use trailing commas in multi-line struct literals

### Types and Naming
- Use PascalCase for exported types/functions
- Use camelCase for unexported types/functions
- Use ALL_CAPS for constants
- Interface names: `-er` suffix (e.g., `Reader`, `Writer`)
- Constructor functions: `New` prefix (e.g., `NewClient`)

### Error Handling
- Always check errors explicitly
- Wrap errors with context: `fmt.Errorf("context: %w", err)`
- Use `%w` verb for error wrapping (not `%v`)
- Return errors early to reduce nesting
- Use `log/slog` for logging errors, not `fmt.Printf`

### Logging (slog)
- Use structured logging with `log/slog`
- Use key-value pairs OR attributes, never mix
- Use context-aware logging methods
- Use static message strings
- Use snake_case for keys
- Example: `slog.Info("message", slog.String("key", value))`

### Testing
- Use `testify` package (assert, mock)
- Use table-driven tests
- Test function naming: `Test<FunctionName>`
- Use `t.Run()` for subtests with descriptive names
- Create temporary directories with `os.MkdirTemp()`
- Clean up resources with `defer`

### Concurrency
- Use `context.Context` for cancellation
- Use channels for communication between goroutines
- Use `sync.WaitGroup` for coordinating goroutines
- Use `go.uber.org/fx` for dependency injection

### Configuration
- Use `spf13/viper` for configuration management
- Use `spf13/afero` for filesystem abstraction in tests

## Git Conventions

### Commit Messages
- Be descriptive and explain "why" not "what"
- Use present tense: "Add feature" not "Added feature"
- Reference issues when applicable

### Branch Naming
- Format: `feat/ISSUE_NUMBER`
- Example: `feat/123`

## Linting Configuration

The project uses `golangci-lint` with these linters enabled:
- `errcheck`: Check unchecked errors
- `goimports`: Format imports
- `revive`: General linting
- `govet`: Vet analysis
- `staticcheck`: Static analysis
- `sloglint`: Enforce slog best practices

## Dependencies

- Go version: 1.22+
- Key dependencies:
  - `helm.sh/helm/v3`: Helm SDK
  - `go.uber.org/fx`: Dependency injection
  - `github.com/spf13/viper`: Configuration
  - `github.com/stretchr/testify`: Testing
  - `github.com/sigstore/cosign/v2`: Signing
  - `oras.land/oras-go/v2`: OCI registry

## Security

- Run Trivy scan before submitting: `trivy fs --exit-code 1 --severity HIGH,CRITICAL .`
- Never commit secrets or credentials
- Use environment variables for sensitive configuration
