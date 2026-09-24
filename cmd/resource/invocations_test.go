package resource

import "testing"

func TestInvocationsCommand(t *testing.T) {
	if invocationsCmd.Parent() != Cmd {
		t.Fatal("resource invocations not registered")
	}
	for _, name := range []string{"environment", "days", "slow"} {
		if invocationsCmd.Flags().Lookup(name) == nil {
			t.Fatalf("resource invocations missing --%s", name)
		}
	}
}
