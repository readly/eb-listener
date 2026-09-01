# AGENTS.md

Guidance for coding agents working in `github.com/readly/eb-listener`.
Keep changes focused, minimal, and consistent with existing code.

## 1) Project Context
- Language: Go (`go 1.22.6`)
- Type: CLI for listening to AWS EventBridge events
- Module path: `github.com/readly/eb-listener`
- Entrypoint: `cmd/eb-listener/main.go`
- Core packages:
  - `pkg/app` for CLI command setup
  - `pkg/eb` for EventBridge rule/target lifecycle
  - `pkg/listen` for SQS queue lifecycle and message polling

## 2) Extra Rule Files Check
Repository was checked for agent instruction overlays:
- `.cursor/rules/`: not present
- `.cursorrules`: not present
- `.github/copilot-instructions.md`: not present

If these files appear later, treat them as top-priority repository rules.

## 3) Environment Setup
- Preferred: `nix develop`
- Nix shell tools: `go`, `delve`, `goreleaser`
- Non-Nix setup: Go `1.22.x` + valid AWS credentials/profile

## 4) Build and Run Commands
- CLI help: `go run ./cmd/eb-listener --help`
- Build binary: `go build ./cmd/eb-listener`
- List buses: `AWS_REGION=us-west-1 go run ./cmd/eb-listener list`
- Listen on bus: `AWS_PROFILE=secret go run ./cmd/eb-listener listen --bus pinkbus`
- Release snapshot build: `goreleaser build --snapshot --clean`

## 5) Lint / Format / Static Analysis
- Format (required): `gofmt -w .`
- Optional formatter wrapper: `go fmt ./...`
- Static analysis: `go vet ./...`
- Note: no `golangci-lint` config currently exists in this repo

## 6) Test Commands (Including Single Test)
- Run all tests: `go test ./...`
- Run one package: `go test ./pkg/listen`
- Run one test: `go test ./pkg/listen -run '^TestName$' -v`
- Run one subtest: `go test ./pkg/listen -run '^TestName$/Subcase$' -v`
- Run single test with race detector: `go test -race ./pkg/listen -run '^TestName$'`
- Coverage check: `go test ./... -cover`

Current state note: repository currently has no `*_test.go` files.
When adding/changing behavior, add tests where practical.

## 7) Coding Style Guidelines

### Imports and Formatting
- Always keep code `gofmt`-clean
- Let Go tooling manage import ordering/grouping
- Keep standard library imports separate from external imports
- Remove unused imports immediately
- Avoid import aliases unless they meaningfully improve clarity

### Types and Data Modeling
- Prefer concrete types over `interface{}`
- Keep exported symbols minimal and intentional
- Model domain data with structs (`Bus`, `SQS`, `Event` style)
- Use `json.RawMessage` for dynamic JSON payload sections
- Keep struct tags explicit and accurate
- Favor typed constants for repeated literals where useful

### Naming Conventions
- Use `camelCase` for local vars/functions
- Use `PascalCase` for exported identifiers
- Keep acronym casing consistent (`SQS`, `CLI`, `ARN`, `ID`)
- Use short receiver names (`b *Bus`, `s *SQS`)
- Use meaningful boolean names (`verbose`, `found`, `ok`)
- Avoid vague names like `data`, `obj`, `tmp` unless truly temporary

### Error Handling
- Return errors rather than panic in normal execution paths
- Wrap upstream errors with context via `%w`
- Prefer messages like `fmt.Errorf("failed to create queue: %w", err)`
- Keep error strings lowercase and without trailing punctuation
- Never ignore errors silently
- If cleanup fails during error handling, log cleanup failure and return main error

### Context Usage
- Accept `context.Context` as first argument for I/O and long-running operations
- Prefer caller-provided contexts in new code
- Avoid introducing fresh `context.TODO()` unless explicitly justified
- Use cancellation (`context.WithCancel`) for goroutine lifecycle control

### Concurrency and Channels
- Define send/close ownership clearly
- Close channels only from producer side
- Ensure goroutines have deterministic shutdown paths
- Coordinate shutdown with `sync.WaitGroup`
- Avoid goroutine leaks and blocked sends

### Logging
- Use structured logging with `log/slog`
- Use stable, searchable keys (for example `id`, `bus`, `url`, `signal`, `error`)
- `Debug` for flow detail, `Info` for milestones, `Error` for failures
- Include enough context to diagnose AWS/API failures quickly

### AWS and Side-Effect Safety
- Keep generated names deterministic using run IDs
- Preserve cleanup of temporary EventBridge rule/target and SQS queue
- Keep IAM scope as narrow as practical
- Do not broaden permissions without clear necessity

## 8) Change Workflow for Agents
Before finishing a task:
1. Run `gofmt -w .`
2. Run `go vet ./...`
3. Run `go test ./...` (or the narrowest relevant package/test scope)
4. Update docs if CLI flags, behavior, or usage changed

When implementing features/fixes:
- Prefer incremental refactors over broad rewrites
- Keep CLI behavior backward compatible unless explicitly asked to change it
- Add/update tests for changed behavior when practical
- Keep unrelated edits out of the same change

## 9) Commit Guidance (If Asked to Commit)
- Use concise, imperative commit messages
- Focus commit message on intent/why, not only file-level what
- Keep commits logically scoped and easy to review

## 10) Helpful Go Tooling Shortcuts
- Inspect package/type/function docs: `go doc pkg/path.Symbol`
- Inspect all docs in package: `go doc -all pkg/path`
- Explore dependency source:
  1. `go mod download -json MODULE`
  2. Read source under the returned `Dir`
- Prefer `go run ./cmd/eb-listener` for execution-oriented checks to avoid extra artifacts

## 11) Operational and Safety Notes
- This tool creates temporary AWS resources; always preserve cleanup paths.
- Avoid broadening AWS permissions without explicit need.
- Do not introduce destructive defaults in CLI behavior.
- Handle signal-driven shutdown paths carefully (`os.Interrupt`, `SIGTERM`, `SIGHUP`).
- Keep logs actionable for operations and incident debugging.
