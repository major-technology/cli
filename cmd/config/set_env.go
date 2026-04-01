package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var validEnvs = map[string]string{
	"local":   "configs/local.json",
	"staging": "configs/staging.json",
	"prod":    "configs/prod.json",
}

var setEnvCmd = &cobra.Command{
	Use:   "set-env <local|staging|prod>",
	Short: "Set the CLI environment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSetEnv(cmd, args[0])
	},
}

func runSetEnv(cmd *cobra.Command, env string) error {
	configPath, ok := validEnvs[env]
	if !ok {
		return fmt.Errorf("invalid environment %q, must be one of: local, staging, prod", env)
	}

	envFilePath, err := envFilePath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(envFilePath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(envFilePath, []byte(configPath), 0644); err != nil {
		return fmt.Errorf("failed to write environment file: %w", err)
	}

	cmd.Printf("Environment set to %s\n", env)
	return nil
}

func envFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".major", "env"), nil
}
