package token

import (
	"testing"

	"github.com/zalando/go-keyring"
)

func TestGetTokenUsesEnvironment(t *testing.T) {
	t.Setenv("MAJOR_TOKEN", "test-injected-token")
	got, err := GetToken()
	if err != nil {
		t.Fatal(err)
	}
	if got != "test-injected-token" {
		t.Fatalf("got %q", got)
	}
}

func TestGetTokenFallsBackToKeyringWhenEnvEmpty(t *testing.T) {
	t.Setenv("MAJOR_TOKEN", "")
	keyring.MockInit()
	if err := StoreToken("keychain-token"); err != nil {
		t.Fatal(err)
	}
	got, err := GetToken()
	if err != nil {
		t.Fatal(err)
	}
	if got != "keychain-token" {
		t.Fatalf("got %q", got)
	}
}

func TestGetTokenErrorsWhenEnvEmptyAndKeyringEmpty(t *testing.T) {
	t.Setenv("MAJOR_TOKEN", "")
	keyring.MockInit()
	_, err := GetToken()
	if err == nil {
		t.Fatal("expected error when no env token and no keyring token")
	}
}

func TestGetTokenPrefersEnvironmentOverKeyring(t *testing.T) {
	t.Setenv("MAJOR_TOKEN", "test-injected-token")
	keyring.MockInit()
	if err := StoreToken("keychain-token"); err != nil {
		t.Fatal(err)
	}
	got, err := GetToken()
	if err != nil {
		t.Fatal(err)
	}
	if got != "test-injected-token" {
		t.Fatalf("got %q", got)
	}
}

func TestHasInjectedToken(t *testing.T) {
	t.Setenv("MAJOR_TOKEN", "test-injected-token")
	if !HasInjectedToken() {
		t.Fatal("expected HasInjectedToken to be true")
	}
	t.Setenv("MAJOR_TOKEN", "")
	if HasInjectedToken() {
		t.Fatal("expected HasInjectedToken to be false when empty")
	}
}

func TestGetDefaultOrgPrefersEnvironmentOverKeyring(t *testing.T) {
	t.Setenv("MAJOR_ORG_ID", "env-org")
	keyring.MockInit()
	if err := StoreDefaultOrg("keychain-org", "Keychain Org"); err != nil {
		t.Fatal(err)
	}
	id, name, err := GetDefaultOrg()
	if err != nil {
		t.Fatal(err)
	}
	if id != "env-org" || name != "env-org" {
		t.Fatalf("got %q %q", id, name)
	}
}
