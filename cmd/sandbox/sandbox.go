package sandbox

import (
	"fmt"

	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/cmd/target"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "sandbox",
	Short: "Commands that run inside a Major sandbox",
	Args:  utils.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	Cmd.AddCommand(newUploadURLCmd())
}

type uploadURLResult struct{ *api.SandboxUploadURLResponse }

func (r uploadURLResult) String() string {
	return fmt.Sprintf("%s\n\nUpload a file to it (single use, expires %s):\n  curl -T <file> '%s'", r.UploadURL, r.ExpiresAt, r.UploadURL)
}

func newUploadURLCmd() *cobra.Command {
	var path string
	cmd := &cobra.Command{
		Use:   "upload-url",
		Short: "Get a single-use URL that uploads one file into this sandbox at --path",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if path == "" {
				return fmt.Errorf("--path is required")
			}
			result, err := singletons.GetAPIClient().CreateSandboxUploadURL(path)
			if err != nil {
				return err
			}
			return target.Output(cmd, uploadURLResult{result})
		},
	}
	cmd.Flags().StringVar(&path, "path", "", "Destination path relative to the sandbox workspace root, e.g. public/logo.png")
	cmd.Flags().Bool("json", false, "Print one JSON result")
	return cmd
}
