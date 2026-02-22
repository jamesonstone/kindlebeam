package kindlebeam

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jamesonstone/kindlebeam/internal/app"
	"github.com/jamesonstone/kindlebeam/internal/config"
)

func newConvertCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "convert [flags] <input-file>...",
		Short: "convert documents with pandoc",
		Args:  cobra.MinimumNArgs(1),
		RunE:  runConvert,
	}

	cmd.Flags().String("from", "", "pandoc input format (alias for --input-format)")
	cmd.Flags().String("to", "", "pandoc output format (alias for --output-format)")
	cmd.Flags().String("input-format", "", "pandoc input format (default: autodetect → markdown)")
	cmd.Flags().String("output-format", "", "pandoc output format (default: pdf)")
	cmd.Flags().String("output-dir", "", "directory for converted files (default: ./kindlebeam_out)")
	cmd.Flags().String("pandoc-args", "", "extra arguments to pass to pandoc (quoted string)")

	return cmd
}

func runConvert(cmd *cobra.Command, args []string) error {
	cfg, cfgPath, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	logger := app.NewLogger(verbose)
	logger.Debugf("using config at %s", cfgPath)

	inputFormat, _ := cmd.Flags().GetString("input-format")
	from, _ := cmd.Flags().GetString("from")
	if inputFormat == "" {
		inputFormat = from
	}

	outputFormat, _ := cmd.Flags().GetString("output-format")
	to, _ := cmd.Flags().GetString("to")
	if outputFormat == "" {
		outputFormat = to
	}

	outputDir, _ := cmd.Flags().GetString("output-dir")
	pandocArgs, _ := cmd.Flags().GetString("pandoc-args")

	options := app.ConvertOptions{
		InputFormat:   inputFormat,
		OutputFormat:  outputFormat,
		OutputDir:     outputDir,
		DryRun:        dryRun,
		PandocArgsRaw: pandocArgs,
	}

	workflow, err := app.NewWorkflow(cfg, logger)
	if err != nil {
		return err
	}

	ctx := context.Background()
	_, err = workflow.ConvertOnly(ctx, args, options)
	return err
}
