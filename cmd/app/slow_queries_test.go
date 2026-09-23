package app

import "testing"

func TestSlowQueriesListRegistered(t *testing.T) {
	for _, sub := range slowQueriesCmd.Commands() {
		if sub.Name() == "list" {
			return
		}
	}

	t.Fatal("app slow-queries missing subcommand \"list\"")
}

func TestSlowQueriesListExposesFlags(t *testing.T) {
	for _, name := range []string{"environment", "days"} {
		if slowQueriesListCmd.Flags().Lookup(name) == nil {
			t.Fatalf("app slow-queries list missing --%s", name)
		}
	}
}

func TestSlowQueriesRegisteredOnAppCmd(t *testing.T) {
	for _, sub := range Cmd.Commands() {
		if sub.Name() == "slow-queries" {
			return
		}
	}

	t.Fatal("app slow-queries not registered on the app command")
}
