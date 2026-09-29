package app

import (
	"testing"

	"github.com/major-technology/cli/clients/api"
)

func TestDBStatusText(t *testing.T) {
	resource, database := "res-1", "db-1"
	text := dbStatusText(&api.ManagedDatabaseStatusResponse{Status: "active", ResourceID: &resource, DatabaseID: &database, Message: "ready"})
	if text != "Status: active\nResource ID: res-1\nDatabase ID: db-1\nready" {
		t.Fatalf("text: %q", text)
	}
	if text := dbStatusText(&api.ManagedDatabaseStatusResponse{Status: "none", Message: "No database"}); text != "Status: none\nNo database" {
		t.Fatalf("none: %q", text)
	}
}
