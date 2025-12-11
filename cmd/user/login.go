package user

import (
	"errors"
	"fmt"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	apiClient "github.com/major-technology/cli/clients/api"
	mjrToken "github.com/major-technology/cli/clients/token"
	clierrors "github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to the major app",
	Long:  `Login and stores your session token`,
	RunE: func(cobraCmd *cobra.Command, args []string) error {
		return runLogin(cobraCmd)
	},
}

func runLogin(cobraCmd *cobra.Command) error {
	// Get the API client (no token yet for login flow)
	apiClient := singletons.GetAPIClient()
	startResp, err := apiClient.StartLogin()
	if err != nil {
		return clierrors.WrapError("failed to start login", err)
	}

	if err := utils.OpenBrowser(startResp.VerificationURI); err != nil {
		// ignore, failed to open browser
	}
	cobraCmd.Println("Attempting to automatically open the SSO authorization page in your default browser.")
	cobraCmd.Printf("If the browser does not open or you wish to use a different device to authorize this request, open the following URL:\n\n%s\n", startResp.VerificationURI)

	token, err := pollForToken(cobraCmd, apiClient, startResp.DeviceCode, startResp.Interval, startResp.ExpiresIn)
	if err != nil {
		return clierrors.WrapError("authentication failed", err)
	}

	if err := mjrToken.StoreToken(token); err != nil {
		return clierrors.WrapError("failed to store token", err)
	}

	// Fetch organizations (token will be fetched automatically)
	orgsResp, err := apiClient.GetOrganizations()
	if err != nil {
		return clierrors.WrapError("failed to fetch organizations", err)
	}

	// Let user select default organization
	if len(orgsResp.Organizations) > 0 {
		selectedOrg, err := SelectOrganization(cobraCmd, orgsResp.Organizations)
		if err != nil {
			return clierrors.WrapError("failed to select organization", err)
		}

		if err := mjrToken.StoreDefaultOrg(selectedOrg.ID, selectedOrg.Name); err != nil {
			return clierrors.WrapError("failed to store default organization", err)
		}

		cobraCmd.Printf("Default organization set to: %s\n", selectedOrg.Name)
	} else {
		return clierrors.ErrorNoOrganizationsAvailable
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
				// Check if authorization is still pending - this is an expected state
				if errors.Is(err, clierrors.ErrorAuthorizationPending) {
					cobraCmd.Print(".")
					continue
				}
				// Any other error is unexpected
				return "", clierrors.WrapError("failed to poll", err)
			}

			// Success - got the token
			if pollResp.AccessToken != "" {
				cobraCmd.Println() // New line after the dots
				return pollResp.AccessToken, nil
			}

			// Unexpected response
			return "", errors.New("unexpected response - no access token received")
		}
	}
}

// SelectOrganization prompts the user to select an organization from the list
func SelectOrganization(cobraCmd *cobra.Command, orgs []apiClient.Organization) (*apiClient.Organization, error) {
	if len(orgs) == 0 {
		return nil, clierrors.ErrorNoOrganizationsAvailable
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
		return nil, clierrors.WrapError("failed to get selection", err)
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

	// Highlighted style for the recommended demo command
	demoCommandStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("10")). // Green
		Bold(true)

	recommendedBadge := lipgloss.NewStyle().
		Foreground(lipgloss.Color("0")).  // Black text
		Background(lipgloss.Color("10")). // Green background
		Bold(true).
		Padding(0, 1)

	demoDescStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("7")). // Brighter gray/white
		MarginLeft(2)

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

	demoCommand := demoCommandStyle.Render("major demo create") + " " + recommendedBadge.Render("Recommended")
	demoDesc := demoDescStyle.Render("Create a demo app connected to a database to play around with")

	cloneCommand := commandStyle.Render("major app clone")
	cloneDesc := descriptionStyle.Render("Clone an existing app from GitHub")

	createCommand := commandStyle.Render("major app create")
	createDesc := descriptionStyle.Render("Create a brand new app from a template")

	content := fmt.Sprintf("%s\n\n%s\n%s\n\n%s\n%s\n\n%s\n%s",
		nextStepsTitle,
		demoCommand,
		demoDesc,
		cloneCommand,
		cloneDesc,
		createCommand,
		createDesc,
	)

	box := boxStyle.Render(content)

	// Print everything
	cobraCmd.Println(successMsg)
	cobraCmd.Println(box)
}
