package user

import (
	"fmt"
	"os"

	mjrToken "github.com/major-technology/cli/clients/token"
	clierrors "github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

var ensureAuthCmd = &cobra.Command{
	Use:    "ensure-auth",
	Short:  "Ensure valid authentication, running login if token is missing or expired",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runEnsureAuth(cmd)
	},
}

func runEnsureAuth(cmd *cobra.Command) error {
	// All UI output goes to stderr so stdout contains only the token.
	cmd.SetOut(os.Stderr)
	cmd.SetErr(os.Stderr)

	token, needsLogin := checkExistingAuth()
	if !needsLogin {
		fmt.Fprint(os.Stdout, token)
		return nil
	}

	if err := runBrowserLogin(cmd); err != nil {
		return err
	}

	if err := ensureOrgSelected(cmd); err != nil {
		return err
	}

	newToken, err := mjrToken.GetToken()
	if err != nil {
		return clierrors.WrapError("failed to get token after login", err)
	}
	fmt.Fprint(os.Stdout, newToken)
	return nil
}

// checkExistingAuth returns (token, needsLogin). If the stored token is valid,
// it returns the token with needsLogin=false. Otherwise needsLogin=true.
func checkExistingAuth() (string, bool) {
	token, err := mjrToken.GetToken()
	if err != nil || token == "" {
		return "", true
	}

	client := singletons.GetAPIClient()
	_, err = client.VerifyToken()
	if err != nil {
		return "", true
	}

	return token, false
}

// runBrowserLogin performs the device-code login flow (opens browser, polls).
func runBrowserLogin(cmd *cobra.Command) error {
	client := singletons.GetAPIClient()
	startResp, err := client.StartLogin()
	if err != nil {
		return clierrors.WrapError("failed to start login", err)
	}

	_ = utils.OpenBrowser(startResp.VerificationURI)
	cmd.Println("Session expired. Opening browser for authentication...")
	cmd.Printf("If the browser doesn't open, visit:\n%s\n", startResp.VerificationURI)

	token, err := pollForToken(cmd, client, startResp.DeviceCode, startResp.Interval, startResp.ExpiresIn)
	if err != nil {
		return clierrors.WrapError("authentication failed", err)
	}

	if err := mjrToken.StoreToken(token); err != nil {
		return clierrors.WrapError("failed to store token", err)
	}

	return nil
}

// ensureOrgSelected verifies an org is set. If not, it auto-selects when
// there's exactly one org, or returns an error asking the user to pick one.
func ensureOrgSelected(cmd *cobra.Command) error {
	orgID, _, err := mjrToken.GetDefaultOrg()
	if err == nil && orgID != "" {
		return nil
	}

	client := singletons.GetAPIClient()
	orgsResp, err := client.GetOrganizations()
	if err != nil {
		return clierrors.WrapError("failed to fetch organizations", err)
	}

	if len(orgsResp.Organizations) == 0 {
		return clierrors.ErrorNoOrganizationsAvailable
	}

	if len(orgsResp.Organizations) == 1 {
		org := orgsResp.Organizations[0]
		if err := mjrToken.StoreDefaultOrg(org.ID, org.Name); err != nil {
			return clierrors.WrapError("failed to store default organization", err)
		}
		cmd.Printf("Organization set to: %s\n", org.Name)
		return nil
	}

	return &clierrors.CLIError{
		Title:      "Multiple organizations available",
		Suggestion: "Run 'major org select' to choose a default organization.",
		Err:        fmt.Errorf("interactive org selection required"),
	}
}
