# CONSTITUTION

The constitution establishes the rules, principles, and patterns that govern `kindlebeam` development. All code changes, decisions, and architecture choices must align with these guidelines.

---

## PRINCIPLES

### 1. Explicit Error Handling and Fail-Fast

- All errors must be handled explicitly; silent failures are unacceptable.
- Validation happens at function boundaries; the sooner invalid input is rejected, the better.
- Error messages must be specific, actionable, and include sufficient context for debugging.
- Use `fmt.Errorf(...%w...)` to wrap errors with caller context.

### 2. Single Responsibility

- Each module, function, and type must have exactly one reason to change.
- Layered architecture enforces separation: CLI layer → service layer → infrastructure/repository layer.
- Avoid mixing concerns (e.g., don't parse CLI arguments inside a business logic function).

### 3. Security by Default

- Block dangerous operations at the boundary (e.g., validate Pandoc arguments, block code-execution flags).
- Sanitize all paths to prevent traversal attacks; use `filepath.Clean`, `filepath.Abs`, and symlink resolution.
- Never log sensitive data (email addresses, file contents); log only metadata and structured information.
- All external process invocations (Pandoc, mail) must use `exec.CommandContext` with validated arguments.

### 4. Minimal Dependencies

- Prefer the Go standard library for all tasks unless a clear gap exists.
- When external dependencies are necessary, choose stable, well-maintained libraries with small footprints.
- Current permitted dependencies: `spf13/cobra` (CLI framework), `mattn/go-shellwords` (argument parsing).
- New dependencies require architectural justification and must be documented here.

### 5. Readability Over Cleverness

- Optimize for code clarity, maintainability, and understandability by future maintainers.
- Use descriptive names for variables, functions, struct fields, and types.
- Prefer explicit loops and clear control flow over clever, compact idioms.
- Comments explain the _why_, not the _what_; code should be self-explanatory for the _what_.

### 6. Configuration Over Hardcoding

- All user-configurable values live in the config file (`~/Library/Application Support/kindlebeam/config.json` on macOS).
- CLI flags override config values; config provides defaults; hard-coded fallbacks are allowed only for universal defaults (e.g., `"markdown"` as fallback format).
- Sensible defaults reduce friction; users should not need to configure anything to start using the tool.

---

## CONSTRAINTS

### Invariant Rules (Must Never Be Violated)

1. **Security Gates**
   - Pandoc arguments that enable code execution or filesystem access MUST be blocked.
   - Blocked flags: `--lua-filter`, `--filter`, `-F`, `--extract-media`, `--include-in-header`, `-H`, `--include-before`, `-B`, `--include-after`, `-A`, `--reference-doc`, `--data-dir`, `--syntax-definition`, `--abbreviations`, `--log`.
   - Path traversal MUST be prevented via `filepath.EvalSymlinks` and `filepath.Abs` validation.

2. **Separation of Layers**
   - CLI commands (`cmd/kindlebeam/*.go`) MUST NOT contain business logic; they delegate to the service/workflow layer.
   - Service layer (`internal/app/workflow.go`) MUST NOT contain CLI concerns; it accepts context, well-formed options, and returns clear results.
   - Infrastructure clients (`internal/pandoc`, `internal/mailer`) MUST only wrap external processes; no business logic allowed.
   - Config operations MUST only happen in the config package; no config loading scattered throughout other packages.

3. **Error Propagation**
   - All errors MUST propagate up the call stack with context; never silently ignore errors.
   - At the CLI layer, errors MUST be printed to stderr with the `❌` prefix before exiting.

4. **Testing Obligation**
   - Any new business logic MUST have unit tests (see testing section).
   - Tests MUST NOT depend on external tools (Pandoc, mail) being installed; use mocks or environment isolation.
   - Integration tests are encouraged for workflows but MUST use temporary directories and mocked external tools.

5. **Documentation Requirement**
   - Public functions and types (exported, capitalized) MUST have Go docstrings (comments immediately before the declaration).
   - Docstrings MUST explain the function's purpose, parameters, return values, and any error conditions.
   - Non-exported helpers MAY omit docstrings if the code is self-documenting; use judgment.

6. **Version Alignment**
   - Go version MUST be 1.22 or later (as specified in `go.mod`).
   - Dependency versions MUST be regularly reviewed; `go mod tidy` before commits.

---

## ARCHITECTURAL DESIGN

### Layered Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  CLI Layer (cmd/kindlebeam/)                                │
│  - Command definitions (root, convert, send, config)        │
│  - Flag parsing                                              │
│  - Argument validation                                       │
│  - Error display                                             │
└────────────────────┬────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────┐
│  Service / Business Logic Layer (internal/app/)              │
│  - Workflow orchestration (convert, send, convert+send)     │
│  - Format resolution and inference                           │
│  - Path validation and output naming                         │
│  - Logger (structured logging with emoji prefixes)          │
└────────────────────┬────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────┐
│  Infrastructure / Repository Layer                          │
│  - Config: load/save JSON, path resolution, defaults        │
│  - Pandoc: wrapper around pandoc binary, process mgt        │
│  - Mailer: wrapper around mail binary, MIME handling        │
└─────────────────────────────────────────────────────────────┘
```

### Request/Result Pattern

- Command workflows accept structured **Options** objects (e.g., `WorkflowOptions`, `ConvertOptions`, `SendOptions`).
- Example: `func (w *Workflow) ConvertAndSend(ctx context.Context, files []string, opts WorkflowOptions) error`
- External process clients return **Result** objects that include command details and stderr for debugging.
- Example: `type ConvertResult struct { Command []string; Stderr string }`

### Dependency Injection

- Services receive their dependencies via constructor functions (e.g., `NewWorkflow(cfg, log, ...)`).
- No global state; all services are stateless or manage explicit state passed via function arguments.
- Example: `func NewWorkflow(cfg config.Config, log *Logger) (*Workflow, error)`

---

## CODE STYLE AND CONVENTIONS

### Naming Conventions

- **Packages**: lowercase, short, no underscores (e.g., `pandoc`, `mailer`, `config`).
- **Functions**: verb-noun pattern (e.g., `validateInputFile`, `buildOutputPath`, `inferInputFormat`).
- **Variables**: short names in small scopes, descriptive in larger scopes (e.g., `cfg` for config in loop, `configuration` for module-level).
- **Constants**: UPPER_CASE with underscores (e.g., `defaultOutputDir`).
- **Interfaces**: ended with "er" or descriptive of behavior (future use when needed).

### Comments and Documentation

- Function docstrings: explain purpose, parameters, returns, and error conditions.
- Inline comments: explain _why_ non-obvious code exists, not _what_ it does.
- Use lowercase at the start of comments unless referring to identifiers.
- Actionable comments: prefix with `TODO:`, `FIXME:`, or `NOTE:` with context.
- Example:

  ```go
  // validatePandocArgs checks that no dangerous arguments are present.
  // Dangerous flags can execute arbitrary code or access sensitive filesystem areas.
  func validatePandocArgs(args []string) error { ... }
  ```

### Code Structure

- Keep functions focused and short; extract helper functions generously.
- Avoid deep nesting (max 3–4 levels); use early returns and guard clauses.
- Group related constants and declarations together.
- Internal helper functions (unexported) are acceptable for clarity; don't overuse them, but don't avoid them either.

### Error Message Style

- First letter lowercase (e.g., `"invalid format %q"` not `"Invalid format %q"`).
- Include the context: what was attempted, what went wrong, how to fix it.
- Don't repeat the error prefix; let the logger (❌) handle that.
- Examples:
  - `"pandoc not found (%s): %w"` ✅
  - `"create output dir: %w"` ✅
  - `"cannot access %q: %w"` ✅

### Logging

- Use the `Logger` type in `internal/app/logger.go` for all logging.
- Emoji prefixes: `ℹ️` (info), `✅` (success), `❌` (error), `🔍` (debug, `--verbose` only).
- Log to stderr; stdout is reserved for actual output.
- Log at appropriate levels:
  - `Infof`: user-facing status ("converting...", "sending...").
  - `Successf`: operation completed ("sent file X").
  - `Errorf`: errors that the user should be aware of.
  - `Debugf`: implementation details, only in `--verbose` mode.

---

## SECURITY PRINCIPLES

### Pandoc Argument Validation

1. Block all code-execution and filesystem-access flags (see Constraints section).
2. Validate arguments before passing them to `exec.CommandContext`.
3. Use the `validatePandocArgs([]string) error` function in `internal/app/workflow.go`.
4. Rationale: Pandoc filters can execute arbitrary code; blocking them prevents command injection.

### Path Sanitization

1. Use `filepath.Clean()` to resolve `.` and `..`.
2. Use `filepath.Abs()` to convert to absolute path.
3. Use `filepath.EvalSymlinks()` to resolve symlinks (for input files, not output).
4. Check file existence and type (not a directory) for input files.
5. Use helper functions: `validateFilePath()` and `validateInputFile()` in `internal/app/workflow.go`.

### Sensitive Data Handling

- Never log full file contents.
- Log file paths, email addresses only when essential for debugging.
- Recommend `--verbose` flag for diagnostic output rather than enabling by default.
- Config file should be readable only by the owner (mode `0o644` for the file, `0o755` for directories).

---

## DEPENDENCIES AND THEIR PURPOSES

### External Dependencies

| Package                          | Version   | Purpose                               | Justification                                                          |
| -------------------------------- | --------- | ------------------------------------- | ---------------------------------------------------------------------- |
| `github.com/spf13/cobra`         | `v1.8.1`  | CLI framework with subcommand support | Eliminates boilerplate; well-tested and stable.                        |
| `github.com/mattn/go-shellwords` | `v1.0.12` | Parse shell-like argument strings     | Handles parameterized `--pandoc-args` safely without invoking a shell. |

### Standard Library (Required)

- `os`, `os/exec` – file I/O, process management.
- `context` – context propagation and cancellation.
- `fmt` – string formatting and error creation.
- `json` – config file parsing and generation.
- `path/filepath` – path manipulation and validation.
- `strings` – string utilities.
- `bytes` – buffer management for process I/O.
- `encoding/base64` – MIME attachment encoding in `sendWithSendmail`.
- `io` (implicit via others) – I/O abstraction.

---

## DEVELOPMENT WORKFLOW

### Change Classification

All work falls into one of two tracks:

#### 1. Spec-Driven (Formal)

**Use for:**

- New features or substantial behavioral changes.
- Architectural overhauls or refactoring of major components.
- Changes affecting the public CLI interface.
- Complex changes requiring detailed planning.

**Workflow:**

1. Create a feature directory under `docs/specs/<feature-name>/`.
2. Write `SPEC.md` (requirements, acceptance criteria).
3. Write `PLAN.md` (implementation strategy, architectural decisions).
4. Write `TASKS.md` (ordered, atomic tasks).
5. Implement tasks in order.
6. Verify against acceptance criteria.
7. Update spec docs if reality diverges (spec is authority).

**Artifacts:**

- `docs/specs/<feature>/SPEC.md` – requirements and acceptance criteria.
- `docs/specs/<feature>/PLAN.md` – implementation strategy.
- `docs/specs/<feature>/TASKS.md` – executable task list.
- Optional: `ANALYSIS.md`, `research.md`, `implementation-details/`.

#### 2. Ad Hoc (Lightweight)

**Use for:**

- Bug fixes and isolated defects.
- Security reviews and minor security improvements.
- Dependency updates and version bumps.
- Config file adjustments.
- Small refinements and polish.
- Refactoring internal implementations (not architecture).

**Workflow:**

1. Understand the problem thoroughly.
2. Implement the minimal fix.
3. Verify against the reported issue.
4. Update practical docs only (README, inline docs, API docs).
5. **Do NOT create spec artifacts** for ad hoc work.

**Exception: Ad Hoc with Existing Specs**

- If ad hoc work touches code with existing `docs/specs/<feature>/` documents, update them if behavior or approach changes.
- Skip spec updates only for purely mechanical changes (formatting, typos, dependency version bumps).

---

## TESTING STRATEGY

### Unit Testing

**What to test:**

- Format inference logic (`inferInputFormat`, `buildOutputPath`).
- Config loading, merging, and default application.
- Argument validation (Pandoc args, file paths).
- Effective value resolution (CLI override → config → default).

**Test patterns:**

```go
func TestInferInputFormat(t *testing.T) {
    cases := map[string]string{
        "file.md":  "markdown",
        "file.org": "org",
        // ...
    }
    for name, want := range cases {
        got := inferInputFormat(name)
        if got != want {
            t.Errorf("got %q, want %q", got, want)
        }
    }
}
```

**Test requirements:**

- Use the standard `testing` package.
- Test function names follow `TestFunctionName` pattern.
- Use subtests for related cases: `t.Run("case name", func(t *testing.T) { ... })`.
- Do not depend on external tools or the internet.
- Use `t.TempDir()` for file I/O tests.

### Integration Testing

**What to test:**

- `ConvertOnly`, `SendOnly`, `ConvertAndSend` workflows.
- Correct behavior of `--dry-run`, `--no-send`, `--no-clean` flags.
- End-to-end CLI invocation with mocked Pandoc and mail.

**Testing approach:**

- Create mock binaries (shell scripts) in temp directories.
- Inject mocks into `PATH` or use config file overrides.
- Record invoked commands to files for assertion.
- Test that real workflows call mocks correctly and handle results.

### Running Tests

```sh
make test          # Run all tests
go test ./...      # Equivalent
go test -v ./...   # Verbose
go test -run TestFoo ./... # Run specific test
```

---

## BUILD AND DEPLOYMENT

### Build Process

```sh
make build   # Compile binary to ./bin/kindlebeam/kindlebeam
make run     # Run binary without rebuilding
make test    # Run all tests
make fmt     # Format code with gofmt
make lint    # Run linter (if golangci-lint installed)
```

**Output location:** `./bin/kindlebeam/kindlebeam` (executable).

### Distribution

- Binary is standalone; no runtime dependencies on Go.
- Required external tools: `pandoc`, `mail` (BSD/GNU mail command).
- Installation: copy binary to a directory on user's `PATH` (e.g., `/usr/local/bin`).

---

## GOALS AND NON-GOALS

### Goals

1. **Simple, reliable document conversion:** Convert supported formats via Pandoc with minimal configuration.
2. **Seamless Kindle delivery:** Email converted files to Kindle with one command.
3. **Sensible defaults:** Users should not need to configure anything for common workflows (Markdown → PDF → Kindle).
4. **Clear error messages:** When something goes wrong, the user understands what happened and how to fix it.
5. **Security by default:** Block common attack vectors (code injection, path traversal) at the boundary.
6. **Minimal friction:** Few dependencies, small binary, no network calls, works offline.

### Non-Goals (Explicitly Out of Scope)

1. **Document merging:** Combining multiple input files into a single output file.
2. **Rich email templates:** HTML emails, custom styling, embedded images (beyond attachments).
3. **Online storage integration:** Google Drive, Dropbox, S3 backends (this is local-only).
4. **Advanced metadata injection:** Embedding titles, authors, or table-of-contents metadata into output.
5. **Watch mode or file system monitoring:** Auto-convert on file changes.
6. **Batch scheduling or cron integration:** Task scheduling or repeated conversions.
7. **GUI or TUI:** Remains a CLI tool.
8. **Format plugins or extensibility:** Only built-in Pandoc formats are supported.

### Future Direction

- Multi-file attachment per email (low priority).
- Alternative email backends (SendGrid, SES, SMTP) instead of system `mail`.
- More granular logging and metrics.
- Dry-run mode improvements (estimate file sizes, preview PDF structure).

---

## DEFINITIONS

### Key Terms

- **Workflow:** A sequence of operations (convert, send, or both) applied to input files.
- **Format inference:** Determining Pandoc input format from file extension or falling back to config defaults.
- **Output path:** The directory and filename where a converted file is written.
- **Effective value:** The value used after merging CLI flag → config file → hardcoded default (in that priority order).
- **Dry-run:** Preview mode; compute operations and log commands without executing Pandoc or mail.
- **Sanitization:** Validating and cleaning input (paths, arguments) to prevent injection or traversal attacks.
- **Client:** A thin wrapper around an external tool (Pandoc, mail) with a request/result interface.

---

## CHANGE REVIEW CHECKLIST

Before committing any change, verify:

- [ ] **Security:** No dangerous Pandoc flags allowed; paths sanitized; no sensitive data logged.
- [ ] **Errors:** All errors handled explicitly with context; no silent failures.
- [ ] **Separation:** No business logic in CLI commands; no CLI concerns in service layer.
- [ ] **Tests:** New logic has unit tests; no external tool dependencies in tests.
- [ ] **Documentation:** Public functions have docstrings; comments explain _why_, not _what_.
- [ ] **Naming:** Functions and variables use clear, descriptive names.
- [ ] **Dependencies:** Any new external dependency is justified and documented here.
- [ ] **Spec Alignment:** If spec-driven, verify implementation matches SPEC.md/PLAN.md.

---

## REFERENCES

- **Spec-driven development:** See `docs/specs/<feature>/SPEC.md`, `PLAN.md`, `TASKS.md`.
- **CLI interface:** See `README.md` for user documentation.
- **Implementation details:** See inline comments and function docstrings in source code.
- **Project roadmap:** See Non-Goals section and consider feature priorities before starting spec-driven work.
