package file

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	mjrToken "github.com/major-technology/cli/clients/token"
	clierrors "github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/middleware"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

// maxFileBytes mirrors the API cap.
const maxFileBytes = 5 * 1024 * 1024

var (
	flagPushFileID string
	flagPushName   string
	flagPushJSON   bool
)

var pushCmd = &cobra.Command{
	Use:   "push <path>",
	Short: "Push a .md or .html file and print its Major link",
	Long: `Push a local markdown or HTML file. Without --file-id a new file is created
in your default organization. With --file-id a new version is added to that file.`,
	Args: cobra.ExactArgs(1),
	PreRunE: middleware.Compose(
		middleware.CheckLogin,
	),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPush(cmd, args[0])
	},
}

func init() {
	pushCmd.Flags().StringVar(&flagPushFileID, "file-id", "", "Add a new version to this existing file")
	pushCmd.Flags().StringVar(&flagPushName, "name", "", "Display name (defaults to the file name)")
	pushCmd.Flags().BoolVar(&flagPushJSON, "json", false, "Output in JSON format")
}

// kindForPath maps the extension to the API kind, or errors for anything else.
func kindForPath(path string) (string, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md":
		return "markdown", nil
	case ".html":
		return "html", nil
	default:
		return "", &clierrors.CLIError{
			Title:      fmt.Sprintf("unsupported file type %q", filepath.Ext(path)),
			Suggestion: "Only .md and .html files can be pushed.",
		}
	}
}

func runPush(cmd *cobra.Command, path string) error {
	if flagPushFileID != "" && flagPushName != "" {
		return &clierrors.CLIError{
			Title:      "--name cannot be combined with --file-id",
			Suggestion: "Use `major file rename <file-id> <new-name>` to rename an existing file.",
		}
	}

	kind, err := kindForPath(path)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return clierrors.WrapError("failed to read file", err)
	}
	if len(data) > maxFileBytes {
		return &clierrors.CLIError{
			Title:      fmt.Sprintf("file is %d bytes; the limit is %d bytes", len(data), maxFileBytes),
			Suggestion: "Shrink the file or split it.",
		}
	}

	apiClient := singletons.GetAPIClient()

	if flagPushFileID != "" {
		resp, err := apiClient.PushFileVersion(flagPushFileID, kind, string(data))
		if err != nil {
			return err
		}
		return printPushResult(cmd, resp.File.Link, resp.File.ID, resp.File.Version)
	}

	name := flagPushName
	if name == "" {
		name = filepath.Base(path)
	}

	orgID, _, err := mjrToken.GetDefaultOrg()
	if err != nil {
		return err
	}

	resp, err := apiClient.CreateFile(orgID, name, kind, string(data))
	if err != nil {
		return err
	}
	return printPushResult(cmd, resp.File.Link, resp.File.ID, resp.File.Version)
}

func printPushResult(cmd *cobra.Command, link, fileID string, version int) error {
	if flagPushJSON {
		return utils.WriteJSON(cmd, map[string]any{"fileId": fileID, "version": version, "link": link})
	}
	fmt.Fprintln(cmd.OutOrStdout(), link)
	utils.Hint(cmd, fmt.Sprintf("file id %s, version %d. Push an update with: major file push <path> --file-id %s", fileID, version, fileID))
	return nil
}
