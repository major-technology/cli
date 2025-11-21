package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/clients/config"
	mjrToken "github.com/major-technology/cli/clients/token"
	"github.com/major-technology/cli/cmd/app"
	"github.com/major-technology/cli/cmd/git"
	"github.com/major-technology/cli/cmd/org"
	"github.com/major-technology/cli/cmd/resource"
	"github.com/major-technology/cli/cmd/user"
	clierrors "github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/middleware"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

var (
	Version    = "dev"                // set by -ldflags, exported for middleware
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

var rootCmd = &cobra.Command{
	Use:               "major",
	Short:             "The major CLI",
	Long:              `The major CLI is a tool to help you create and manage major applications`,
	Version:           Version,
	SilenceErrors:     true, // We handle errors centrally
	SilenceUsage:      true, // Don't show usage on errors
	PersistentPreRunE: middleware.Compose(middleware.CheckVersion(Version)),
	Run: func(cmd *cobra.Command, args []string) {
		if ok := showLoginPromptIfNeeded(cmd); ok {
			cmd.Help()
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		clierrors.PrintError(rootCmd, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Disable the default completion command (we use our own)
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
	client := api.NewClient(cfg.APIURL)
	singletons.SetAPIClient(client)
}
