package user

import (
	"fmt"
	"os"

	mjrToken "github.com/major-technology/cli/clients/token"
	"github.com/spf13/cobra"
)

var tokenCmd = &cobra.Command{
	Use:    "token",
	Short:  "Print the stored CLI token",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runToken()
	},
}

func runToken() error {
	token, err := mjrToken.GetToken()
	if err != nil {
		return err
	}

	fmt.Fprint(os.Stdout, token)
	return nil
}
