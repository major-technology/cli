package user

import (
	mjrToken "github.com/major-technology/cli/clients/token"
	"github.com/spf13/cobra"
)

var tokenCmd = &cobra.Command{
	Use:    "token",
	Short:  "Print the stored CLI token",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runToken(cmd)
	},
}

func runToken(cmd *cobra.Command) error {
	token, err := mjrToken.GetToken()
	if err != nil {
		return err
	}

	cmd.Print(token)
	return nil
}
