/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package user

import (
	"github.com/spf13/cobra"
)

// Cmd represents the user command
var Cmd = &cobra.Command{
	Use:   "user",
	Short: "User management commands",
	Long:  `Commands for managing user authentication and profile.`,
}

func init() {
	// Add user subcommands
	Cmd.AddCommand(loginCmd)
	Cmd.AddCommand(logoutCmd)
	Cmd.AddCommand(whoamiCmd)
}
