package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	apiClient "github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/clients/config"
	mjrToken "github.com/major-technology/cli/clients/token"
	"github.com/major-technology/cli/cmd/app"
	"github.com/major-technology/cli/cmd/git"
	"github.com/major-technology/cli/cmd/org"
	"github.com/major-technology/cli/cmd/resource"
	"github.com/major-technology/cli/cmd/user"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

var (
	version    = "dev"                // set by -ldflags
	configFile = "configs/local.json" // can also be set by -ldflags
)

func showLoginPromptIfNeeded(cmd *cobra.Command) bool {
	// Check if user is logged in
	_, err := mjrToken.GetToken()
	if err != nil {
		// User is not logged in, show helpful styled message
		boxStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#87D7FF")).
			Padding(0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#87D7FF"))

		commandStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFD700"))

		message := fmt.Sprintf("👋 Get started by running: %s",
			commandStyle.Render("major user login"))

		cmd.Println(boxStyle.Render(message))
		return false
	}
	return true
}

// checkVersion checks if the CLI version is up to date and handles upgrade prompts
func checkVersion(cmd *cobra.Command) error {
	if version == "dev" {
		return nil
	}

	client := singletons.GetAPIClient()

	resp, err := client.CheckVersion(version)
	if err != nil {
		fmt.Println(err)
		// Silently ignore version check errors to not disrupt user workflow
		return nil
	}

	// Check for force upgrade
	if resp.ForceUpgrade {
		latestVersion := ""
		if resp.LatestVersion != nil {
			latestVersion = *resp.LatestVersion
		}
		return &apiClient.ForceUpgradeError{LatestVersion: latestVersion}
	}

	// Check for optional upgrade
	if resp.CanUpgrade {
		warningStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFD700"))

		commandStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#87D7FF"))

		message := fmt.Sprintf("%s %s",
			warningStyle.Render("There's a new version of major available."),
			fmt.Sprintf("Run %s to get the newest version.",
				commandStyle.Render("major update")))

		cmd.Println(message)
		cmd.Println() // Add a blank line for spacing
	}

	return nil
}

var rootCmd = &cobra.Command{
	Use:     "major",
	Short:   "The major CLI",
	Long:    `The major CLI is a tool to help you create and manage major applications`,
	Version: version,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Check version before every command
		if err := checkVersion(cmd); err != nil {
			// If there's a force upgrade error, handle it and exit
			if !apiClient.CheckErr(cmd, err) {
				os.Exit(1)
			}
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		if ok := showLoginPromptIfNeeded(cmd); ok {
			cmd.Help()
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Disable the completion command
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// Disable the help command (use -h flag instead)
	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})

	// Set custom help function to show login prompt after help
	defaultHelpFunc := rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		defaultHelpFunc(cmd, args)
		if cmd == rootCmd {
			showLoginPromptIfNeeded(cmd)
		}
	})

	// Register subcommands
	rootCmd.AddCommand(user.Cmd)
	rootCmd.AddCommand(org.Cmd)
	rootCmd.AddCommand(app.Cmd)
	rootCmd.AddCommand(resource.Cmd)
	rootCmd.AddCommand(git.GitCmd)
}

func initConfig() {
	var err error
	cfg, err := config.Load(configFile)
	cobra.CheckErr(err)

	// Set config in singletons package
	singletons.SetConfig(cfg)

	// Initialize API client with base URL (token will be fetched automatically per-request)
	client := apiClient.NewClient(cfg.APIURL)
	singletons.SetAPIClient(client)
}
