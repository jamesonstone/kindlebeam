package kindlebeam

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jamesonstone/kindlebeam/internal/config"
)

func newInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "interactive setup wizard for kindlebeam",
		Long:  "walk through a series of questions to configure kindlebeam for first use",
		RunE:  runInit,
	}

	return cmd
}

func runInit(cmd *cobra.Command, args []string) error {
	fmt.Println("\n✨ Welcome to kindlebeam setup!")
	fmt.Println("==================================================")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	// Ask for Kindle email (required)
	kindleEmail := promptRequired(reader, "Kindle email address", "")
	if kindleEmail == "" {
		return fmt.Errorf("kindle email is required to use kindlebeam")
	}

	// Ask for default output format (optional)
	defaultOutputFormat := promptOptional(reader, "Default output format (e.g., pdf, epub, docx)", "pdf")

	// Ask for pandoc path (optional)
	pandocPath := promptOptional(reader, "Path to pandoc executable (leave blank for auto-detect)", "")

	// Ask for mail command (optional)
	mailCommand := promptOptional(reader, "Mail command (leave blank for auto-detect)", "")

	// Build config
	cfg := config.Config{
		KindleEmail:         kindleEmail,
		DefaultOutputFormat: defaultOutputFormat,
		DefaultInputFormat:  "markdown", // always use markdown as default
		PandocPath:          pandocPath,
		MailCommand:         mailCommand,
	}

	// Save config
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Println()
	fmt.Println("✅ Configuration saved!")
	fmt.Println()
	fmt.Println("You're all set! Try converting a document:")
	fmt.Println("  kindlebeam docs/readme.md")
	fmt.Println()

	return nil
}

// promptRequired repeatedly asks for input until a non-empty value is provided.
func promptRequired(reader *bufio.Reader, prompt, defaultValue string) string {
	for {
		fmt.Print("📝 " + prompt)
		if defaultValue != "" {
			fmt.Print(" [" + defaultValue + "]")
		}
		fmt.Print(": ")

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input != "" {
			return input
		}
		if defaultValue != "" {
			return defaultValue
		}

		fmt.Println("⚠️  This field is required, please enter a value.")
	}
}

// promptOptional asks for input and returns the provided value or default.
func promptOptional(reader *bufio.Reader, prompt, defaultValue string) string {
	fmt.Print("📝 " + prompt)
	if defaultValue != "" {
		fmt.Print(" [" + defaultValue + "]")
	}
	fmt.Print(": ")

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input != "" {
		return input
	}
	return defaultValue
}
