package app

import "testing"

func TestErrorsSubcommandsRegistered(t *testing.T) {
	want := map[string]bool{"list": false, "get": false, "resolve": false, "enable": false}

	for _, sub := range errorsCmd.Commands() {
		if _, ok := want[sub.Name()]; ok {
			want[sub.Name()] = true
		}
	}

	for name, found := range want {
		if !found {
			t.Fatalf("app errors missing subcommand %q", name)
		}
	}
}

func TestErrorsListExposesIgnoredFlag(t *testing.T) {
	if errorsListCmd.Flags().Lookup("ignored") == nil {
		t.Fatal("app errors list missing --ignored")
	}
}

func TestErrorsRegisteredOnAppCmd(t *testing.T) {
	for _, sub := range Cmd.Commands() {
		if sub.Name() == "errors" {
			return
		}
	}

	t.Fatal("app errors not registered on the app command")
}
