package file

import "testing"

func TestRoleForAccess(t *testing.T) {
	cases := map[string]string{"viewer": "File:Viewer", "editor": "File:Editor", "admin": "File:Admin", "Viewer": "File:Viewer"}
	for access, want := range cases {
		got, err := roleForAccess(access)
		if err != nil {
			t.Fatalf("%s: %v", access, err)
		}
		if got != want {
			t.Fatalf("%s: role = %q, want %q", access, got, want)
		}
	}

	if _, err := roleForAccess("owner"); err == nil {
		t.Fatal("expected error for owner")
	}
}
