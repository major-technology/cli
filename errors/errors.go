package errors

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// CLIError represents a user-facing CLI error with a title, suggestion, and underlying error
type CLIError struct {
	Title      string
	Suggestion string
	Err        error
}

// Standard error interface
func (e *CLIError) Error() string {
	return e.Title
}

// Unwrap allows standard errors.Is/As to work on this struct
func (e *CLIError) Unwrap() error {
	return e.Err
}

// WrapError wraps an existing error with additional context using Standard Lib
func WrapError(msg string, ogerr error) *CLIError {
	var cliError *CLIError

	if errors.As(ogerr, &cliError) {
		return &CLIError{
			Title:      msg,
			Suggestion: cliError.Suggestion,
			Err:        fmt.Errorf("%s: %w", msg, cliError.Err),
		}
	}

	return &CLIError{
		Title: msg,
		Err:   fmt.Errorf("%s: %w", msg, ogerr),
	}
}

func PrintError(cmd *cobra.Command, err error) {
	errorStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF5F87")).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FF5F87"))

	commandStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#87D7FF"))

	var title, suggestion string

	// standard errors.As works perfectly with your custom struct
	var cliErr *CLIError
	if errors.As(err, &cliErr) {
		title = cliErr.Title
		suggestion = cliErr.Suggestion
	} else {
		title = err.Error()
	}

	var message string
	if suggestion != "" {
		message = fmt.Sprintf("%s\n\n%s", title, commandStyle.Render(suggestion))
	} else {
		message = title
	}

	cmd.Println(errorStyle.Render(message))
}

// Authentication/Session Errors
var ErrorNotLoggedIn = &CLIError{
	Title:      "Not logged in!",
	Suggestion: "Run 'major user login' to get started.",
	Err:        errors.New("user not logged in"),
}

var ErrorSessionExpired = &CLIError{
	Title:      "Your session has expired!",
	Suggestion: "Run 'major user login' to login again.",
	Err:        errors.New("session expired"),
}

var ErrorAuthenticationFailed = &CLIError{
	Title:      "Authentication failed",
	Suggestion: "Run 'major user login' to authenticate",
	Err:        errors.New("authentication failed"),
}

// Dependency/Tool Errors
var ErrorPnpmNotFound = &CLIError{
	Title:      "pnpm not found",
	Suggestion: "pnpm is required. Install it with: brew install pnpm\nOr if you have Node.js: corepack enable",
	Err:        errors.New("pnpm not found in PATH"),
}

var ErrorNodeNotFound = &CLIError{
	Title:      "Node.js not found",
	Suggestion: "Node.js is required. Please install it from https://nodejs.org",
	Err:        errors.New("node not found in PATH"),
}

func ErrorNodeVersionTooOld(required, current string) *CLIError {
	return &CLIError{
		Title:      fmt.Sprintf("Node.js version %s or higher required", required),
		Suggestion: fmt.Sprintf("You are running version %s. Please install nvm and run: nvm use 22", current),
		Err:        fmt.Errorf("node version %s is required, but found %s", required, current),
	}
}

// Git Errors
var ErrorGitNotFound = &CLIError{
	Title:      "Git not found",
	Suggestion: "Git is required. Please install it from https://git-scm.com",
	Err:        errors.New("git not found in PATH"),
}

var ErrorNotGitRepository = &CLIError{
	Title:      "Not a major repository",
	Suggestion: "Make sure you're inside a major project directory.",
	Err:        errors.New("not a git repository"),
}

var ErrorUnsupportedGitRemoteURL = &CLIError{
	Title:      "Unsupported git remote URL format",
	Suggestion: "Only GitHub SSH (git@github.com:owner/repo.git) and HTTPS (https://github.com/owner/repo.git) URLs are supported.",
	Err:        errors.New("unsupported git remote URL format"),
}

func ErrorUnsupportedGitRemoteURLWithFormat(url string) *CLIError {
	return &CLIError{
		Title:      "Unsupported git remote URL format",
		Suggestion: fmt.Sprintf("Only GitHub SSH (git@github.com:owner/repo.git) and HTTPS (https://github.com/owner/repo.git) URLs are supported.\n\nReceived: %s", url),
		Err:        fmt.Errorf("unsupported git remote URL format: %s", url),
	}
}

// Configuration Errors
var ErrorConfigNotFound = &CLIError{
	Title:      "Configuration not found",
	Suggestion: "Unable to load CLI configuration. Please reinstall the CLI.",
	Err:        errors.New("configuration file not found"),
}

var ErrorInvalidConfig = &CLIError{
	Title:      "Invalid configuration",
	Suggestion: "CLI configuration is invalid. Please reinstall the CLI.",
	Err:        errors.New("invalid configuration format"),
}

// Resource Errors
var ErrorNoResourcesAvailable = &CLIError{
	Title:      "No resources available",
	Suggestion: "No resources are available for your organization.",
	Err:        errors.New("no resources available"),
}

var ErrorResourceNotFound = &CLIError{
	Title:      "Resource not found",
	Suggestion: "The requested resource does not exist.",
	Err:        errors.New("resource not found"),
}

// Application Errors
var ErrorApplicationNotFound = &CLIError{
	Title:      "Application not found",
	Suggestion: "The application does not exist or you don't have access to it.",
	Err:        errors.New("application not found"),
}

var ErrorNoApplicationContext = &CLIError{
	Title:      "No application context",
	Suggestion: "Run this command from within an application directory, or specify an application.",
	Err:        errors.New("no application context"),
}

// Organization Errors
var ErrorNoOrganizationSelected = &CLIError{
	Title:      "No organization selected",
	Suggestion: "Run 'major org select' to choose an organization.",
	Err:        errors.New("no organization selected"),
}

var ErrorOrganizationNotFound = &CLIError{
	Title:      "Organization not found",
	Suggestion: "The organization does not exist or you don't have access to it.",
	Err:        errors.New("organization not found"),
}

// Network/API Errors
var ErrorNetworkFailure = &CLIError{
	Title:      "Network error",
	Suggestion: "Unable to connect to Major services. Please check your internet connection.",
	Err:        errors.New("network failure"),
}

var ErrorAPIUnavailable = &CLIError{
	Title:      "Major API unavailable",
	Suggestion: "Major services are currently unavailable. Please try again later.",
	Err:        errors.New("API unavailable"),
}

// General Errors
var ErrorInvalidInput = &CLIError{
	Title:      "Invalid input",
	Suggestion: "Please check your input and try again.",
	Err:        errors.New("invalid input"),
}

var ErrorOperationCancelled = &CLIError{
	Title:      "Operation cancelled",
	Suggestion: "",
	Err:        errors.New("operation cancelled by user"),
}

// API Error Codes - Authentication & Authorization (2000-2099)
var ErrorUnauthorized = &CLIError{
	Title:      "Unauthorized",
	Suggestion: "You don't have permission to perform this action. Try running 'major user login' again.",
	Err:        errors.New("unauthorized"),
}

var ErrorInvalidToken = &CLIError{
	Title:      "Your session has expired!",
	Suggestion: "Run 'major user login' to login again.",
	Err:        errors.New("invalid or expired token"),
}

var ErrorInvalidUserCode = &CLIError{
	Title:      "Invalid user code",
	Suggestion: "The user code you entered is invalid. Please try logging in again.",
	Err:        errors.New("invalid user code"),
}

var ErrorTokenNotFound = &CLIError{
	Title:      "Not logged in!",
	Suggestion: "Run 'major user login' to get started.",
	Err:        errors.New("no authentication token found"),
}

var ErrorInvalidDeviceCode = &CLIError{
	Title:      "Invalid device code",
	Suggestion: "The device code is invalid. Please try logging in again.",
	Err:        errors.New("invalid device code"),
}

var ErrorAuthorizationPending = &CLIError{
	Title:      "Authorization pending",
	Suggestion: "Please complete the login process in your browser.",
	Err:        errors.New("authorization pending"),
}

// API Error Codes - Organization (3000-3099)
var ErrorOrganizationNotFoundAPI = &CLIError{
	Title:      "Organization not found",
	Suggestion: "The organization does not exist or you don't have access to it.",
	Err:        errors.New("organization not found"),
}

var ErrorNotOrgMember = &CLIError{
	Title:      "Not an organization member",
	Suggestion: "You are not a member of this organization. Please contact an admin.",
	Err:        errors.New("not an organization member"),
}

var ErrorNoCreatePermission = &CLIError{
	Title:      "No permission to create",
	Suggestion: "You don't have permission to create resources in this organization.",
	Err:        errors.New("no create permission"),
}

// API Error Codes - Application (4000-4099)
var ErrorApplicationNotFoundAPI = &CLIError{
	Title:      "Application not found",
	Suggestion: "The application does not exist or you don't have access to it.",
	Err:        errors.New("application not found"),
}

var ErrorNoApplicationAccess = &CLIError{
	Title:      "No application access",
	Suggestion: "You don't have permission to access this application.",
	Err:        errors.New("no application access"),
}

var ErrorDuplicateAppName = &CLIError{
	Title:      "Application name already exists",
	Suggestion: "An application with this name already exists. Please choose a different name.",
	Err:        errors.New("duplicate application name"),
}

// API Error Codes - GitHub Integration (5000-5099)
var ErrorGitHubRepoNotFound = &CLIError{
	Title:      "GitHub repository not found",
	Suggestion: "Most likely, this is not a major application repository. \n\nYou can create a new application with 'major app create' or clone an existing application with 'major app clone'.",
	Err:        errors.New("github repository not found"),
}

var ErrorGitHubRepoAccessDenied = &CLIError{
	Title:      "GitHub repository access denied",
	Suggestion: "Unable to access the GitHub repository. Please check your permissions.",
	Err:        errors.New("github repository access denied"),
}

var ErrorGitHubCollaboratorAddFailed = &CLIError{
	Title:      "Failed to add GitHub collaborator",
	Suggestion: "Unable to add collaborator to the GitHub repository. Please check your permissions.",
	Err:        errors.New("github collaborator add failed"),
}

var ErrorForceUpgrade = &CLIError{
	Title:      "Your CLI version is out of date and must be upgraded.",
	Suggestion: "Run: major update",
	Err:        errors.New("CLI version is out of date and must be upgraded"),
}

var ErrorTokenNotActive = &CLIError{
	Title:      "Token not active",
	Suggestion: "Please run major user login to login again.",
	Err:        errors.New("token not active"),
}

var ErrorNoApplicationsAvailable = &CLIError{
	Title:      "No applications available for this organization",
	Suggestion: "Create an application first with 'major app create'",
	Err:        errors.New("no applications available"),
}

var ErrorGitRepositoryAccessFailed = &CLIError{
	Title:      "Failed to access repository after accepting invitation",
	Suggestion: "Please check your SSH keys are configured correctly. Run 'ssh -T git@github.com' to test your GitHub SSH connection.",
	Err:        errors.New("failed to access repository after accepting invitation"),
}

var ErrorGitCloneFailed = &CLIError{
	Title:      "Failed to clone repository",
	Suggestion: "Please check your SSH keys are configured correctly. Run 'ssh -T git@github.com' to test your GitHub SSH connection.",
	Err:        errors.New("failed to clone repository"),
}

var ErrorRepositoryAccessTimeout = &CLIError{
	Title:      "Timeout waiting for repository access",
	Suggestion: "Please try again after accepting the invitation.",
	Err:        errors.New("timeout waiting for repository access"),
}

var ErrorApplicationNameRequired = &CLIError{
	Title:      "Application name required",
	Suggestion: "Please enter a name for your application.",
	Err:        errors.New("application name required"),
}

var ErrorApplicationDescriptionRequired = &CLIError{
	Title:      "Application description required",
	Suggestion: "Please enter a description for your application.",
	Err:        errors.New("application description required"),
}

var ErrorNoTemplatesAvailable = &CLIError{
	Title:      "No templates available",
	Suggestion: "No templates are available. Please contact support.",
	Err:        errors.New("no templates available"),
}

var ErrorNoValidCloneMethodAvailable = &CLIError{
	Title:      "No valid clone method available",
	Suggestion: "Please check your SSH keys are configured correctly. Run 'ssh -T git@github.com' to test your GitHub SSH connection.",
	Err:        errors.New("no valid clone method available"),
}

var ErrorFailedToSelectResources = &CLIError{
	Title:      "Failed to select resources",
	Suggestion: "Please run 'major app resources' to select resources later.",
	Err:        errors.New("failed to select resources"),
}

var ErrorNotInGitRepository = &CLIError{
	Title:      "Not in a git repository",
	Suggestion: "You probably need to cd into your application directory first.",
	Err:        errors.New("not in a git repository"),
}

var ErrorNoOrganizationsAvailable = &CLIError{
	Title:      "No organizations available",
	Suggestion: "Please create one on https://app.major.build. Then run `major org select` to select it.",
	Err:        errors.New("no organizations available"),
}

var ErrorFailedToSelectResourcesTryAgain = &CLIError{
	Title:      "Failed to select resources",
	Suggestion: "We're not sure what went wrong. Please try again! Contact support if the problem persists.",
	Err:        errors.New("failed to select resources"),
}

var ErrorNoGitRemoteFoundInDirectory = &CLIError{
	Title:      "No git remote found in directory",
	Suggestion: "Please make sure you are in a git repository and have a remote origin set.",
	Err:        errors.New("no git remote found in directory"),
}

var ErrorOldProjectNotSupported = &CLIError{
	Title:      "Old project not supported",
	Suggestion: "This project is not supported. Please create a new project with 'major app create'.",
	Err:        errors.New("old project not supported"),
}
