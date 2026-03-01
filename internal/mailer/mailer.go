package mailer

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type Client struct {
	binary     string
	binaryName string
}

func attachmentFlagUnsupported(stderr string) bool {
	msg := strings.ToLower(stderr)
	return strings.Contains(msg, "illegal option -- a") ||
		strings.Contains(msg, "unknown option -- a") ||
		strings.Contains(msg, "invalid option -- a")
}

type SendRequest struct {
	To          string
	Subject     string
	Body        string
	Attachments []string
}

type SendResult struct {
	Command []string
	Stderr  string
}

func NewClient(binary string) (*Client, error) {
	if binary == "" {
		binary = detectMailBinary()
	}
	path, err := exec.LookPath(binary)
	if err != nil {
		return nil, fmt.Errorf("mail command not found (%s): %w\nOn macOS, install mailutils: brew install mailutils", binary, err)
	}
	return &Client{
		binary:     path,
		binaryName: filepath.Base(binary),
	}, nil
}

// detectMailBinary returns the best available mail binary for the current platform.
func detectMailBinary() string {
	// Prefer GNU mail (supports -a for attachments) if available
	if path, err := exec.LookPath("mail"); err == nil {
		// Check if it's GNU mail by looking for -a support
		if runtime.GOOS == "linux" {
			return path
		}
	}

	// On macOS, the built-in /usr/bin/mail doesn't support attachments
	// Try to find alternatives
	alternatives := []string{
		"mailx",    // Often GNU mailx with -a support
		"s-nail",   // Modern mail with attachment support
		"mutt",     // mutt supports attachments
		"sendmail", // Fall back to sendmail with MIME
	}

	for _, alt := range alternatives {
		if _, err := exec.LookPath(alt); err == nil {
			return alt
		}
	}

	// Default to mail, will fail with helpful error if attachments needed
	return "mail"
}

func (c *Client) Binary() string {
	return c.binary
}

func (c *Client) Send(ctx context.Context, req SendRequest) (SendResult, error) {
	if req.To == "" {
		return SendResult{}, fmt.Errorf("mail: missing recipient")
	}

	// Choose sending method based on binary and platform
	switch {
	case c.binaryName == "sendmail" || strings.HasSuffix(c.binary, "/sendmail"):
		return c.sendWithSendmail(ctx, req)
	case c.binaryName == "mutt":
		return c.sendWithMutt(ctx, req)
	case c.binaryName == "s-nail":
		return c.sendWithSNail(ctx, req)
	default:
		return c.sendWithMail(ctx, req)
	}
}

// sendWithMail uses the standard mail/mailx command (GNU version with -a support)
func (c *Client) sendWithMail(ctx context.Context, req SendRequest) (SendResult, error) {
	args := []string{"-s", req.Subject}
	for _, a := range req.Attachments {
		args = append(args, "-a", a)
	}
	args = append(args, req.To)

	cmd := exec.CommandContext(ctx, c.binary, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdin = bytes.NewBufferString(req.Body)

	if err := cmd.Run(); err != nil {
		stderrStr := stderr.String()
		if len(req.Attachments) > 0 && attachmentFlagUnsupported(stderrStr) {
			if fallbackPath, lookupErr := exec.LookPath("sendmail"); lookupErr == nil {
				fallback := &Client{
					binary:     fallbackPath,
					binaryName: filepath.Base(fallbackPath),
				}
				return fallback.sendWithSendmail(ctx, req)
			}
			return SendResult{Command: append([]string{c.binary}, args...), Stderr: stderrStr},
				fmt.Errorf("mail failed: built-in mail does not support attachments and sendmail was not found.\n" +
					"Install GNU mailutils: brew install mailutils\n" +
					"Or configure a different mail command: kindlebeam config set mail-command <command>")
		}
		return SendResult{Command: append([]string{c.binary}, args...), Stderr: stderrStr}, fmt.Errorf("mail failed: %w", err)
	}

	return SendResult{Command: append([]string{c.binary}, args...), Stderr: stderr.String()}, nil
}

// sendWithMutt uses mutt which has good attachment support
func (c *Client) sendWithMutt(ctx context.Context, req SendRequest) (SendResult, error) {
	args := []string{"-s", req.Subject}
	for _, a := range req.Attachments {
		args = append(args, "-a", a)
	}
	args = append(args, "--", req.To)

	cmd := exec.CommandContext(ctx, c.binary, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdin = bytes.NewBufferString(req.Body)

	if err := cmd.Run(); err != nil {
		return SendResult{Command: append([]string{c.binary}, args...), Stderr: stderr.String()}, fmt.Errorf("mutt failed: %w", err)
	}

	return SendResult{Command: append([]string{c.binary}, args...), Stderr: stderr.String()}, nil
}

// sendWithSNail uses s-nail (modern BSD mail replacement)
func (c *Client) sendWithSNail(ctx context.Context, req SendRequest) (SendResult, error) {
	args := []string{"-s", req.Subject}
	for _, a := range req.Attachments {
		args = append(args, "-a", a)
	}
	args = append(args, req.To)

	cmd := exec.CommandContext(ctx, c.binary, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdin = bytes.NewBufferString(req.Body)

	if err := cmd.Run(); err != nil {
		return SendResult{Command: append([]string{c.binary}, args...), Stderr: stderr.String()}, fmt.Errorf("s-nail failed: %w", err)
	}

	return SendResult{Command: append([]string{c.binary}, args...), Stderr: stderr.String()}, nil
}

// sendWithSendmail constructs a MIME message and pipes it to sendmail
func (c *Client) sendWithSendmail(ctx context.Context, req SendRequest) (SendResult, error) {
	boundary := "----=_KindleBeam_Boundary_" + fmt.Sprintf("%d", os.Getpid())

	var msg bytes.Buffer
	msg.WriteString(fmt.Sprintf("To: %s\r\n", req.To))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", req.Subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n", boundary))
	msg.WriteString("\r\n")

	// Body part
	msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	msg.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(req.Body)
	msg.WriteString("\r\n")

	// Attachment parts
	for _, attachPath := range req.Attachments {
		data, err := os.ReadFile(attachPath)
		if err != nil {
			return SendResult{}, fmt.Errorf("read attachment %s: %w", attachPath, err)
		}

		filename := filepath.Base(attachPath)
		mimeType := detectMimeType(filename)

		msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		msg.WriteString(fmt.Sprintf("Content-Type: %s; name=\"%s\"\r\n", mimeType, filename))
		msg.WriteString("Content-Transfer-Encoding: base64\r\n")
		msg.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n", filename))
		msg.WriteString("\r\n")

		// Base64 encode in chunks of 76 characters
		encoded := base64.StdEncoding.EncodeToString(data)
		for i := 0; i < len(encoded); i += 76 {
			end := i + 76
			if end > len(encoded) {
				end = len(encoded)
			}
			msg.WriteString(encoded[i:end])
			msg.WriteString("\r\n")
		}
	}

	msg.WriteString(fmt.Sprintf("--%s--\r\n", boundary))

	args := []string{"-t"}
	cmd := exec.CommandContext(ctx, c.binary, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdin = &msg

	if err := cmd.Run(); err != nil {
		return SendResult{Command: append([]string{c.binary}, args...), Stderr: stderr.String()}, fmt.Errorf("sendmail failed: %w", err)
	}

	return SendResult{Command: append([]string{c.binary}, args...), Stderr: stderr.String()}, nil
}

func detectMimeType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".pdf":
		return "application/pdf"
	case ".epub":
		return "application/epub+zip"
	case ".mobi":
		return "application/x-mobipocket-ebook"
	case ".azw", ".azw3":
		return "application/vnd.amazon.ebook"
	case ".doc":
		return "application/msword"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".txt":
		return "text/plain"
	case ".html", ".htm":
		return "text/html"
	default:
		return "application/octet-stream"
	}
}
