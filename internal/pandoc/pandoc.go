package pandoc

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

type Client struct {
	binary string
}

type ConvertRequest struct {
	InputFormat  string
	OutputFormat string
	InputFile    string
	OutputFile   string
	ExtraArgs    []string
}

type ConvertResult struct {
	Command []string
	Stderr  string
}

func NewClient(binary string) (*Client, error) {
	if binary == "" {
		binary = "pandoc"
	}
	path, err := exec.LookPath(binary)
	if err != nil {
		return nil, fmt.Errorf("pandoc not found (%s): %w", binary, err)
	}
	return &Client{binary: path}, nil
}

func (c *Client) Binary() string {
	return c.binary
}

func (c *Client) Convert(ctx context.Context, req ConvertRequest) (ConvertResult, error) {
	args := []string{"-f", req.InputFormat, "-t", req.OutputFormat, "-o", req.OutputFile, req.InputFile}
	if len(req.ExtraArgs) > 0 {
		args = append(args, req.ExtraArgs...)
	}

	cmd := exec.CommandContext(ctx, c.binary, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return ConvertResult{Command: append([]string{c.binary}, args...), Stderr: stderr.String()}, fmt.Errorf("pandoc failed: %w", err)
	}

	return ConvertResult{Command: append([]string{c.binary}, args...), Stderr: stderr.String()}, nil
}
