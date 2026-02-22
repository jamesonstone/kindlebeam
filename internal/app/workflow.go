package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mattn/go-shellwords"

	"github.com/jamesonstone/kindlebeam/internal/config"
	"github.com/jamesonstone/kindlebeam/internal/mailer"
	"github.com/jamesonstone/kindlebeam/internal/pandoc"
)

const defaultOutputDir = "kindlebeam_out"

// dangerousPandocArgs contains pandoc flags that can execute arbitrary code
// or access the filesystem in unsafe ways.
var dangerousPandocArgs = map[string]bool{
	"--lua-filter":        true,
	"--filter":            true,
	"-F":                  true,
	"--extract-media":     true,
	"--include-in-header": true,
	"-H":                  true,
	"--include-before":    true,
	"-B":                  true,
	"--include-after":     true,
	"-A":                  true,
	"--reference-doc":     true,
	"--data-dir":          true,
	"--syntax-definition": true,
	"--abbreviations":     true,
	"--log":               true,
}

// validatePandocArgs checks that no dangerous arguments are present.
func validatePandocArgs(args []string) error {
	for _, arg := range args {
		// Handle both "--flag=value" and "--flag" formats
		flag := arg
		if idx := strings.Index(arg, "="); idx != -1 {
			flag = arg[:idx]
		}
		if dangerousPandocArgs[flag] {
			return fmt.Errorf("pandoc argument %q is not allowed for security reasons", flag)
		}
	}
	return nil
}

// validateFilePath sanitizes and validates a file path to prevent path traversal attacks.
// It returns the cleaned absolute path or an error if the path is invalid.
func validateFilePath(path string) (string, error) {
	// Clean the path to resolve . and .. components
	cleaned := filepath.Clean(path)

	// Convert to absolute path
	absPath, err := filepath.Abs(cleaned)
	if err != nil {
		return "", fmt.Errorf("invalid path %q: %w", path, err)
	}

	// Verify the file exists and get its real path (resolving symlinks)
	realPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		// If file doesn't exist yet, that's okay for output paths
		// but for input files we need to check existence separately
		if os.IsNotExist(err) {
			return absPath, nil
		}
		return "", fmt.Errorf("invalid path %q: %w", path, err)
	}

	return realPath, nil
}

// validateInputFile validates an input file path and ensures it exists and is readable.
func validateInputFile(path string) (string, error) {
	validPath, err := validateFilePath(path)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(validPath)
	if err != nil {
		return "", fmt.Errorf("cannot access %q: %w", path, err)
	}

	if info.IsDir() {
		return "", fmt.Errorf("%q is a directory, not a file", path)
	}

	return validPath, nil
}

// WorkflowOptions controls the root convert+send workflow.
type WorkflowOptions struct {
	InputFormat   string
	OutputFormat  string
	OutputDir     string
	KindleEmail   string
	Subject       string
	DryRun        bool
	NoSend        bool
	NoClean       bool
	PandocArgsRaw string
}

// ConvertOptions controls conversion-only workflows.
type ConvertOptions struct {
	InputFormat   string
	OutputFormat  string
	OutputDir     string
	DryRun        bool
	PandocArgsRaw string
}

// SendOptions controls send-only workflows.
type SendOptions struct {
	KindleEmail string
	Subject     string
	Body        string
	DryRun      bool
}

type Workflow struct {
	cfg    config.Config
	log    *Logger
	pdc    *pandoc.Client
	mailer *mailer.Client
}

func NewWorkflow(cfg config.Config, log *Logger) (*Workflow, error) {
	pdc, err := pandoc.NewClient(cfg.EffectivePandocPath())
	if err != nil {
		return nil, err
	}

	ml, err := mailer.NewClient(cfg.EffectiveMailCommand())
	if err != nil {
		return nil, err
	}

	return &Workflow{
		cfg:    cfg,
		log:    log,
		pdc:    pdc,
		mailer: ml,
	}, nil
}

// ConvertAndSend performs convert+send for the given input files.
func (w *Workflow) ConvertAndSend(ctx context.Context, files []string, opts WorkflowOptions) error {
	convOpts := ConvertOptions{
		InputFormat:   opts.InputFormat,
		OutputFormat:  opts.OutputFormat,
		OutputDir:     opts.OutputDir,
		DryRun:        opts.DryRun,
		PandocArgsRaw: opts.PandocArgsRaw,
	}

	outputs, err := w.ConvertOnly(ctx, files, convOpts)
	if err != nil {
		return err
	}

	if opts.NoSend {
		return nil
	}

	sendOpts := SendOptions{
		KindleEmail: opts.KindleEmail,
		Subject:     opts.Subject,
		Body:        "sent with kindlebeam",
		DryRun:      opts.DryRun,
	}

	for i, out := range outputs {
		input := files[i]
		subject := sendOpts.Subject
		if subject == "" {
			subject = deriveSubjectFromFile(input)
		}

		if err := w.sendFile(ctx, out, subject, sendOpts); err != nil {
			return err
		}

		if !opts.DryRun && !opts.NoClean {
			if err := os.Remove(out); err != nil {
				w.log.Errorf("failed to remove %s: %v", out, err)
			}
		}
	}

	return nil
}

// ConvertOnly converts input files and returns the list of output file paths.
func (w *Workflow) ConvertOnly(ctx context.Context, files []string, opts ConvertOptions) ([]string, error) {
	if len(files) == 0 {
		return nil, fmt.Errorf("no input files provided")
	}

	outputDir := opts.OutputDir
	if outputDir == "" {
		outputDir = defaultOutputDir
	}

	if !opts.DryRun {
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			return nil, fmt.Errorf("create output dir: %w", err)
		}
	}

	parser := shellwords.NewParser()

	outputs := make([]string, 0, len(files))
	for _, in := range files {
		// Validate and sanitize the input file path
		validPath, err := validateInputFile(in)
		if err != nil {
			return nil, err
		}
		in = validPath

		inFormat := opts.InputFormat
		if inFormat == "" {
			inFormat = w.inferInputFormat(in)
		}
		if inFormat == "" {
			inFormat = w.cfg.DefaultInput()
		}

		outFormat := opts.OutputFormat
		if outFormat == "" {
			outFormat = w.cfg.DefaultOutput()
		}

		outPath := buildOutputPath(outputDir, in, outFormat)

		var extraArgs []string
		if opts.PandocArgsRaw != "" {
			args, err := parser.Parse(opts.PandocArgsRaw)
			if err != nil {
				return nil, fmt.Errorf("parse pandoc-args: %w", err)
			}
			// Validate that no dangerous arguments are present
			if err := validatePandocArgs(args); err != nil {
				return nil, err
			}
			extraArgs = args
		}

		if opts.DryRun {
			cmd := buildPandocCommandPreview(w.pdc.Binary(), inFormat, outFormat, in, outPath, extraArgs)
			w.log.Infof("would run: %s", strings.Join(cmd, " "))
		} else {
			w.log.Infof("converting %s → %s", in, outPath)
			result, err := w.pdc.Convert(ctx, pandoc.ConvertRequest{
				InputFormat:  inFormat,
				OutputFormat: outFormat,
				InputFile:    in,
				OutputFile:   outPath,
				ExtraArgs:    extraArgs,
			})
			if err != nil {
				if result.Stderr != "" {
					w.log.Errorf("pandoc stderr: %s", strings.TrimSpace(result.Stderr))
				}
				return nil, err
			}
			w.log.Successf("converted %s", outPath)
		}

		outputs = append(outputs, outPath)
	}

	return outputs, nil
}

// SendOnly sends existing files without performing conversions.
func (w *Workflow) SendOnly(ctx context.Context, files []string, opts SendOptions) error {
	if len(files) == 0 {
		return fmt.Errorf("no files to send")
	}

	for _, f := range files {
		// Validate and sanitize the file path
		validPath, err := validateInputFile(f)
		if err != nil {
			return err
		}
		f = validPath

		subject := opts.Subject
		if subject == "" {
			subject = deriveSubjectFromFile(f)
		}

		if err := w.sendFile(ctx, f, subject, opts); err != nil {
			return err
		}
	}

	return nil
}

func (w *Workflow) sendFile(ctx context.Context, filePath, subject string, opts SendOptions) error {
	body := opts.Body
	if body == "" {
		body = "sent with kindlebeam"
	}

	if opts.DryRun {
		cmd := buildMailCommandPreview(w.mailer.Binary(), opts.KindleEmail, subject, []string{filePath})
		w.log.Infof("would run: %s", strings.Join(cmd, " "))
		return nil
	}

	w.log.Infof("sending %s to %s", filePath, opts.KindleEmail)
	result, err := w.mailer.Send(ctx, mailer.SendRequest{
		To:          opts.KindleEmail,
		Subject:     subject,
		Body:        body,
		Attachments: []string{filePath},
	})
	if err != nil {
		if result.Stderr != "" {
			w.log.Errorf("mail stderr: %s", strings.TrimSpace(result.Stderr))
		}
		return err
	}

	w.log.Successf("sent %s", filePath)
	return nil
}

func (w *Workflow) inferInputFormat(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".md", ".markdown":
		return "markdown"
	case ".org":
		return "org"
	case ".rst":
		return "rst"
	case ".tex":
		return "latex"
	case ".html", ".htm":
		return "html"
	case ".docx":
		return "docx"
	case ".epub":
		return "epub"
	default:
		return ""
	}
}

func buildOutputPath(outputDir, inputPath, outputFormat string) string {
	base := filepath.Base(inputPath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	var outExt string
	switch strings.ToLower(outputFormat) {
	case "pdf":
		outExt = ".pdf"
	case "epub":
		outExt = ".epub"
	case "docx":
		outExt = ".docx"
	default:
		outExt = "." + strings.ToLower(outputFormat)
	}

	return filepath.Join(outputDir, name+outExt)
}

func deriveSubjectFromFile(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	if name == "" {
		return base
	}
	return name
}

func buildPandocCommandPreview(binary, inputFormat, outputFormat, inputFile, outputFile string, extraArgs []string) []string {
	args := []string{"-f", inputFormat, "-t", outputFormat, "-o", outputFile, inputFile}
	if len(extraArgs) > 0 {
		args = append(args, extraArgs...)
	}
	return append([]string{binary}, args...)
}

func buildMailCommandPreview(binary, to, subject string, attachments []string) []string {
	args := []string{"-s", subject}
	for _, a := range attachments {
		args = append(args, "-a", a)
	}
	args = append(args, to)
	return append([]string{binary}, args...)
}
