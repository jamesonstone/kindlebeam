# kindlebeam CLI Implementation Plan

## Problem statement

The `kindlebeam` CLI should take one or more source documents, convert them via Pandoc into a Kindle-friendly format (default: Markdown → PDF), and optionally email the resulting files to a configured Kindle address using the system `mail` command on macOS. The tool must provide a simple default workflow (`kindlebeam <files>` → convert+send) plus explicit `convert`, `send`, and `config` subcommands, with sensible defaults and clear error handling.

## Current state

* Repository `github.com/jamesonstone/kindlebeam` currently contains only `README.md` and a Go-oriented `.gitignore` with no Go module, source code, or tests.
* Target environment is macOS with `zsh`; external runtime dependencies will be:
  * `pandoc` available on `PATH` (or at a configured path).
  * BSD `mail` (macOS `mail`) available on `PATH` (or at a configured command name).
* No existing configuration files, CLI framework, or test harness are present yet.

## High-level design

* **Language and tooling**
  * Implement the CLI in Go, using a standard Go module (`go.mod`) rooted at `github.com/jamesonstone/kindlebeam`.
  * Use a structured CLI library (e.g., `spf13/cobra`) to model the root command and subcommands (`convert`, `send`, `config`) and global flags (`--verbose`, `--dry-run`).
* **Layered architecture (adapted for CLI)**
  * **CLI layer (presentation/controller)**: command definitions, flag parsing, wiring of subcommands, basic argument validation.
  * **Service layer (business logic)**:
    * Conversion service: given input files and format options, orchestrates Pandoc invocations and output path management.
    * Send service: given output files and email metadata, orchestrates `mail` invocations.
    * Workflow service: “convert+send” orchestration used by the default/root command.
  * **Repository / infrastructure layer**:
    * Config repository: load/save JSON config, resolve config file path, merge CLI overrides with stored defaults.
    * External command adapters: thin wrappers for `pandoc` and `mail` with well-defined request/response objects and error types.

## CLI interface and behavior

### Command structure

* **Root command `kindlebeam`**
  * Usage: `kindlebeam [global flags] [flags] <input-file>...`
  * Behavior:
    * If a subcommand is specified (`convert`, `send`, `config`), delegate accordingly.
    * Otherwise, treat remaining arguments as input files and perform **convert + send**:
    * Infer input format per file (extension → Pandoc format) unless `--input-format` is provided.
    * Determine output format (CLI `--output-format` → config `default_output_format` → `pdf`).
    * Convert each file via Pandoc.
    * Unless `--no-send` is set, send each converted file to Kindle using the configured Kindle email.
* **`kindlebeam convert`**
  * Usage: `kindlebeam convert [global flags] [flags] <input-file>...`
  * Flags:
    * `--to, -t <format>` (output format; alias of `--output-format`).
    * `--from, -f <format>` (input format; alias of `--input-format`).
    * `--output-dir <path>` (override default output directory).
    * `--pandoc-args "<args>"` (extra arguments passed through to Pandoc; see below).
  * Behavior: convert input files only; no email sending.
* **`kindlebeam send`**
  * Usage: `kindlebeam send [global flags] [flags] <file>...`
  * Flags:
    * `--kindle-email <email>` (override config).
    * `--subject <subject>`.
    * `--body <text>` (default body if omitted).
  * Behavior: send one or more existing files as attachments via `mail`.
* **`kindlebeam config`**
  * Usage: `kindlebeam config <subcommand> [...]`.
  * Subcommands:
    * `show` – print effective configuration (including resolved config path and defaults).
    * `set kindle-email <email>` – update Kindle email in config.
    * `set default-output-format <format>` – update default output format.
    * `set mail-command <cmd>` – update mail command name/path.
    * (Optionally) `set pandoc-path <path>` – set explicit Pandoc location.

### Global and shared flags

* **Global** (root persistent flags, inherited by subcommands):
  * `--version` – print version and exit.
  * `--verbose` – enable verbose logging (print built Pandoc and mail commands, stderr, and key steps).
  * `--dry-run` – resolve config, compute formats, and build commands but **do not** execute Pandoc or mail; instead, print the planned actions.
* **Root/convert/send flags** (where applicable):
  * `--input-format <format>` – explicit Pandoc input format; defaults to autodetect → config `default_input_format` → `markdown`.
  * `--output-format <format>` – explicit Pandoc output format; defaults to config `default_output_format` → `pdf`.
  * `--output-dir <path>` – directory for output files; defaults to directory of input file or a configurable default (see open questions).
  * `--kindle-email <email>` – override configured address when sending.
  * `--subject <subject>` – email subject; default derived from input filename or a configurable template.
  * `--no-send` – root command only; perform conversions but skip sending.
  * `--no-clean` – when sending, keep converted files on disk instead of cleaning them after a successful send.
  * `--pandoc-args "<args>"` – additional Pandoc arguments; parse safely into an argument slice (e.g., using a shellwords-style parser) before appending.

## Configuration design

* **Config file schema** (Go struct aligned with the provided JSON example):
  * `kindle_email: string` – primary Kindle address for sending.
  * `default_kindle_email: string` – alias or legacy field; treated as a fallback if `kindle_email` is empty.
  * `default_output_format: string` – default Pandoc `-t`/output format (default `"pdf"`).
  * `default_input_format: string` – default Pandoc `-f`/input format when autodetect fails (default `"markdown"`).
  * `mail_command: string` – command name or path for sending email (default `"mail"`).
  * `pandoc_path: string` – command name or path for Pandoc (default `"pandoc"`).
* **Config file location resolution**
  * Primary: `os.UserConfigDir()` + `/kindlebeam/config.json` (covers `~/.config/kindlebeam` on Linux and `~/Library/Application Support/kindlebeam` on macOS automatically).
  * If the primary path cannot be determined or does not exist yet, create the directory and file on first write.
  * Optionally support an explicit override via an environment variable (e.g., `KINDLEBEAM_CONFIG`) for advanced users and tests.
* **Load/merge strategy**
  * On every CLI invocation:
    * Resolve config path.
    * If file exists, load JSON into a `Config` struct and validate fields (e.g., non-empty email when sending).
    * Merge CLI flags over config values (flags take precedence).
    * Apply hard-coded defaults for any remaining unset fields.
  * `config set` subcommands:
    * Load existing config (if any), update the relevant field, then write back JSON with stable formatting.

## Conversion and send workflow

### Format inference and output naming

* **Input format inference**
  * If `--input-format`/`--from` is provided, use it directly.
  * Otherwise, infer from file extension via a small mapping (ext → Pandoc format): e.g., `.md` → `markdown`, `.org` → `org`, `.rst` → `rst`, `.tex` → `latex`, `.html` → `html`, `.docx` → `docx`, `.epub` → `epub`.
  * If extension is unknown, fall back to `default_input_format` from config, then `markdown`.
* **Output format resolution**
  * If `--output-format`/`--to` is provided, use it.
  * Otherwise, use `default_output_format` from config, then `pdf`.
* **Output filename construction**
  * For each input file:
    * Base name: strip extension from input file.
    * Output extension: based on resolved output format (e.g., `pdf` → `.pdf`, `epub` → `.epub`).
    * Output directory: `--output-dir` if provided; else, `./kindbeam_out` relative to the current working directory for root and `convert` commands. The directory is created on demand if it does not exist. The `send` command never deletes or relocates user-provided files.

### Pandoc invocation

* Encapsulate Pandoc calls in a `PandocClient` in the infrastructure layer:
  * Build `exec.CommandContext` with:
    * Binary: `config.pandoc_path` if set, else `"pandoc"` resolved via `exec.LookPath` at startup.
    * Args: `-f <input_format> -t <output_format> -o <output_file> <input_file> ...extra_args`.
  * When `--pandoc-args` is provided, parse the string into extra args and append after the standard args.
  * Capture stdout/stderr and exit code.
  * On non-zero exit code, surface a structured error that includes stderr (and print it under `--verbose`).

### Mail invocation

* Encapsulate `mail` calls in a `MailClient` in the infrastructure layer:
  * Command: `config.mail_command` (default `"mail"`), validated via `exec.LookPath` at startup.
  * Build arguments based on BSD `mail` with attachment support, e.g.:
    * `-s <subject>` for subject.
    * `-a <file>` for each attachment (or equivalent appropriate for the detected `mail` variant).
  * Provide body text via stdin (from `--body` or a default message).
  * On non-zero exit code, surface a structured error containing stderr.
* By default, send **one email per output file** (simple, predictable behavior); consider multi-attachment support as a future enhancement if desired.

### Root workflow (convert + send)

* For each input file:
  * Validate the file exists and is readable.
  * Resolve input/output formats and output path.
  * If `--dry-run`: log the planned Pandoc and mail commands, then stop (no actual execution).
  * Otherwise:
    * Call `PandocClient.Convert`.
    * If `--no-send` is not set, call `MailClient.Send` with the Kindle email and computed subject/body.
    * If sending succeeds and `--no-clean` is not set, remove the generated output file.
* Fail-fast on clearly invalid configuration (missing Kindle email when sending, missing Pandoc/mail binaries) with descriptive messages.

## Error handling and logging

* **Error handling**
  * Validate required inputs early (files, formats, Kindle email before send).
  * Distinguish between:
    * Configuration errors (e.g., missing Kindle email, invalid format names).
    * Environment errors (missing `pandoc` or `mail`, permission problems).
    * Runtime command failures (Pandoc/mail exit codes, I/O issues).
  * Exit with non-zero codes and concise error messages; under `--verbose`, include underlying stderr from Pandoc/mail.
* **Logging**
  * Default mode: minimal, user-friendly messages for major steps only (e.g., “converting X → Y”, “sending Y to Z”).
  * `--verbose` mode:
    * Log full Pandoc and mail command lines (with paths and args), working directories, and timing if inexpensive to measure.
    * Prefix log lines with emojis to improve readability (e.g., info `ℹ️`, success `✅`, error `❌`, debug `🔍`) while still writing to stderr for logs.
  * Avoid logging full file contents or sensitive data (email addresses only where necessary).

## Testing strategy

* **Unit tests**
  * Config repository:
    * Path resolution using `os.UserConfigDir` and optional env override.
    * JSON load/save, including defaulting and backward-compatible handling of `default_kindle_email`.
  * Format inference:
    * Extension → input format mapping with override and fallback behavior.
    * Output filename and directory construction.
  * CLI parsing:
    * Global flag handling, per-command flags, and argument validation.
    * Root command behavior when invoked with and without explicit subcommands.
  * Command building:
    * Pandoc argument lists given different formats and `--pandoc-args` values.
    * Mail argument lists given various subjects, bodies, and file lists.
* **Integration tests**
  * Use temporary directories and PATH overrides to inject mock `pandoc` and `mail` shell scripts that:
    * Record received arguments to a file.
    * Optionally simulate success or failure via exit codes.
  * End-to-end scenarios:
    * `kindlebeam convert` producing expected outputs with mocks.
    * `kindlebeam` (default) performing convert+send with mocks, respecting `--no-send`, `--no-clean`, and `--dry-run`.
    * `kindlebeam send` attaching multiple files and honoring subjects/bodies.
  * Ensure tests are deterministic and do not require real Pandoc/mail or network access.

## Confirmed decisions

* **Language and stack**
  * Implement in Go, compiled to a binary for easy distribution and usage.
* **Default output directory**
  * For conversions initiated by the root or `convert` command, default output directory is `./kindbeam_out` relative to the current working directory, unless `--output-dir` is provided.
  * The `./kindbeam_out` directory is created on demand if it does not exist.
* **Email granularity**
  * For multiple input files, send one email per converted output file.
* **Cleanup semantics**
  * When the root workflow performs convert+send and send succeeds, delete the converted files by default so no temporary artifacts remain.
  * `--no-clean` on the root command preserves converted output files in the chosen output directory (default `./kindbeam_out`).
  * `convert` leaves converted files in the output directory, and `send` never deletes or relocates user-provided files.
* **Extra Pandoc args**
  * `--pandoc-args "<args>"` is parsed using a shellwords-style parser to support spaces and quoting, then appended to the Pandoc argument list.
* **Config overrides**
  * Support a `KINDLEBEAM_CONFIG` environment variable to point to an alternate JSON config file, overriding the default `os.UserConfigDir()/kindlebeam/config.json` location.
