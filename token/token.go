package token

import (
	"fmt"

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
)

// storeToken saves the access token to the system keyring
func StoreToken(token string) error {
	err := keyring.Set(keyringService, keyringUser, token)
	if err != nil {
		return fmt.Errorf("failed to store token in keyring: %w", err)
	}
	return nil
}

// getToken retrieves the access token from the system keyring
func GetToken() (string, error) {
	token, err := keyring.Get(keyringService, keyringUser)
	if err != nil {
		return "", fmt.Errorf("failed to get token from keyring: %w", err)
	}
	return token, nil
}

// deleteToken removes the access token from the system keyring
func DeleteToken() error {
	err := keyring.Delete(keyringService, keyringUser)
	if err != nil {
		return fmt.Errorf("failed to delete token from keyring: %w", err)
	}
	return nil
}

// StoreDefaultOrg saves the default organization ID to the system keyring
func StoreDefaultOrg(orgID string, orgName string) error {
	err := keyring.Set(keyringService, keyringOrgUser, orgID)
	if err != nil {
		return fmt.Errorf("failed to store default org in keyring: %w", err)
	}
	err = keyring.Set(keyringService, keyringOrgName, orgName)
	if err != nil {
		return fmt.Errorf("failed to store default org name in keyring: %w", err)
	}
	return nil
}

// GetDefaultOrg retrieves the default organization ID from the system keyring
func GetDefaultOrg() (string, string, error) {
	orgID, err := keyring.Get(keyringService, keyringOrgUser)
	if err != nil {
		return "", "", fmt.Errorf("failed to get default org from keyring: %w", err)
	}
	orgName, err := keyring.Get(keyringService, keyringOrgName)
	if err != nil {
		return "", "", fmt.Errorf("failed to get default org name from keyring: %w", err)
	}
	return orgID, orgName, nil
}

// DeleteDefaultOrg removes the default organization ID from the system keyring
func DeleteDefaultOrg() error {
	err := keyring.Delete(keyringService, keyringOrgUser)
	if err != nil {
		return fmt.Errorf("failed to delete default org from keyring: %w", err)
	}
	err = keyring.Delete(keyringService, keyringOrgName)
	if err != nil {
		return fmt.Errorf("failed to delete default org name from keyring: %w", err)
	}
	return nil
}
