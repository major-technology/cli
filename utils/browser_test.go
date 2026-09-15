package utils

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestOpenOrPrintBrowserPrintsWithoutOpening(t *testing.T) {
	orig := BrowserStart
	opened := []string{}
	BrowserStart = func(url string) error {
		opened = append(opened, url)
		return nil
	}
	t.Cleanup(func() { BrowserStart = orig })

	cmd := &cobra.Command{Use: "docs"}
	root := &cobra.Command{Use: "major"}
	root.PersistentFlags().Bool("non-interactive", false, "Never prompt or open a browser")
	root.AddCommand(cmd)
	cmd.SetIn(strings.NewReader(""))
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)

	url := "https://docs.major.build/"
	if err := OpenOrPrintBrowser(cmd, url); err != nil {
		t.Fatal(err)
	}
	if len(opened) != 0 {
		t.Fatalf("opened browser in non-interactive mode: %v", opened)
	}
	if !strings.Contains(out.String(), url) {
		t.Fatalf("output %q must name destination URL", out.String())
	}
}
