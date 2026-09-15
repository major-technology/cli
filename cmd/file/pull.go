package file

import (
	"fmt"
	"io"
	"net/http"
	"os"

	clierrors "github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/middleware"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

var flagPullOut string

var pullCmd = &cobra.Command{
	Use:   "pull <file-id>",
	Short: "Download the latest version of a hosted file",
	Args:  cobra.ExactArgs(1),
	PreRunE: middleware.Compose(
		middleware.CheckLogin,
	),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPull(cmd, args[0])
	},
}

func init() {
	pullCmd.Flags().StringVarP(&flagPullOut, "out", "o", "", "Write to this path instead of stdout")
}

func runPull(cmd *cobra.Command, fileID string) error {
	apiClient := singletons.GetAPIClient()

	meta, err := apiClient.GetFileContentURL(fileID)
	if err != nil {
		return clierrors.WrapError("failed to resolve file", err)
	}

	resp, err := http.Get(meta.URL)
	if err != nil {
		return clierrors.WrapError("failed to download file", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return clierrors.WrapError("failed to download file", fmt.Errorf("status %d", resp.StatusCode))
	}

	var out io.Writer = cmd.OutOrStdout()
	if flagPullOut != "" {
		f, err := os.Create(flagPullOut)
		if err != nil {
			return clierrors.WrapError("failed to create output file", err)
		}
		defer f.Close()
		out = f
	}

	if _, err := io.Copy(out, resp.Body); err != nil {
		return clierrors.WrapError("failed to write file", err)
	}
	return nil
}
