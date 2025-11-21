package cmd

import (
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

var docsCmd = &cobra.Command{
	Use:   "docs",
	Short: "Open the Major documentation",
	Long:  `Opens the Major documentation in your default browser (https://docs.major.build/)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.Println("Opening documentation...")
		return utils.OpenBrowser("https://docs.major.build/")
	},
}

func init() {
	rootCmd.AddCommand(docsCmd)
}
