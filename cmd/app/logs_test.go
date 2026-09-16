package app

import "testing"

func TestLogsExposesPreviewFlag(t *testing.T) {
	if logsCmd.Flags().Lookup("preview") == nil {
		t.Fatal("app logs missing --preview")
	}
}
