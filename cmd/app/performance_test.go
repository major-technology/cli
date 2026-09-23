package app

import "testing"

func TestPerformanceListRegistered(t *testing.T) {
	for _, sub := range performanceCmd.Commands() {
		if sub.Name() == "list" {
			return
		}
	}

	t.Fatal("app performance missing subcommand \"list\"")
}

func TestPerformanceListExposesFlags(t *testing.T) {
	for _, name := range []string{"environment", "days", "all"} {
		if performanceListCmd.Flags().Lookup(name) == nil {
			t.Fatalf("app performance list missing --%s", name)
		}
	}
}

func TestPerformanceRegisteredOnAppCmd(t *testing.T) {
	for _, sub := range Cmd.Commands() {
		if sub.Name() == "performance" {
			return
		}
	}

	t.Fatal("app performance not registered on the app command")
}
