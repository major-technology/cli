package utils

import (
	"fmt"
	"os/exec"
	"runtime"
)

// OpenBrowser opens the specified URL in the default browser
func OpenBrowser(url string) error {
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
