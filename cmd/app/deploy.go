package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/major-technology/cli/clients/git"
	"github.com/major-technology/cli/clients/token"
	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/singletons"
	pkgErrors "github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// deployCmd represents the deploy command
var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy a new version of the application",
	Long:  `Creates a new version by committing and pushing changes, then deploying to the platform.`,
	RunE: func(cobraCmd *cobra.Command, args []string) error {
		return runDeploy(cobraCmd)
	},
}

func runDeploy(cobraCmd *cobra.Command) error {
	// Check if we're in a git repository
	if !git.IsGitRepository() {
		return errors.ErrorNotInGitRepository
	}

	// Get application ID
	applicationID, err := getApplicationID()
	if err != nil {
		return errors.WrapError("failed to get application ID", err)
	}

	// Check for uncommitted changes
	hasChanges, err := git.HasUncommittedChanges()
	if err != nil {
		return errors.WrapError("failed to check for uncommitted changes: %w", err)
	}

	if hasChanges {
		cobraCmd.Println("📝 Uncommitted changes detected")

		// Prompt for commit message
		var commitMessage string
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewText().
					Title("Commit Message").
					Description("Enter a commit message for your changes").
					Value(&commitMessage).
					Validate(func(s string) error {
						if strings.TrimSpace(s) == "" {
							return fmt.Errorf("commit message is required")
						}
						return nil
					}),
			),
		)

		if err := form.Run(); err != nil {
			return errors.WrapError("failed to collect commit message", err)
		}

		// Stage all changes
		if err := git.Add(); err != nil {
			return errors.WrapError("failed to stage changes", err)
		}
		cobraCmd.Println("✓ Changes staged")

		// Commit changes
		if err := git.Commit(commitMessage); err != nil {
			return errors.WrapError("failed to commit changes", err)
		}
		cobraCmd.Println("✓ Changes committed")

		// Push to remote
		if err := git.PushToMain(); err != nil {
			return errors.WrapError("failed to push changes", err)
		}
		cobraCmd.Println("✓ Changes pushed to remote")
	} else {
		cobraCmd.Println("✓ No uncommitted changes")
	}

	// Get organization ID
	organizationID, _, err := token.GetDefaultOrg()
	if err != nil {
		return errors.WrapError("failed to get default organization", errors.ErrorNoOrganizationSelected)
	}

	// Call API to create new version
	apiClient := singletons.GetAPIClient()
	resp, err := apiClient.CreateApplicationVersion(applicationID)
	if err != nil {
		return err
	}

	cobraCmd.Printf("\n✓ Version created: %s\n", resp.VersionID)

	// Poll deployment status with beautiful UI
	finalStatus, deploymentError, appURL, err := pollDeploymentStatus(applicationID, organizationID, resp.VersionID)
	if err != nil {
		return errors.WrapError("failed to track deployment status", err)
	}

	// Print final status
	if finalStatus == "DEPLOYED" {
		cobraCmd.Printf("\n🎉 Deployment successful!\n")

		// Print application URL from the API response
		if appURL != "" {
			cobraCmd.Printf("\n🌐 Your application is live at:\n")
			cobraCmd.Printf("  %s\n", appURL)
		}
	} else {
		// Display error message if available
		if deploymentError != "" {
			cobraCmd.Printf("\n❌ Deployment failed with status: %s\n", finalStatus)
			cobraCmd.Printf("\n%s\n", formatDeploymentError(deploymentError))
		}
		return pkgErrors.Errorf("deployment failed with status: %s", finalStatus)
	}

	return nil
}

// deploymentStatusModel represents the Bubble Tea model for deployment status tracking
type deploymentStatusModel struct {
	applicationID   string
	organizationID  string
	versionID       string
	spinner         spinner.Model
	status          string
	deploymentError string
	appURL          string
	err             error
	done            bool
	dots            int  // Track number of dots (0-4)
	dotsIncreasing  bool // Track if dots are increasing or decreasing
	tickCounter     int  // Counter to slow down dot animation
}

type statusMsg struct {
	status          string
	deploymentError string
	appURL          string
	err             error
}

type tickMsg time.Time

func (m deploymentStatusModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		pollStatus(m.applicationID, m.organizationID, m.versionID),
	)
}

func (m deploymentStatusModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.done = true
			return m, tea.Quit
		}

	case statusMsg:
		m.status = msg.status
		m.deploymentError = msg.deploymentError
		m.appURL = msg.appURL
		m.err = msg.err

		// Check if we're in a terminal state
		if isTerminalStatus(m.status) || m.err != nil {
			m.done = true
			return m, tea.Quit
		}

		// Wait 2 seconds before polling again
		return m, tickCmd()

	case tickMsg:
		// Time to poll for status update
		return m, pollStatus(m.applicationID, m.organizationID, m.versionID)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)

		// Update dots animation (slower than spinner)
		// Only update every 5th spinner tick to slow down the animation
		m.tickCounter++
		if m.tickCounter >= 5 {
			m.tickCounter = 0
			if m.dotsIncreasing {
				m.dots++
				if m.dots >= 4 {
					m.dotsIncreasing = false
				}
			} else {
				m.dots--
				if m.dots <= 0 {
					m.dotsIncreasing = true
				}
			}
		}

		return m, cmd
	}

	return m, nil
}

func (m deploymentStatusModel) View() string {
	if m.err != nil {
		return ""
	}

	if m.done {
		return ""
	}

	// Style definitions
	spinnerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	statusStyle := lipgloss.NewStyle().Bold(true)

	// Get status display text and color
	statusText, statusColor := getStatusDisplay(m.status)

	// Add animated dots
	dots := strings.Repeat(".", m.dots)
	statusTextWithDots := statusText + dots

	coloredStatus := lipgloss.NewStyle().
		Foreground(lipgloss.Color(statusColor)).
		Bold(true).
		Render(statusTextWithDots)

	return fmt.Sprintf("\n%s %s %s\n",
		spinnerStyle.Render(m.spinner.View()),
		statusStyle.Render("Status:"),
		coloredStatus,
	)
}

func pollStatus(applicationID, organizationID, versionID string) tea.Cmd {
	return func() tea.Msg {
		apiClient := singletons.GetAPIClient()
		resp, err := apiClient.GetVersionStatus(applicationID, organizationID, versionID)
		if err != nil {
			return statusMsg{err: err}
		}
		return statusMsg{
			status:          resp.Status,
			deploymentError: resp.DeploymentError,
			appURL:          resp.AppURL,
		}
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func isTerminalStatus(status string) bool {
	terminalStatuses := []string{
		"BUNDLE_FAILED",
		"BUILD_FAILED",
		"DEPLOY_FAILED",
		"DEPLOYED",
	}
	for _, s := range terminalStatuses {
		if status == s {
			return true
		}
	}
	return false
}

func getStatusDisplay(status string) (string, string) {
	switch status {
	case "BUNDLING":
		return "Bundling application", "214" // Orange
	case "BUNDLE_FAILED":
		return "Bundle failed ✗", "196" // Red
	case "BUILDING":
		return "Building application", "220" // Yellow
	case "BUILD_FAILED":
		return "Build failed ✗", "196" // Red
	case "DEPLOYING":
		return "Deploying application", "117" // Light blue
	case "DEPLOY_FAILED":
		return "Deployment failed ✗", "196" // Red
	case "DEPLOYED":
		return "Deployed successfully ✓", "46" // Green
	default:
		return "Processing", "245" // Gray
	}
}

func pollDeploymentStatus(applicationID, organizationID, versionID string) (string, string, string, error) {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	m := deploymentStatusModel{
		applicationID:   applicationID,
		organizationID:  organizationID,
		versionID:       versionID,
		spinner:         s,
		status:          "",
		deploymentError: "",
		appURL:          "",
		done:            false,
		dots:            1,
		dotsIncreasing:  true,
		tickCounter:     0,
	}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return "", "", "", err
	}

	finalStatusModel := finalModel.(deploymentStatusModel)
	if finalStatusModel.err != nil {
		return "", "", "", finalStatusModel.err
	}

	return finalStatusModel.status, finalStatusModel.deploymentError, finalStatusModel.appURL, nil
}

// formatDeploymentError formats the deployment error message with nice styling
func formatDeploymentError(errorMsg string) string {
	// Style definitions
	errorBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")).
		Padding(1, 2).
		Width(80)

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)

	messageStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	// Format the error message
	title := titleStyle.Render("Deployment Error Details:")
	message := messageStyle.Render(errorMsg)

	content := fmt.Sprintf("%s\n\n%s", title, message)
	return errorBoxStyle.Render(content)
}
