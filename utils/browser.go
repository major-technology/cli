package utils

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
)

// BrowserStart launches a URL in the platform browser. Tests replace it.
var BrowserStart = startBrowser

func startBrowser(url string) error {
	var execCmd *exec.Cmd

	switch runtime.GOOS {
	case "linux":
		execCmd = exec.Command("xdg-open", url)
	case "windows":
		execCmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		execCmd = exec.Command("open", url)
	default:
		return fmt.Errorf("unsupported platform")
	}

	return execCmd.Start()
}

// OpenBrowser opens the specified URL in the default browser
func OpenBrowser(url string) error {
	return BrowserStart(url)
}

// OpenOrPrintBrowser prints url without opening it in non-interactive mode.
func OpenOrPrintBrowser(cmd *cobra.Command, url string) error {
	if IsNonInteractive(cmd) {
		fmt.Fprintln(cmd.OutOrStdout(), url)
		return nil
	}
	return OpenBrowser(url)
}
