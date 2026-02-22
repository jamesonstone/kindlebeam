package kindlebeam

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jamesonstone/kindlebeam/internal/app"
	"github.com/jamesonstone/kindlebeam/internal/config"
)

func newSendCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send [flags] <file>...",
		Short: "send existing files to your kindle",
		Args:  cobra.MinimumNArgs(1),
		RunE:  runSend,
	}

	cmd.Flags().String("kindle-email", "", "override kindle email address")
	cmd.Flags().String("subject", "", "email subject (default: derived from filename)")
	cmd.Flags().String("body", "", "email body text")

	return cmd
}

func runSend(cmd *cobra.Command, args []string) error {
	cfg, cfgPath, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	logger := app.NewLogger(verbose)
	logger.Debugf("using config at %s", cfgPath)

	kindleEmailFlag, _ := cmd.Flags().GetString("kindle-email")
	subject, _ := cmd.Flags().GetString("subject")
	body, _ := cmd.Flags().GetString("body")

	options := app.SendOptions{
		Subject: subject,
		Body:    body,
	}

	options.KindleEmail = cfg.EffectiveKindleEmail(kindleEmailFlag)
	if options.KindleEmail == "" {
		return fmt.Errorf("kindle email is not configured; run 'kindlebeam config set kindle-email <email>' or use --kindle-email")
	}

	workflow, err := app.NewWorkflow(cfg, logger)
	if err != nil {
		return err
	}

	ctx := context.Background()
	return workflow.SendOnly(ctx, args, options)
}
