package app

import (
	"strings"
	"testing"
)

func TestLogsHelpDescribesBothBackends(t *testing.T) {
	searchHelp := logsCmd.Flags().Lookup("search").Usage
	if !strings.Contains(searchHelp, "case-sensitive for deployed") || !strings.Contains(searchHelp, "case-insensitive with --preview") {
		t.Fatalf("search help must distinguish backends: %s", searchHelp)
	}
	if !strings.Contains(logsCmd.Long, "Both deployed and preview logs support pagination") {
		t.Fatalf("pagination help must cover preview logs: %s", logsCmd.Long)
	}
}

func TestLogsExposesPreviewFlag(t *testing.T) {
	if logsCmd.Flags().Lookup("preview") == nil {
		t.Fatal("app logs missing --preview")
	}
}
