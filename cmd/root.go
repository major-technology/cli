package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/clients/config"
	mjrToken "github.com/major-technology/cli/clients/token"
	"github.com/major-technology/cli/cmd/agent"
	"github.com/major-technology/cli/cmd/app"
	cliconfig "github.com/major-technology/cli/cmd/config"
	"github.com/major-technology/cli/cmd/demo"
	"github.com/major-technology/cli/cmd/mcp"
	"github.com/major-technology/cli/cmd/org"
	"github.com/major-technology/cli/cmd/project"
	"github.com/major-technology/cli/cmd/resource"
	"github.com/major-technology/cli/cmd/user"
	"github.com/major-technology/cli/cmd/vars"
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
	PersistentPreRunE: rootPersistentPreRunE,
	Run: func(cmd *cobra.Command, args []string) {
		if ok := showLoginPromptIfNeeded(cmd); ok {
			cmd.Help()
		}
	},
}

func rootPersistentPreRunE(cmd *cobra.Command, args []string) error {
	return middleware.Compose(
		rejectInjectedAuthManagement,
		middleware.ApplyNonInteractive,
		middleware.CheckVersion(Version),
	)(cmd, args)
}

func rejectInjectedAuthManagement(cmd *cobra.Command, args []string) error {
	if !mjrToken.HasInjectedToken() {
		return nil
	}
	parent := cmd.Parent()
	if parent == nil || parent.Name() != "user" {
		return nil
	}
	switch cmd.Name() {
	case "login", "logout", "token":
		return fmt.Errorf("credential is externally managed")
	default:
		return nil
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		clierrors.PrintError(rootCmd, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().Bool("non-interactive", false, "Never prompt or open a browser. GIT_SSH_COMMAND must be a single direct ssh invocation; wrappers and compound commands are rejected.")

	// Disable the default completion command (we use our own)
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// Disable the help command (use -h flag instead)
	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})

	// Register subcommands
	rootCmd.AddGroup(&cobra.Group{ID: "main", Title: "Main Commands"})
	rootCmd.AddGroup(&cobra.Group{ID: "config", Title: "Configurations"})

	user.Cmd.GroupID = "config"
	rootCmd.AddCommand(user.Cmd)

	org.Cmd.GroupID = "config"
	rootCmd.AddCommand(org.Cmd)

	app.Cmd.GroupID = "main"
	rootCmd.AddCommand(app.Cmd)

	agent.Cmd.GroupID = "main"
	rootCmd.AddCommand(agent.Cmd)
	for _, command := range agent.TargetCommands() {
		command.GroupID = "main"
		rootCmd.AddCommand(command)
	}

	rootCmd.AddCommand(demo.Cmd)

	resource.Cmd.GroupID = "main"
	rootCmd.AddCommand(resource.Cmd)

	vars.Cmd.GroupID = "main"
	rootCmd.AddCommand(vars.Cmd)

	project.Cmd.GroupID = "main"
	rootCmd.AddCommand(project.Cmd)

	rootCmd.AddCommand(mcp.Cmd)

	rootCmd.AddCommand(cliconfig.Cmd)
}

func initConfig() {
	// Check for persistent environment override
	if homeDir, err := os.UserHomeDir(); err == nil {
		if data, err := os.ReadFile(filepath.Join(homeDir, ".major", "env")); err == nil {
			if override := strings.TrimSpace(string(data)); override != "" {
				configFile = override
			}
		}
	}

	var err error
	cfg, err := config.Load(configFile)
	cobra.CheckErr(err)

	// Set config in singletons package
	singletons.SetConfig(cfg)

	// Initialize API client with base URL (token will be fetched automatically per-request)
	client := api.NewClient(cfg.APIURL)
	singletons.SetAPIClient(client)
}
