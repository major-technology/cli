/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package user

import (
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

// Cmd represents the user command
var Cmd = &cobra.Command{
	Use:   "user",
	Short: "User management commands",
	Long:  `Commands for managing user authentication and profile.`,
	Args:  utils.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	// Add user subcommands
	Cmd.AddCommand(loginCmd)
	Cmd.AddCommand(logoutCmd)
	Cmd.AddCommand(whoamiCmd)
	Cmd.AddCommand(gitconfigCmd)
}
