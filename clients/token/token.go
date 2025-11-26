package token

import (
	clierrors "github.com/major-technology/cli/errors"
	"github.com/zalando/go-keyring"
)

const (
	// keyringService is the service name for storing credentials in the system keyring
	keyringService = "major-cli"
	// keyringUser is the username for storing credentials in the system keyring
	keyringUser = "default"
	// keyringOrgUser is the username for storing the default organization in the system keyring
	keyringOrgUser = "default-org"
	// keyringOrgName is the name for storing the default organization name in the system keyring
	keyringOrgName = "default-org-name"
	// keyringGithubUsername is the key for storing the GitHub username in the system keyring
	keyringGithubUsername = "github-username"
)

// storeToken saves the access token to the system keyring
func StoreToken(token string) error {
	err := keyring.Set(keyringService, keyringUser, token)
	if err != nil {
		return clierrors.WrapError("failed to store token in keyring", err)
	}
	return nil
}

// getToken retrieves the access token from the system keyring
func GetToken() (string, error) {
	token, err := keyring.Get(keyringService, keyringUser)
	if err != nil {
		return "", clierrors.WrapError("failed to get token from keyring", err)
	}
	return token, nil
}

// deleteToken removes the access token from the system keyring
func DeleteToken() error {
	err := keyring.Delete(keyringService, keyringUser)
	if err != nil {
		return clierrors.WrapError("failed to delete token from keyring", err)
	}
	return nil
}

// StoreDefaultOrg saves the default organization ID to the system keyring
func StoreDefaultOrg(orgID string, orgName string) error {
	err := keyring.Set(keyringService, keyringOrgUser, orgID)
	if err != nil {
		return clierrors.WrapError("failed to store default org in keyring", err)
	}
	err = keyring.Set(keyringService, keyringOrgName, orgName)
	if err != nil {
		return clierrors.WrapError("failed to store default org name in keyring", err)
	}
	return nil
}

// GetDefaultOrg retrieves the default organization ID from the system keyring
func GetDefaultOrg() (string, string, error) {
	orgID, err := keyring.Get(keyringService, keyringOrgUser)
	if err != nil {
		return "", "", clierrors.WrapError("failed to get default org from keyring", err)
	}
	orgName, err := keyring.Get(keyringService, keyringOrgName)
	if err != nil {
		return "", "", clierrors.WrapError("failed to get default org name from keyring", err)
	}
	return orgID, orgName, nil
}

// DeleteDefaultOrg removes the default organization ID from the system keyring
func DeleteDefaultOrg() error {
	err := keyring.Delete(keyringService, keyringOrgUser)
	if err != nil {
		return clierrors.WrapError("failed to delete default org from keyring", err)
	}
	err = keyring.Delete(keyringService, keyringOrgName)
	if err != nil {
		return clierrors.WrapError("failed to delete default org name from keyring", err)
	}
	return nil
}

// StoreGithubUsername saves the GitHub username to the system keyring
func StoreGithubUsername(username string) error {
	err := keyring.Set(keyringService, keyringGithubUsername, username)
	if err != nil {
		return clierrors.WrapError("failed to store GitHub username in keyring", err)
	}
	return nil
}

// GetGithubUsername retrieves the GitHub username from the system keyring
// Returns empty string and nil error if not found
func GetGithubUsername() (string, error) {
	username, err := keyring.Get(keyringService, keyringGithubUsername)
	if err != nil {
		// Check if it's a "not found" error
		if err == keyring.ErrNotFound {
			return "", nil
		}
		return "", clierrors.WrapError("failed to get GitHub username from keyring", err)
	}
	return username, nil
}

// DeleteGithubUsername removes the GitHub username from the system keyring
func DeleteGithubUsername() error {
	err := keyring.Delete(keyringService, keyringGithubUsername)
	if err != nil {
		// Ignore "not found" errors
		if err == keyring.ErrNotFound {
			return nil
		}
		return clierrors.WrapError("failed to delete GitHub username from keyring", err)
	}
	return nil
}
