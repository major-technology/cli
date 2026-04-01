package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var getEnvCmd = &cobra.Command{
	Use:   "get-env",
	Short: "Print the current CLI environment",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGetEnv(cmd)
	},
}

func runGetEnv(cmd *cobra.Command) error {
	envFile, err := envFilePath()
	if err != nil {
		return err
	}

	data, err := os.ReadFile(envFile)
	if err != nil {
		cmd.Println("prod")
		return nil
	}

	configPath := strings.TrimSpace(string(data))
	for name, path := range validEnvs {
		if path == configPath {
			fmt.Fprintln(cmd.OutOrStdout(), name)
			return nil
		}
	}

	cmd.Println("prod")
	return nil
}
