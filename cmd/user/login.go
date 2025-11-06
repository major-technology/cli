package user

import (
	"fmt"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	apiClient "github.com/major-technology/cli/clients/api"
	mjrToken "github.com/major-technology/cli/clients/token"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to the major app",
	Long:  `Login and stores your session token`,
	Run: func(cobraCmd *cobra.Command, args []string) {
		cobra.CheckErr(runLogin(cobraCmd))
	},
}

func runLogin(cobraCmd *cobra.Command) error {
	// Get the API client (no token yet for login flow)
	apiClient := singletons.GetAPIClient()
	startResp, err := apiClient.StartLogin()
	if err != nil {
		return fmt.Errorf("failed to start login: %w", err)
	}

	if err := utils.OpenBrowser(startResp.VerificationURI); err != nil {
		// ignore, failed to open browser
	}
	cobraCmd.Println("Attempting to automatically open the SSO authorization page in your default browser.")
	cobraCmd.Printf("If the browser does not open or you wish to use a different device to authorize this request, open the following URL:\n\n%s\n", startResp.VerificationURI)

	token, err := pollForToken(cobraCmd, apiClient, startResp.DeviceCode, startResp.Interval, startResp.ExpiresIn)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	if err := mjrToken.StoreToken(token); err != nil {
		return fmt.Errorf("failed to store token: %w", err)
	}

	// Fetch organizations (token will be fetched automatically)
	orgsResp, err := apiClient.GetOrganizations()
	if err != nil {
		return fmt.Errorf("failed to fetch organizations: %w", err)
	}

	// Let user select default organization
	if len(orgsResp.Organizations) > 0 {
		selectedOrg, err := SelectOrganization(cobraCmd, orgsResp.Organizations)
		if err != nil {
			return fmt.Errorf("failed to select organization: %w", err)
		}

		if err := mjrToken.StoreDefaultOrg(selectedOrg.ID, selectedOrg.Name); err != nil {
			return fmt.Errorf("failed to store default organization: %w", err)
		}

		cobraCmd.Printf("Default organization set to: %s\n", selectedOrg.Name)
	}

	printSuccessMessage(cobraCmd)
	return nil
}

// pollForToken polls POST /cli/login/poll until authenticated or timeout
func pollForToken(cobraCmd *cobra.Command, client *apiClient.Client, deviceCode string, interval int, expiresIn int) (string, error) {
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()
	timeoutChan := time.After(time.Duration(expiresIn) * time.Second)

	for {
		select {
		case <-timeoutChan:
			return "", fmt.Errorf("authentication timeout - code expired")
		case <-ticker.C:
			pollResp, err := client.PollLogin(deviceCode)
			if err != nil {
				return "", fmt.Errorf("failed to poll: %w", err)
			}

			// Check if still pending
			if pollResp.Error == "authorization_pending" {
				cobraCmd.Print(".")
				continue
			}

			// Success - got the token
			if pollResp.AccessToken != "" {
				cobraCmd.Println() // New line after the dots
				return pollResp.AccessToken, nil
			}

			// Unexpected response
			return "", fmt.Errorf("unexpected response - no access token received")
		}
	}
}

// SelectOrganization prompts the user to select an organization from the list
func SelectOrganization(cobraCmd *cobra.Command, orgs []apiClient.Organization) (*apiClient.Organization, error) {
	if len(orgs) == 0 {
		return nil, fmt.Errorf("no organizations available")
	}

	// If only one organization, automatically select it
	if len(orgs) == 1 {
		cobraCmd.Printf("Only one organization available. Automatically selecting it.\n")
		return &orgs[0], nil
	}

	// Create options for huh select
	options := make([]huh.Option[string], len(orgs))
	for i, org := range orgs {
		options[i] = huh.NewOption(org.Name, org.ID)
	}

	var selectedID string

	// Create and run the select form
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select a default organization").
				Options(options...).
				Value(&selectedID),
		),
	)

	err := form.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to get selection: %w", err)
	}

	// Find the selected organization
	for i, org := range orgs {
		if org.ID == selectedID {
			return &orgs[i], nil
		}
	}

	return nil, fmt.Errorf("selected organization not found")
}

// printSuccessMessage displays a nicely formatted success message with next steps
func printSuccessMessage(cobraCmd *cobra.Command) {
	// Define styles
	successStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("10")). // Green
		MarginTop(1).
		MarginBottom(1)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12")) // Blue

	commandStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("14")). // Cyan
		Bold(true)

	descriptionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")). // Gray
		MarginLeft(2)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("12")). // Blue
		Padding(1, 2).
		MarginTop(1).
		MarginBottom(1)

	// Build the message
	successMsg := successStyle.Render("✓ Successfully authenticated!")

	nextStepsTitle := titleStyle.Render("What's next?")

	pullCommand := commandStyle.Render("major app pull")
	pullDesc := descriptionStyle.Render("Pull an existing app from GitHub")

	createCommand := commandStyle.Render("major app create")
	createDesc := descriptionStyle.Render("Create a brand new app from a template")

	content := fmt.Sprintf("%s\n\n%s\n%s\n\n%s\n%s",
		nextStepsTitle,
		pullCommand,
		pullDesc,
		createCommand,
		createDesc,
	)

	box := boxStyle.Render(content)

	// Print everything
	cobraCmd.Println(successMsg)
	cobraCmd.Println(box)
}
