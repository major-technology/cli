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
