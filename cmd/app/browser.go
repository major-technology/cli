package app

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/major-technology/cli/clients/workspace"
	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

// sandboxToolsURLEnv names the app sandbox pod's tool endpoint. Only an app
// sandbox shell sets it. The endpoint keeps browser navigation on the preview
// server and forwards the call to the pod's Playwright browser.
const sandboxToolsURLEnv = "MAJOR_SANDBOX_TOOLS_URL"

// screenshotDir is gitignored in every app sandbox workspace, and read_file
// returns image files from it as images.
const screenshotDir = ".session-files"

var (
	flagBrowserFullPage      bool
	flagBrowserConsoleLevel  string
	flagBrowserIncludeStatic bool
	flagBrowserWaitTime      float64
	flagBrowserWaitText      string
	flagBrowserWaitTextGone  string
)

type sandboxToolContent struct {
	Type     string `json:"type"`
	Text     string `json:"text"`
	Data     string `json:"data"`
	MimeType string `json:"mimeType"`
}

type sandboxToolResult struct {
	Content []sandboxToolContent `json:"content"`
	IsError bool                 `json:"isError"`
}

var browserCmd = &cobra.Command{
	Use:   "browser",
	Short: "Drive the app sandbox's browser against the preview server",
	Long: `Drive the browser in the app sandbox. Navigation is limited to the
preview server at localhost:3000.

These commands run only inside a Major app sandbox.`,
	Args: utils.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.Help()
		return nil
	},
}

var browserNavigateCmd = &cobra.Command{
	Use:   "navigate <url>",
	Short: "Open a page of the preview server (a relative path like /dashboard works)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runBrowserTextTool(cmd, "browser_navigate", map[string]any{"url": args[0]})
	},
}

var browserSnapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Print the page's accessibility snapshot",
	Args:  utils.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runBrowserTextTool(cmd, "browser_snapshot", map[string]any{})
	},
}

var browserScreenshotCmd = &cobra.Command{
	Use:   "screenshot [filename]",
	Short: "Save a screenshot under .session-files/ and print its path",
	Long: `Save a screenshot of the current page under .session-files/ in the app
workspace and print its path. Read that path with read_file to view the image.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var filename string
		if len(args) == 1 {
			filename = args[0]
		}

		return runBrowserScreenshot(cmd, filename)
	},
}

var browserConsoleCmd = &cobra.Command{
	Use:   "console",
	Short: "Print the page's console messages",
	Args:  utils.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runBrowserTextTool(cmd, "browser_console_messages", map[string]any{"level": flagBrowserConsoleLevel})
	},
}

var browserNetworkCmd = &cobra.Command{
	Use:   "network",
	Short: "Print the network requests since the page loaded",
	Args:  utils.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runBrowserTextTool(cmd, "browser_network_requests", map[string]any{"includeStatic": flagBrowserIncludeStatic})
	},
}

var browserWaitForCmd = &cobra.Command{
	Use:   "wait-for",
	Short: "Wait for text to appear or disappear, or for a number of seconds",
	Args:  utils.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		toolArgs := map[string]any{}
		if cmd.Flags().Changed("time") {
			toolArgs["time"] = flagBrowserWaitTime
		}
		if flagBrowserWaitText != "" {
			toolArgs["text"] = flagBrowserWaitText
		}
		if flagBrowserWaitTextGone != "" {
			toolArgs["textGone"] = flagBrowserWaitTextGone
		}

		if len(toolArgs) == 0 {
			return &errors.CLIError{Title: "pass --time, --text, or --text-gone"}
		}

		return runBrowserTextTool(cmd, "browser_wait_for", toolArgs)
	},
}

var browserCloseCmd = &cobra.Command{
	Use:   "close",
	Short: "Close the browser page",
	Args:  utils.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runBrowserTextTool(cmd, "browser_close", map[string]any{})
	},
}

func init() {
	browserScreenshotCmd.Flags().BoolVar(&flagBrowserFullPage, "full-page", false, "Capture the full scrollable page instead of the viewport")
	browserConsoleCmd.Flags().StringVar(&flagBrowserConsoleLevel, "level", "info", "Lowest level to show: error, warning, info, or debug")
	browserNetworkCmd.Flags().BoolVar(&flagBrowserIncludeStatic, "include-static", false, "Include successful static resources (images, fonts, scripts)")
	browserWaitForCmd.Flags().Float64Var(&flagBrowserWaitTime, "time", 0, "Seconds to wait")
	browserWaitForCmd.Flags().StringVar(&flagBrowserWaitText, "text", "", "Text to wait for")
	browserWaitForCmd.Flags().StringVar(&flagBrowserWaitTextGone, "text-gone", "", "Text to wait to disappear")

	browserCmd.AddCommand(browserNavigateCmd)
	browserCmd.AddCommand(browserSnapshotCmd)
	browserCmd.AddCommand(browserScreenshotCmd)
	browserCmd.AddCommand(browserConsoleCmd)
	browserCmd.AddCommand(browserNetworkCmd)
	browserCmd.AddCommand(browserWaitForCmd)
	browserCmd.AddCommand(browserCloseCmd)
}

func runBrowserTextTool(cmd *cobra.Command, name string, args map[string]any) error {
	result, err := callSandboxTool(name, args)
	if err != nil {
		return err
	}

	for _, c := range result.Content {
		if c.Type == "text" {
			fmt.Fprintln(cmd.OutOrStdout(), c.Text)
		}
	}

	return nil
}

func runBrowserScreenshot(cmd *cobra.Command, filename string) error {
	root, _, err := workspace.Locate("")
	if err != nil {
		return errors.WrapError("failed to find the app workspace", err)
	}

	toolArgs := map[string]any{"type": "png"}
	if flagBrowserFullPage {
		toolArgs["fullPage"] = true
	}

	result, err := callSandboxTool("browser_take_screenshot", toolArgs)
	if err != nil {
		return err
	}

	for _, c := range result.Content {
		if c.Type != "image" {
			continue
		}

		data, err := base64.StdEncoding.DecodeString(c.Data)
		if err != nil {
			return errors.WrapError("failed to decode the screenshot", err)
		}

		name := filepath.Base(filename)
		if filename == "" || name == "." || name == ".." || name == string(filepath.Separator) {
			name = fmt.Sprintf("screenshot-%d.png", time.Now().UnixMilli())
		}

		rel := filepath.Join(screenshotDir, name)
		if err := os.MkdirAll(filepath.Join(root, screenshotDir), 0o755); err != nil {
			return errors.WrapError("failed to create "+screenshotDir, err)
		}
		if err := os.WriteFile(filepath.Join(root, rel), data, 0o644); err != nil {
			return errors.WrapError("failed to save the screenshot", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Saved screenshot to %s\n", rel)
		return nil
	}

	return &errors.CLIError{Title: "the browser returned no screenshot image"}
}

func callSandboxTool(name string, args map[string]any) (*sandboxToolResult, error) {
	toolsURL := os.Getenv(sandboxToolsURLEnv)
	if toolsURL == "" {
		return nil, &errors.CLIError{
			Title:      "Browser commands run only inside a Major sandbox",
			Suggestion: "Start the app's sandbox, then run this command in its shell.",
		}
	}

	body, err := json.Marshal(args)
	if err != nil {
		return nil, errors.WrapError("failed to encode the browser arguments", err)
	}

	resp, err := http.Post(toolsURL+name, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, errors.WrapError("failed to reach the sandbox browser", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &errors.CLIError{Title: fmt.Sprintf("the sandbox browser answered HTTP %d", resp.StatusCode)}
	}

	var result sandboxToolResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, errors.WrapError("failed to read the browser result", err)
	}

	if result.IsError {
		var texts []string
		for _, c := range result.Content {
			if c.Type == "text" {
				texts = append(texts, c.Text)
			}
		}

		return nil, &errors.CLIError{Title: strings.Join(texts, "\n")}
	}

	return &result, nil
}
