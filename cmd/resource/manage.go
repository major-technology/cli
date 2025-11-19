package resource

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/major-technology/cli/middleware"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

// manageCmd represents the manage command
var manageCmd = &cobra.Command{
	Use:   "manage",
	Short: "Manage application resources",
	Long:  `Select and configure resources for your application.`,
	PreRunE: middleware.Compose(
		middleware.CheckLogin,
	),
	Run: func(cobraCmd *cobra.Command, args []string) {
		cobra.CheckErr(runManage(cobraCmd))
	},
}

func runManage(cobraCmd *cobra.Command) error {
	// Get application info from current directory
	appInfo, err := utils.GetApplicationInfo("")
	if err != nil {
		return fmt.Errorf("failed to identify application: %w", err)
	}

	apiClient := singletons.GetAPIClient()

	cobraCmd.Println("\nSelecting resources for your application...")
	selectedResources, err := utils.SelectApplicationResources(cobraCmd, apiClient, appInfo.OrganizationID, appInfo.ApplicationID)
	if err != nil {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9")) // Red
		cobraCmd.Println(errorStyle.Render("Failed to configure resources."))
		return err
	}

	if selectedResources == nil {
		return nil
	}

	// Handle post-selection logic based on template
	templateName := ""
	if appInfo.TemplateName != nil {
		templateName = *appInfo.TemplateName
	}

	if templateName == "Vite" {
		cobraCmd.Println("\nAdding resources to Vite project...")
		if err := utils.AddResourcesToViteProject(cobraCmd, ".", selectedResources, appInfo.ApplicationID); err != nil {
			cobraCmd.Printf("Warning: Failed to add resources to project: %v\n", err)
			cobraCmd.Println("You can manually add them later using 'pnpm clients:add'")
		}
	} else {
		// Default/Next.js flow: regenerate RESOURCES.md
		cobraCmd.Println("\nUpdating RESOURCES.md...")
		filePath, _, err := utils.GenerateResourcesFile(".")
		if err != nil {
			cobraCmd.Printf("Warning: Failed to update RESOURCES.md: %v\n", err)
		} else {
			cobraCmd.Printf("✓ Updated %s\n", filePath)
		}
	}

	return nil
}

func init() {
	// Add manage subcommand
	Cmd.AddCommand(manageCmd)
}
