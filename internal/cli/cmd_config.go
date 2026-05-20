package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/bavanchun/Typeburn/internal/cli/output"
	"github.com/bavanchun/Typeburn/internal/config"
	"github.com/bavanchun/Typeburn/internal/theme"
)

func newConfigCmd(e env) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:           "config",
		Short:         "Read or update settings",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	cmd.PersistentFlags().BoolVar(&asJSON, "json", false, "print config as JSON")
	cmd.AddCommand(configListCmd(e, &asJSON))
	cmd.AddCommand(configGetCmd(e))
	cmd.AddCommand(configSetCmd(e))
	return cmd
}

func configListCmd(e env, asJSON *bool) *cobra.Command {
	return &cobra.Command{
		Use:           "list",
		Short:         "List all settings",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			s := e.loadSettings()
			if *asJSON {
				return output.RenderJSON(cmd.OutOrStdout(), s)
			}
			return output.RenderTable(cmd.OutOrStdout(), []string{"key", "value"}, configRows(s))
		},
	}
}

func configGetCmd(e env) *cobra.Command {
	return &cobra.Command{
		Use:           "get <key>",
		Short:         "Print a setting value",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			value, ok := configGet(e.loadSettings(), args[0])
			if !ok {
				return usageError("unknown config key %q", args[0])
			}
			fmt.Fprintln(cmd.OutOrStdout(), value)
			return nil
		},
	}
}

func configSetCmd(e env) *cobra.Command {
	return &cobra.Command{
		Use:           "set <key> <value>",
		Short:         "Persist a setting value",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			s := e.loadSettings()
			if err := configSet(&s, args[0], args[1]); err != nil {
				return err
			}
			s.Normalize()
			if err := e.saveSettings(s); err != nil {
				return ioError("could not save settings: %w", err)
			}
			return nil
		},
	}
}

func configRows(s config.Settings) [][]string {
	return [][]string{
		{"theme", s.Theme},
		{"default_mode", string(s.DefaultMode)},
		{"default_length", strconv.Itoa(s.DefaultLength)},
		{"blink_cursor", strconv.FormatBool(s.BlinkCursor)},
	}
}

func configGet(s config.Settings, key string) (string, bool) {
	for _, row := range configRows(s) {
		if row[0] == key {
			return row[1], true
		}
	}
	return "", false
}

func configSet(s *config.Settings, key, value string) error {
	switch key {
	case "theme":
		if !containsString(theme.Names(), value) {
			return usageError("invalid theme %q (valid: %s)", value, strings.Join(theme.Names(), ", "))
		}
		s.Theme = value
	case "default_mode":
		mode := config.Mode(value)
		if !validMode(mode) {
			return usageError("invalid default_mode %q (valid: time, words, quote, code)", value)
		}
		s.DefaultMode = mode
	case "default_length":
		n, err := strconv.Atoi(value)
		if err != nil || n < 0 {
			return usageError("default_length must be a non-negative integer")
		}
		if lens := config.LengthsFor(s.DefaultMode); lens != nil && !containsInt(lens, n) {
			return usageError("invalid default_length %d for mode %s", n, s.DefaultMode)
		}
		s.DefaultLength = n
	case "blink_cursor":
		v, ok := parseBool(value)
		if !ok {
			return usageError("blink_cursor must be true, false, 1, or 0")
		}
		s.BlinkCursor = v
	default:
		return usageError("unknown config key %q", key)
	}
	return nil
}

func parseBool(value string) (bool, bool) {
	switch value {
	case "true", "1":
		return true, true
	case "false", "0":
		return false, true
	default:
		return false, false
	}
}
