package config

import (
	"testing"
)

func TestAPIURLOverride(t *testing.T) {
	t.Setenv("MAJOR_API_URL", "http://localhost:3001/cli/")
	got, err := Load("configs/prod.json")
	if err != nil {
		t.Fatal(err)
	}
	if got.APIURL != "http://localhost:3001/cli" {
		t.Fatalf("APIURL = %q", got.APIURL)
	}
}

func TestAPIURLDefaultPreservedWithoutOverride(t *testing.T) {
	t.Setenv("MAJOR_API_URL", "")
	got, err := Load("configs/prod.json")
	if err != nil {
		t.Fatal(err)
	}
	if got.APIURL != "https://api.prod.major.build/cli" {
		t.Fatalf("APIURL = %q", got.APIURL)
	}
}
