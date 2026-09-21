package app

import (
	"strings"
	"testing"
)

func TestAiProxySubcommandsRegistered(t *testing.T) {
	want := map[string]bool{"status": false, "enable": false}

	for _, sub := range aiProxyCmd.Commands() {
		if _, ok := want[sub.Name()]; ok {
			want[sub.Name()] = true
		}
	}

	for name, found := range want {
		if !found {
			t.Fatalf("app ai-proxy missing subcommand %q", name)
		}
	}
}

func TestAiProxyEnableHelpStatesTheSpendingLimit(t *testing.T) {
	if !strings.Contains(aiProxyEnableCmd.Long, "$10") {
		t.Fatal("app ai-proxy enable help must state the monthly spending limit")
	}
}
