package kindlebeam

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jamesonstone/kindlebeam/internal/config"
)

func newConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "manage kindlebeam configuration",
	}

	cmd.AddCommand(newConfigShowCommand())
	cmd.AddCommand(newConfigSetCommand())

	return cmd
}

func newConfigShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "show effective configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, path, err := config.Load()
			if err != nil {
				return err
			}

			data, err := json.MarshalIndent(cfg, "", "  ")
			if err != nil {
				return err
			}

			fmt.Fprintf(os.Stdout, "config file: %s\n%s\n", path, string(data))
			return nil
		},
	}
}

func newConfigSetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set",
		Short: "set configuration values",
		RunE:  runConfigSet,
	}

	return cmd
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: kindlebeam config set <key> <value>")
	}

	key := args[0]
	value := args[1]

	cfg, _, err := config.Load()
	if err != nil {
		return err
	}

	switch key {
	case "kindle-email":
		cfg.KindleEmail = value
	case "default-output-format":
		cfg.DefaultOutputFormat = value
	case "mail-command":
		cfg.MailCommand = value
	case "pandoc-path":
		cfg.PandocPath = value
	default:
		return fmt.Errorf("unknown config key %q", key)
	}

	if err := config.Save(cfg); err != nil {
		return err
	}

	fmt.Fprintln(os.Stdout, "✅ config updated")
	return nil
}
