package file

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	clierrors "github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/middleware"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

var flagPullOut string

// downloadClient fetches the file body from the short-lived S3 URL; 120s
// mirrors the upload timeout since these can be large files too.
var downloadClient = &http.Client{Timeout: 120 * time.Second}

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
		return err
	}

	resp, err := downloadClient.Get(meta.URL)
	if err != nil {
		return clierrors.WrapError("failed to download file", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return clierrors.WrapError("failed to download file", fmt.Errorf("status %d", resp.StatusCode))
	}

	if flagPullOut == "" {
		if _, err := io.Copy(cmd.OutOrStdout(), resp.Body); err != nil {
			return clierrors.WrapError("failed to write file", err)
		}
		return nil
	}

	// Write to a temp file first and rename into place on success, so a
	// failed download never leaves a truncated file at the target path.
	tmpPath := flagPullOut + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return clierrors.WrapError("failed to create output file", err)
	}

	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return clierrors.WrapError("failed to write file", err)
	}

	if err := f.Close(); err != nil {
		os.Remove(tmpPath)
		return clierrors.WrapError("failed to write file", err)
	}

	if err := os.Rename(tmpPath, flagPullOut); err != nil {
		os.Remove(tmpPath)
		return clierrors.WrapError("failed to write file", err)
	}

	return nil
}
