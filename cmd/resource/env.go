package resource

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/middleware"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

// envCmd represents the env command
var envCmd = &cobra.Command{
	Use:   "env",
	Short: "View and change the environment for this application",
	Long:  `View your current environment selection and switch between available environments.`,
	PreRunE: middleware.Compose(
		middleware.CheckLogin,
	),
	RunE: func(cobraCmd *cobra.Command, args []string) error {
		return runEnv(cobraCmd)
	},
}

func runEnv(cobraCmd *cobra.Command) error {
	// Get application info from current directory
	appInfo, err := utils.GetApplicationInfo("")
	if err != nil {
		return errors.WrapError("failed to identify application", err)
	}

	// Get the API client
	apiClient := singletons.GetAPIClient()

	// Fetch current environment
	currentEnvResp, err := apiClient.GetApplicationEnvironment(appInfo.ApplicationID)
	if err != nil {
		return errors.WrapError("failed to get current environment", err)
	}

	// Fetch all available environments
	envListResp, err := apiClient.ListApplicationEnvironments(appInfo.ApplicationID)
	if err != nil {
		return errors.WrapError("failed to list environments", err)
	}

	if len(envListResp.Environments) == 0 {
		return &errors.CLIError{
			Title:      "No environments available",
			Suggestion: "Your organization doesn't have any environments configured.",
		}
	}

	// If only one environment, just show current and exit
	if len(envListResp.Environments) == 1 {
		printCurrentEnvironment(cobraCmd, currentEnvResp)
		cobraCmd.Println("\nOnly one environment is available for this application.")
		return nil
	}

	// Let user select a new environment
	selectedEnv, err := selectEnvironment(cobraCmd, envListResp.Environments, currentEnvResp.EnvironmentID)
	if err != nil {
		return errors.WrapError("failed to select environment", err)
	}

	// Set the new environment
	setResp, err := apiClient.SetApplicationEnvironment(appInfo.ApplicationID, selectedEnv.ID)
	if err != nil {
		return errors.WrapError("failed to set environment", err)
	}

	// Print success
	printEnvironmentChanged(cobraCmd, setResp.EnvironmentName)

	return nil
}

// printCurrentEnvironment displays the current environment in a styled box
func printCurrentEnvironment(cobraCmd *cobra.Command, envResp *api.GetApplicationEnvironmentResponse) {
	// Styles
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12")) // Blue

	envNameStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("10")) // Green

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")). // Gray
		Italic(true)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("12")).
		Padding(1, 2).
		MarginTop(1).
		MarginBottom(1)

	// Build content
	title := titleStyle.Render("Current Environment")

	envName := "Not set"
	if envResp.EnvironmentName != nil {
		envName = *envResp.EnvironmentName
	}
	currentEnv := envNameStyle.Render(envName)

	description := descStyle.Render(
		"Switching environments changes where your application\n" +
			"connects to—both locally and in the Major web app.\n" +
			"Each environment can have different resources and\n" +
			"configuration values.",
	)

	content := fmt.Sprintf("%s\n\n%s\n\n%s", title, currentEnv, description)
	box := boxStyle.Render(content)

	cobraCmd.Println(box)
}

// selectEnvironment prompts the user to select an environment from the list
func selectEnvironment(cobraCmd *cobra.Command, envs []api.EnvironmentItem, currentEnvID *string) (*api.EnvironmentItem, error) {
	// Print explanation
	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")). // Gray
		Italic(true).
		MarginBottom(1)

	cobraCmd.Println()
	cobraCmd.Println(descStyle.Render(
		"Switching environments changes where your application connects to—\n" +
			"both locally and in dev mode on the Major web app.",
	))
	cobraCmd.Println()

	// Create options for huh select
	options := make([]huh.Option[string], len(envs))
	for i, env := range envs {
		displayName := env.Name
		if currentEnvID != nil && env.ID == *currentEnvID {
			displayName += " ← current"
		}
		options[i] = huh.NewOption(displayName, env.ID)
	}

	var selectedID string

	// Set initial value to current environment if available
	if currentEnvID != nil {
		selectedID = *currentEnvID
	}

	// Create and run the select form
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select an environment").
				Options(options...).
				Value(&selectedID),
		),
	)

	if err := form.Run(); err != nil {
		return nil, errors.WrapError("failed to get selection", err)
	}

	// Find the selected environment
	for i, env := range envs {
		if env.ID == selectedID {
			return &envs[i], nil
		}
	}

	return nil, fmt.Errorf("selected environment not found")
}

// printEnvironmentChanged displays a success message after switching environments
func printEnvironmentChanged(cobraCmd *cobra.Command, newEnvName string) {
	successStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("10")). // Green
		MarginTop(1)

	envNameStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("14")) // Cyan

	tipStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")). // Gray
		Italic(true).
		MarginTop(1)

	cobraCmd.Println(successStyle.Render("✓ Environment switched successfully!"))
	cobraCmd.Println()
	cobraCmd.Printf("Now using: %s\n", envNameStyle.Render(newEnvName))
	cobraCmd.Println(tipStyle.Render("Run 'major app start' to use the new environment locally."))
}
