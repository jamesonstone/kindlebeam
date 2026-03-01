package mailer

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestAttachmentFlagUnsupported(t *testing.T) {
	cases := []struct {
		name   string
		stderr string
		want   bool
	}{
		{name: "illegal option", stderr: "mail: illegal option -- a", want: true},
		{name: "unknown option", stderr: "mail: unknown option -- a", want: true},
		{name: "invalid option", stderr: "mail: invalid option -- a", want: true},
		{name: "other error", stderr: "mail: cannot open mailbox", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := attachmentFlagUnsupported(tc.stderr)
			if got != tc.want {
				t.Fatalf("attachmentFlagUnsupported(%q) = %v, want %v", tc.stderr, got, tc.want)
			}
		})
	}
}

func TestSendWithMailFallsBackToSendmail(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-script based test")
	}

	dir := t.TempDir()
	mailPath := filepath.Join(dir, "mail")
	sendmailPath := filepath.Join(dir, "sendmail")
	capturePath := filepath.Join(dir, "captured_message.txt")
	attachmentPath := filepath.Join(dir, "test.txt")

	writeExecutable(t, mailPath, "#!/bin/sh\necho \"mail: illegal option -- a\" >&2\nexit 64\n")
	writeExecutable(t, sendmailPath, "#!/bin/sh\n/bin/cat > \"$KINDLEBEAM_TEST_CAPTURE\"\n")

	if err := os.WriteFile(attachmentPath, []byte("hello from attachment"), 0o644); err != nil {
		t.Fatalf("write attachment: %v", err)
	}

	t.Setenv("PATH", dir)
	t.Setenv("KINDLEBEAM_TEST_CAPTURE", capturePath)

	c := &Client{binary: mailPath, binaryName: "mail"}
	result, err := c.Send(context.Background(), SendRequest{
		To:          "user@kindle.com",
		Subject:     "subject",
		Body:        "sent with kindlebeam",
		Attachments: []string{attachmentPath},
	})
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}

	if len(result.Command) < 2 || result.Command[0] != sendmailPath || result.Command[1] != "-t" {
		t.Fatalf("fallback command = %v, want [%s -t]", result.Command, sendmailPath)
	}

	data, err := os.ReadFile(capturePath)
	if err != nil {
		t.Fatalf("read captured message: %v", err)
	}

	msg := string(data)
	if !strings.Contains(msg, "To: user@kindle.com") {
		t.Fatalf("captured message missing recipient header")
	}
	if !strings.Contains(msg, "Content-Disposition: attachment; filename=\"test.txt\"") {
		t.Fatalf("captured message missing attachment headers")
	}
}

func TestSendWithMailReturnsErrorWhenFallbackUnavailable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-script based test")
	}

	dir := t.TempDir()
	mailPath := filepath.Join(dir, "mail")
	attachmentPath := filepath.Join(dir, "test.txt")

	writeExecutable(t, mailPath, "#!/bin/sh\necho \"mail: illegal option -- a\" >&2\nexit 64\n")
	if err := os.WriteFile(attachmentPath, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write attachment: %v", err)
	}

	t.Setenv("PATH", dir)

	c := &Client{binary: mailPath, binaryName: "mail"}
	_, err := c.Send(context.Background(), SendRequest{
		To:          "user@kindle.com",
		Subject:     "subject",
		Body:        "body",
		Attachments: []string{attachmentPath},
	})
	if err == nil {
		t.Fatalf("expected error when sendmail fallback is unavailable")
	}
	if !strings.Contains(err.Error(), "sendmail was not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func writeExecutable(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write executable %s: %v", path, err)
	}
}
