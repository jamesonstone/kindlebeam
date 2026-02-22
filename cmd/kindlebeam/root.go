package kindlebeam

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jamesonstone/kindlebeam/internal/app"
	"github.com/jamesonstone/kindlebeam/internal/config"
)

const version = "0.1.0"

var (
	verbose bool
	dryRun  bool
)

// Execute runs the root command.
func Execute() {
	if err := newRootCommand().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "❌", err)
		os.Exit(1)
	}
}

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "kindlebeam [flags] <input-file>...",
		Short:         "convert documents with pandoc and beam them to your Kindle",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          runRoot,
	}

	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose logging")
	cmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "print actions without executing pandoc or mail")

	cmd.Flags().String("input-format", "", "pandoc input format (default: autodetect → markdown)")
	cmd.Flags().String("output-format", "", "pandoc output format (default: pdf)")
	cmd.Flags().String("output-dir", "", "directory for converted files (default: ./kindlebeam_out)")
	cmd.Flags().String("kindle-email", "", "override kindle email address")
	cmd.Flags().String("subject", "", "email subject (default: derived from filename)")
	cmd.Flags().Bool("no-send", false, "convert only, do not send to kindle")
	cmd.Flags().Bool("no-clean", false, "do not delete converted files after successful send")
	cmd.Flags().String("pandoc-args", "", "extra arguments to pass to pandoc (quoted string)")

	cmd.AddCommand(newInitCommand())
	cmd.AddCommand(newConvertCommand())
	cmd.AddCommand(newSendCommand())
	cmd.AddCommand(newConfigCommand())

	cmd.Version = version
	cmd.PersistentFlags().Bool("version", false, "show version and exit")
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		showVersion, _ := cmd.Flags().GetBool("version")
		if showVersion {
			fmt.Fprintln(os.Stdout, version)
			os.Exit(0)
		}
		return nil
	}

	return cmd
}

func runRoot(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		showWelcomeScreen(cmd)
		return nil
	}

	cfg, cfgPath, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	logger := app.NewLogger(verbose)
	logger.Debugf("using config at %s", cfgPath)

	options, err := buildWorkflowOptions(cmd, cfg)
	if err != nil {
		return err
	}

	workflow, err := app.NewWorkflow(cfg, logger)
	if err != nil {
		return err
	}

	ctx := context.Background()
	return workflow.ConvertAndSend(ctx, args, options)
}

func buildWorkflowOptions(cmd *cobra.Command, cfg config.Config) (app.WorkflowOptions, error) {
	inputFormat, _ := cmd.Flags().GetString("input-format")
	outputFormat, _ := cmd.Flags().GetString("output-format")
	outputDir, _ := cmd.Flags().GetString("output-dir")
	kindleEmailFlag, _ := cmd.Flags().GetString("kindle-email")
	subject, _ := cmd.Flags().GetString("subject")
	noSend, _ := cmd.Flags().GetBool("no-send")
	noClean, _ := cmd.Flags().GetBool("no-clean")
	pandocArgs, _ := cmd.Flags().GetString("pandoc-args")

	options := app.WorkflowOptions{
		InputFormat:   inputFormat,
		OutputFormat:  outputFormat,
		OutputDir:     outputDir,
		DryRun:        dryRun,
		NoSend:        noSend,
		NoClean:       noClean,
		Subject:       subject,
		PandocArgsRaw: pandocArgs,
	}

	options.KindleEmail = cfg.EffectiveKindleEmail(kindleEmailFlag)
	if !options.NoSend && options.KindleEmail == "" {
		return app.WorkflowOptions{}, fmt.Errorf("kindle email is not configured; run 'kindlebeam config set kindle-email <email>' or use --kindle-email")
	}

	return options, nil
}

func showWelcomeScreen(cmd *cobra.Command) {
	const blue = "\033[38;5;33m"    // Bright blue
	const orange = "\033[38;5;208m" // Orange
	const reset = "\033[0m"         // Reset colors

	fmt.Println()
	fmt.Print(blue)
	fmt.Println("██╗  ██╗██╗███╗   ██╗██████╗ ██╗     ███████╗██████╗ ███████╗ █████╗ ███╗   ███╗")
	fmt.Print(orange)
	fmt.Println("██║ ██╔╝██║████╗  ██║██╔══██╗██║     ██╔════╝██╔══██╗██╔════╝██╔══██╗████╗ ████║")
	fmt.Print(blue)
	fmt.Println("█████╔╝ ██║██╔██╗ ██║██║  ██║██║     █████╗  ██████╔╝█████╗  ███████║██╔████╔██║")
	fmt.Print(orange)
	fmt.Println("██╔═██╗ ██║██║╚██╗██║██║  ██║██║     ██╔══╝  ██╔══██╗██╔══╝  ██╔══██║██║╚██╔╝██║")
	fmt.Print(blue)
	fmt.Println("██║  ██╗██║██║ ╚████║██████╔╝███████╗███████╗██████╔╝███████╗██║  ██║██║ ╚═╝ ██║")
	fmt.Print(orange)
	fmt.Println("╚═╝  ╚═╝╚═╝╚═╝  ╚═══╝╚═════╝ ╚══════╝╚══════╝╚═════╝ ╚══════╝╚═╝  ╚═╝╚═╝     ╚═╝")
	fmt.Print(reset)
	fmt.Println()
	fmt.Print(blue)
	fmt.Println("Convert documents to Kindle-friendly formats")
	fmt.Print(orange)
	fmt.Println("and send them instantly.")
	fmt.Print(reset)
	fmt.Println()
	fmt.Println()
	fmt.Println("Quick start:")
	fmt.Println()
	fmt.Println("  1. Set up kindlebeam (first time only):")
	fmt.Println("     kindlebeam init")
	fmt.Println()
	fmt.Println("  2. Convert and send a document:")
	fmt.Println("     kindlebeam docs/readme.md")
	fmt.Println()
	fmt.Println("Common tasks:")
	fmt.Println()
	fmt.Println("  Convert to PDF (default):")
	fmt.Println("    kindlebeam notes.md")
	fmt.Println()
	fmt.Println("  Convert without sending:")
	fmt.Println("    kindlebeam --no-send document.md")
	fmt.Println()
	fmt.Println("  Convert to a different format:")
	fmt.Println("    kindlebeam --output-format epub book.md")
	fmt.Println()
	fmt.Println("  Preview what would happen (dry-run):")
	fmt.Println("    kindlebeam --dry-run --verbose notes.md")
	fmt.Println()
	fmt.Println("For more information:")
	fmt.Println()
	fmt.Println("  " + cmd.UsageString())
	fmt.Println()
}
