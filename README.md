# kindlebeam

📚 a CLI tool to convert documents with pandoc and beam them to your Kindle email.

---

## overview

`kindlebeam` is a small Go-based CLI that:

- converts one or more input documents to a Kindle-friendly format using pandoc
- emails the converted files to your Kindle email address using the macOS `mail` command
- cleans up temporary converted files for the default workflow

Default workflow:

- input: markdown (`.md`)
- output: pdf
- action: convert → email to Kindle.

```sh
kindlebeam config set kindle-email your_name@kindle.com
kindlebeam docs/specs/file.md
```

This converts `file.md`→`./kindlebeam_out/file.pdf`, sends it to your Kindle, and removes the pdf when the email succeeds.

---

## requirements

### runtime dependencies

- `pandoc` on your `PATH` (or configured via `pandoc_path` in config)
- `mail` on your `PATH` (macOS BSD `mail`, or configured via `mail_command`)

### kindle setup

- you must have a Kindle email address (e.g. `your_name@kindle.com`)
- that address must be configured in your Amazon Kindle settings
- the sender address used by your system `mail` must be whitelisted in your Amazon “Approved Personal Document E-mail List”

---

## installation

### from source (recommended for now)

```sh
git clone https://github.com/jamesonstone/kindlebeam.git
cd kindlebeam
make build
./kindlebeam --version
```

This builds a single `kindlebeam` binary in the repo root.

To install it somewhere on your `PATH`:

```sh
cp kindlebeam /usr/local/bin/
```

### go install (if the module is available)

```sh
go install github.com/jamesonstone/kindlebeam@latest
```

This places the binary in your Go bin directory (often `~/go/bin`).

---

## prerequisite

install `pandoc` with homebrew: `brew install pandoc`

## quick start

1. configure your Kindle email:

   ```sh
   kindlebeam config set kindle-email your_name@kindle.com
   ```

2. convert and send a markdown file:

   ```sh
   kindlebeam notes/today.md
   ```

   behavior:

   - infers input format from `.md` → `markdown`
   - uses default output format `pdf`
   - writes `./kindlebeam_out/today.pdf`
   - sends `today.pdf` to your configured Kindle email
   - removes `today.pdf` after a successful send

3. preview what would happen (no pandoc, no mail):

   ```sh
   kindlebeam --dry-run --verbose notes/today.md
   ```

   this prints the pandoc and mail commands that would be executed but performs no changes.

---

## commands

### root: `kindlebeam`

Usage:

```sh
kindlebeam [flags] <input-file>...
```

behavior:

- when called without a subcommand, performs **convert + send** for each input file
- one email is sent per converted file

key flags:

- `--verbose, -v` – enable verbose logging
- `--dry-run` – compute work and print planned commands without running pandoc or mail
- `--input-format <format>` – explicit pandoc input format; default is autodetect by file extension → config default (`markdown`)
- `--output-format <format>` – explicit pandoc output format; default is config default (`pdf`)
- `--output-dir <path>` – directory for converted files (default: `./kindlebeam_out`)
- `--kindle-email <email>` – override Kindle email from config
- `--subject <subject>` – email subject (default derived from input filename)
- `--no-send` – convert only, do not email
- `--no-clean` – keep converted files after successful send
- `--pandoc-args "<args>"` – extra arguments passed verbatim to pandoc, parsed with a shellwords-style parser

special flag:

- `--version` – print version and exit

### `convert` command

Usage:

```sh
kindlebeam convert [flags] <input-file>...
```

behavior:

- convert one or more input files using pandoc
- writes outputs to `--output-dir` or `./kindlebeam_out`
- **does not** email files

flags:

- `--from, -f <format>` – input format (alias for `--input-format`)
- `--to, -t <format>` – output format (alias for `--output-format`)
- `--input-format <format>` – explicit input format
- `--output-format <format>` – explicit output format
- `--output-dir <path>` – output directory (default `./kindlebeam_out`)
- `--pandoc-args "<args>"` – extra pandoc arguments

examples:

```sh
# markdown → pdf into kindlebeam_out
kindlebeam convert docs/specs/file.md

# org → epub into custom directory
kindlebeam convert --from org --to epub --output-dir out docs/notes.org
```

### `send` command

Usage:

```sh
kindlebeam send [flags] <file>...
```

behavior:

- send one or more existing files as email attachments via `mail`
- never deletes or moves the provided files

flags:

- `--kindle-email <email>` – override Kindle email from config
- `--subject <subject>` – subject line (default derived from filename)
- `--body <text>` – email body text (default `"sent with kindlebeam"`)

example:

```sh
# send an already generated pdf
kindlebeam send --subject "weekly notes" kindlebeam_out/notes.pdf
```

### `config` command

Usage:

```sh
kindlebeam config <subcommand> [...]
```

subcommands:

- `show` – print effective configuration and resolved config file path
- `set kindle-email <email>` – set primary Kindle email
- `set default-output-format <format>` – set default pandoc output format
- `set mail-command <cmd>` – set the `mail` command or path
- `set pandoc-path <path>` – set the pandoc binary or path

examples:

```sh
kindlebeam config show
kindlebeam config set kindle-email your_name@kindle.com
kindlebeam config set default-output-format epub
kindlebeam config set mail-command mail
kindlebeam config set pandoc-path /usr/local/bin/pandoc
```

---

## configuration

### file location

by default, config is stored as JSON at:

- macOS: `${UserConfigDir}/kindlebeam/config.json` (typically `~/Library/Application Support/kindlebeam/config.json`)
- Linux: `${UserConfigDir}/kindlebeam/config.json` (typically `~/.config/kindlebeam/config.json`)

you can override the config path with the `KINDLEBEAM_CONFIG` environment variable:

```sh
export KINDLEBEAM_CONFIG=$HOME/.kindlebeam.json
kindlebeam config set kindle-email your_name@kindle.com
```

### schema

example `config.json`:

```json
{
  "kindle_email": "your_name@kindle.com",
  "default_kindle_email": "your_name@kindle.com",
  "default_output_format": "pdf",
  "default_input_format": "markdown",
  "mail_command": "mail",
  "pandoc_path": "pandoc"
}
```

field behavior:

- `kindle_email` – primary Kindle address used for sending
- `default_kindle_email` – legacy / fallback Kindle address if `kindle_email` is empty
- `default_output_format` – default pandoc output format (e.g. `pdf`, `epub`)
- `default_input_format` – used when file extension cannot be mapped
- `mail_command` – command or path for the `mail` binary
- `pandoc_path` – command or path for the `pandoc` binary

CLI flags always override config values for a single invocation.

---

## format detection

when `--input-format` / `--from` is not provided, `kindlebeam` infers formats from file extensions:

- `.md`, `.markdown` → `markdown`
- `.org` → `org`
- `.rst` → `rst`
- `.tex` → `latex`
- `.html`, `.htm` → `html`
- `.docx` → `docx`
- `.epub` → `epub`

if the extension is unknown, it falls back to `default_input_format` from config, then `markdown`.

output extensions are derived from the output format:

- `pdf` → `.pdf`
- `epub` → `.epub`
- `docx` → `.docx`
- anything else → `.<format>`

---

## logging and troubleshooting

logging uses emoji-prefixed, human-friendly messages:

- `ℹ️` info
- `✅` success
- `❌` error
- `🔍` debug (only when `--verbose` is enabled)

recommendations:

- use `--dry-run` first when wiring up new workflows or experimenting with `--pandoc-args`
- add `--verbose` when debugging failures to see more detail (including stderr from pandoc or mail)

common issues:

- **`pandoc not found`** – install pandoc or set `pandoc_path` in config
- **`mail not found`** – ensure macOS `mail` is available or adjust `mail_command`
- **`kindle email is not configured`** – run `kindlebeam config set kindle-email <email>` or pass `--kindle-email`
- **no document appears on Kindle** – verify your Kindle email and approved sender list in your Amazon account

---

## development

### project layout

- `main.go` – entrypoint calling the cobra root command
- `cmd/kindlebeam` – CLI commands (`kindlebeam`, `convert`, `send`, `config`)
- `internal/config` – config load/save and defaults
- `internal/app` – logger and workflows (convert, send, convert+send)
- `internal/pandoc` – thin wrapper around the `pandoc` binary
- `internal/mailer` – thin wrapper around the system `mail` binary

### build and test

use the `Makefile` targets:

```sh
make build   # go build -o kindlebeam .
make test    # go test ./...
make fmt     # gofmt -w ./
make lint    # golangci-lint run ./... (if installed)
```

### running locally

```sh
make build
./kindlebeam --help
./kindlebeam --dry-run examples/sample.md
```

---

## roadmap / future ideas

- support SMTP or API-based email backends (SES, SendGrid, etc.)
- combine multiple input files into a single document before sending
- inject metadata such as title and author into converted documents
- add a watch mode to auto-convert and send on file changes
- richer templating for email subjects and bodies
