package file

import "testing"

func TestKindForPath(t *testing.T) {
	cases := map[string]string{
		"notes.md":   "markdown",
		"Report.MD":  "markdown",
		"index.html": "html",
		"page.HTML":  "html",
	}
	for path, want := range cases {
		got, err := kindForPath(path)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		if got != want {
			t.Fatalf("%s: kind = %q, want %q", path, got, want)
		}
	}

	if _, err := kindForPath("archive.zip"); err == nil {
		t.Fatal("expected error for .zip")
	}
}
