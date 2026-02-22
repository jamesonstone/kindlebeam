package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jamesonstone/kindlebeam/internal/config"
)

func TestInferInputFormat(t *testing.T) {
	w := &Workflow{cfg: config.Config{}}

	cases := map[string]string{
		"foo.md":      "markdown",
		"bar.org":     "org",
		"baz.rst":     "rst",
		"doc.tex":     "latex",
		"page.html":   "html",
		"paper.docx":  "docx",
		"book.epub":   "epub",
		"unknown.txt": "",
	}

	for name, want := range cases {
		got := w.inferInputFormat(name)
		if got != want {
			t.Errorf("inferInputFormat(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestBuildOutputPath(t *testing.T) {
	out := buildOutputPath("out", "doc.md", "pdf")
	if out != filepath.Join("out", "doc.pdf") {
		t.Fatalf("unexpected output path: %s", out)
	}

	out = buildOutputPath("out", "book.md", "epub")
	if out != filepath.Join("out", "book.epub") {
		t.Fatalf("unexpected output path: %s", out)
	}
}

func TestConfigDefaults(t *testing.T) {
	cfg := config.Config{}
	if got := cfg.DefaultInput(); got != "markdown" {
		t.Fatalf("DefaultInput = %q, want markdown", got)
	}
	if got := cfg.DefaultOutput(); got != "pdf" {
		t.Fatalf("DefaultOutput = %q, want pdf", got)
	}
	if got := cfg.EffectiveMailCommand(); got != "mail" {
		t.Fatalf("EffectiveMailCommand = %q, want mail", got)
	}
	if got := cfg.EffectivePandocPath(); got != "pandoc" {
		t.Fatalf("EffectivePandocPath = %q, want pandoc", got)
	}
}

func TestEffectiveKindleEmail(t *testing.T) {
	cfg := config.Config{
		KindleEmail:        "primary@example.com",
		DefaultKindleEmail: "default@example.com",
	}
	if got := cfg.EffectiveKindleEmail(""); got != "primary@example.com" {
		t.Fatalf("EffectiveKindleEmail(empty) = %q, want primary@example.com", got)
	}
	if got := cfg.EffectiveKindleEmail("override@example.com"); got != "override@example.com" {
		t.Fatalf("EffectiveKindleEmail(override) = %q, want override@example.com", got)
	}
}

func TestResolvePathUsesEnvOverride(t *testing.T) {
	// create a temp file and point KINDLEBEAM_CONFIG to it
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	t.Setenv("KINDLEBEAM_CONFIG", path)

	cfg, gotPath, err := config.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if gotPath != path {
		t.Fatalf("Load path = %q, want %q", gotPath, path)
	}
	if cfg.DefaultInputFormat == "" || cfg.DefaultOutputFormat == "" {
		t.Fatalf("expected defaults applied, got %+v", cfg)
	}
}

func TestValidatePandocArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "safe args",
			args:    []string{"--toc", "--standalone", "-V", "geometry:margin=1in"},
			wantErr: false,
		},
		{
			name:    "empty args",
			args:    []string{},
			wantErr: false,
		},
		{
			name:    "dangerous lua-filter",
			args:    []string{"--lua-filter=/path/to/evil.lua"},
			wantErr: true,
		},
		{
			name:    "dangerous filter short flag",
			args:    []string{"-F", "evil-filter"},
			wantErr: true,
		},
		{
			name:    "dangerous extract-media",
			args:    []string{"--extract-media=/tmp"},
			wantErr: true,
		},
		{
			name:    "dangerous include-in-header",
			args:    []string{"--include-in-header=/etc/passwd"},
			wantErr: true,
		},
		{
			name:    "dangerous -H flag",
			args:    []string{"-H", "/etc/passwd"},
			wantErr: true,
		},
		{
			name:    "mixed safe and dangerous",
			args:    []string{"--toc", "--filter=evil"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePandocArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePandocArgs(%v) error = %v, wantErr %v", tt.args, err, tt.wantErr)
			}
		})
	}
}

func TestValidateInputFile(t *testing.T) {
	// Create a temp file for testing
	dir := t.TempDir()
	validFile := filepath.Join(dir, "test.md")
	if err := os.WriteFile(validFile, []byte("# Test"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	// Create a subdirectory
	subdir := filepath.Join(dir, "subdir")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("create subdir: %v", err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid file",
			path:    validFile,
			wantErr: false,
		},
		{
			name:    "nonexistent file",
			path:    filepath.Join(dir, "nonexistent.md"),
			wantErr: true,
		},
		{
			name:    "directory not allowed",
			path:    subdir,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validateInputFile(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateInputFile(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}
