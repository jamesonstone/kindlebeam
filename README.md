# kindlebeam

📚 a CLI tool to convert documents with pandoc and beam them to your Kindle email.

---

## overview

`kindlebeam` is a small Go-based CLI that:

- 📄 converts one or more input documents to a Kindle-friendly format using `pandoc`
- 📧 emails the converted files to your Kindle email address via automatic mail backend detection
- 🧹 cleans up temporary converted files after successful delivery
- 🔄 supports **Linux**, **macOS**, and other Unix-like systems

Default workflow: `kindlebeam notes/today.md` → infers format → converts → emails → cleans up

---

## requirements

### installation

**pandoc:**

```bash
brew install pandoc              # macOS
sudo apt install pandoc          # Ubuntu/Debian
sudo dnf install pandoc          # Fedora/RHEL
sudo pacman -S pandoc            # Arch
```

**mail binary** (auto-detected; macOS built-in `/usr/bin/mail` lacks attachment support):

```bash
brew install mailutils           # macOS (GNU mail) or
brew install mutt                # macOS alt: full-featured
sudo apt install mailutils       # Ubuntu/Debian
sudo dnf install mailx           # Fedora/RHEL
```

**build from source:**

```bash
git clone https://github.com/jamesonstone/kindlebeam.git && cd kindlebeam
make build
cp kindlebeam /usr/local/bin/    # optional: install to PATH
```

**or via go:**

```bash
go install github.com/jamesonstone/kindlebeam@latest
```

### Kindle setup

- ✉️ create/have a **Kindle email** address (e.g. `your_name@kindle.com`)
- 🔧 **whitelist your sender address** in Amazon Kindle settings → "Approved Personal Document E-mail List"

---

## quick start

```bash
# 1️⃣ configure Kindle email
kindlebeam config set kindle-email your_name@kindle.com

# 2️⃣ convert & send
kindlebeam notes/today.md
# → detects .md → converts to pdf → sends → cleans up

# 3️⃣ preview (dry run)
kindlebeam --dry-run --verbose notes/today.md
```

---

## commands

### `kindlebeam` (default: convert + send)

**Usage:** `kindlebeam [flags] <input-file>...`

| Flag                     | Purpose                                           |
| ------------------------ | ------------------------------------------------- |
| `--dry-run`              | 🏃 show commands, don't execute                   |
| `--verbose, -v`          | 🔍 detailed output & errors                       |
| `--input-format <fmt>`   | 📄 input format (auto-detected)                   |
| `--output-format <fmt>`  | 📄 output format (default: `pdf`)                 |
| `--output-dir <path>`    | 📁 output directory (default: `./kindlebeam_out`) |
| `--kindle-email <email>` | ✉️ override config email                          |
| `--subject <text>`       | 📝 email subject (default: filename)              |
| `--no-send`              | ❌ convert only                                   |
| `--no-clean`             | 🚫 keep converted files                           |
| `--pandoc-args "<args>"` | ⚙️ extra pandoc arguments                         |
| `--version`              | ℹ️ show version                                   |

### `convert` (pandoc only)

**Usage:** `kindlebeam convert [flags] <input-file>...`

```bash
kindlebeam convert docs/file.md                    # → pdf
kindlebeam convert --from org --to epub docs/file  # → epub
kindlebeam convert --to html --pandoc-args "--toc" file.md
```

| Flag                     | Purpose           |
| ------------------------ | ----------------- |
| `--from, -f <fmt>`       | input format      |
| `--to, -t <fmt>`         | output format     |
| `--output-dir <path>`    | output directory  |
| `--pandoc-args "<args>"` | extra pandoc args |

### `send` (email only)

**Usage:** `kindlebeam send [flags] <file>...`

```bash
kindlebeam send kindlebeam_out/notes.pdf
kindlebeam send --subject "reading list" *.pdf
kindlebeam send --kindle-email alternate@kindle.com file.pdf
```

| Flag                     | Purpose                                       |
| ------------------------ | --------------------------------------------- |
| `--kindle-email <email>` | override Kindle email                         |
| `--subject <text>`       | subject line                                  |
| `--body <text>`          | body text (default: `"sent with kindlebeam"`) |

### `config` (settings)

**Usage:** `kindlebeam config <subcommand>`

```bash
kindlebeam config show
kindlebeam config set kindle-email your_name@kindle.com
kindlebeam config set default-output-format epub
kindlebeam config set mail-command mutt
kindlebeam config set pandoc-path /usr/local/bin/pandoc
```

| Subcommand                        | Purpose                       |
| --------------------------------- | ----------------------------- |
| `show`                            | display config & location     |
| `set kindle-email <email>`        | set Kindle email              |
| `set default-output-format <fmt>` | default output format         |
| `set mail-command <cmd>`          | set mail binary/path          |
| `set pandoc-path <path>`          | set pandoc binary/path        |
| `set default-input-format <fmt>`  | format for unknown extensions |

---

## 🔧 configuration

**default location:**

- macOS: `~/Library/Application Support/kindlebeam/config.json`
- Linux: `~/.config/kindlebeam/config.json`
- override: `export KINDLEBEAM_CONFIG=$HOME/.kindlebeam.json`

**schema:**

```json
{
  "kindle_email": "your_name@kindle.com",
  "default_output_format": "pdf",
  "default_input_format": "markdown",
  "mail_command": "mail", // auto-detected if omitted
  "pandoc_path": "pandoc"
}
```

**precedence:** CLI flags > config values

---

## format detection

**input** (auto-detected from extension; fallback: `markdown`):
`.md/.markdown`→`markdown`, `.org`→`org`, `.rst`→`rst`, `.tex`→`latex`, `.html/.htm`→`html`, `.docx`→`docx`, `.epub`→`epub`

**output** (extension derived from format): `.pdf`, `.epub`, `.docx`, `.html`, etc.

---

## 📧 mail backends

**auto-detection order:** `mail`/`mailx` (GNU) → `mutt` → `s-nail` → `sendmail` (MIME fallback)

**⚠️ macOS note:** built-in `/usr/bin/mail` lacks attachment support; kindlebeam auto-falls back to `sendmail` when available.

**explicit configuration:**

```bash
kindlebeam config set mail-command mutt              # or: sendmail, s-nail, mailx
kindlebeam config set mail-command /usr/local/bin/mailx  # full path
```

---

## troubleshooting

**💡 tips:** Use `--dry-run --verbose` before sending; run `kindlebeam config show` to verify settings

**common issues:**

| Issue                            | Fix                                                                                                                                                                      |
| -------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `pandoc: command not found`      | Install pandoc (see [installation](#installation)) OR `kindlebeam config set pandoc-path /path/to/pandoc`                                                                |
| `mail command not found`         | Install mail binary (see [installation](#installation)) OR `kindlebeam config set mail-command mutt`                                                                     |
| macOS: attachment not supported  | kindlebeam auto-falls back to `sendmail`; if unavailable, install `mailutils` or set `mail-command` to `mutt`/`s-nail`                                                  |
| `kindle email is not configured` | `kindlebeam config set kindle-email your_name@kindle.com`                                                                                                                |
| document not on Kindle           | ✅ verify email in config; ✅ check Amazon Kindle settings; ✅ whitelist sender in "Approved Personal Document E-mail List"; ✅ run `kindlebeam send --verbose` for logs |
| `sendmail` errors                | Ensure local mail system running; try `kindlebeam config set mail-command mutt`                                                                                          |

---

## development

**layout:**

- `main.go` – entrypoint
- `cmd/kindlebeam/` – CLI commands (root, convert, send, config)
- `internal/config/` – config load/save
- `internal/app/` – workflows & logger
- `internal/pandoc/` – pandoc wrapper
- `internal/mailer/` – multi-backend mail client

**build:**

```bash
make build   # compile
make test    # run tests
make fmt     # format
make lint    # lint (if golangci-lint installed)

./kindlebeam --help
./kindlebeam --dry-run ~/notes/sample.md --verbose
```

---

## roadmap

- 📬 SMTP/API email backends (SES, SendGrid)
- 📦 combine multiple files before sending
- ✏️ document metadata injection
- 👀 watch mode for auto-conversion
- 🎨 email template support
- 🌐 GUI/web interface
