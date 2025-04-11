# Implementation Plan: Go Version Upgrade & Unit Testing

## Completed Tasks

### Dependency Modernization
- Migrated from dep/Gopkg to Go modules
  - Created go.mod with proper module path
  - Added replace directive for `github.com/imdario/mergo => dario.cat/mergo`
  - Removed Gopkg.toml and Gopkg.lock
- Updated dependencies to compatible versions
- Updated Makefile with Go modules support

### Code Quality & Testing
- Added basic unit tests with proper mocking
- Fixed all linting issues:
  - Error handling: Wrapped all errors from external packages with proper context
  - Code style: Added package comments, function documentation, and constants
  - Clean code: Removed unused variables, renamed unused parameters
  - Formatting: Fixed long lines and improved readability

### Build Infrastructure
- Enhanced bootstrap process for dependency management
- Added lint targets to Makefile for quality checks
- Added test target for running unit tests

## Remaining Tasks

### Code Migration
- Update any deprecated Go language features
- Update code to use newer dependency APIs
- Fix any breaking changes in the dependencies

### Testing
- Add tests for fetchReleases and fetchIndices functions
- Add tests for other formatting outputs (plain, yaml, table)
- Implement integration tests if needed
- Add GitHub Actions or other CI/CD workflow for automated testing

### Documentation & Finalization
- Update version information in plugin.yaml
- Create release notes for the upgrade

### Release
- Create a new release with updated binary artifacts
- Test the plugin with the latest Helm version
- Update the plugin in the Helm plugin repository (if applicable)

## Linting Implementation Plan

### Priority 1: Fix Error Handling Issues ✅
1. **wrapcheck (7 issues)** - Properly wrap errors returned from external packages:
   - [x] Fix error handling in `main.go:137` - wrap `tlsutil.ClientConfig` error
   - [x] Fix error handling in `main.go:189` - wrap `IndexFile.Get` error
   - [x] Fix error handling in `main.go:228` - wrap `json.MarshalIndent` error
   - [x] Fix error handling in `main.go:236` - wrap `yaml.Marshal` error
   - [x] Fix error handling in `main.go:257` - wrap `Client.ListReleases` error
   - [x] Fix error handling in test mocks (main_test.go)

2. **errorlint (2 issues)** - Fix error format strings:
   - [x] Update `main.go:270` to use `%w` format verb for error wrapping
   - [x] Update `main.go:275` to use `%w` format verb for error wrapping

3. **errcheck (4 issues)** - Check error return values:
   - [x] Handle errors in mock methods in `main_test.go`
   - [x] Check error from `os.Pipe()` in tests
   - [x] Check error from `w.Close()` in tests

### Priority 2: Clean Code Issues ✅
1. **unused (4 issues)** - Remove unused code:
   - [x] Remove or use `globalUsage` constant
   - [x] Remove or use `settings` variable
   - [x] Remove unused test helper functions

2. **revive (4 issues)** - Improve code quality:
   - [x] Add package comment to main.go
   - [x] Add comment for exported type `ChartVersionInfo`
   - [x] Handle unused parameters in functions (replaced with `_`)

3. **gocritic (2 issues)** - Follow Go best practices:
   - [x] Invert if condition to simplify code structure in `main.go:180`
   - [x] Replace empty fallthrough with expression list in `main.go:232`

### Priority 3: Minor Improvements ✅
1. **goconst (1 issue)** - Create constant for repeated string literals:
   - [x] Make repeated string "plain" a constant

2. **lll (1 issue)** - Fix long line:
   - [x] Break long formatted line at `main.go:213`

### Makefile Target ✅
- [x] Add `make lint` and `make lint-specific` targets to run linters
