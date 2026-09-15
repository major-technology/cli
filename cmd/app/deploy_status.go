package app

import (
	"github.com/major-technology/cli/errors"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

var (
	flagDeployStatusVersionID string
	flagDeployStatusJSON      bool
)

var deployStatusCmd = &cobra.Command{
	Use:   "deploy-status",
	Short: "Check the status of a deployment",
	Long:  `Returns the current deployment status for a given version ID.`,
	RunE: func(cobraCmd *cobra.Command, args []string) error {
		return runDeployStatus(cobraCmd)
	},
}

func init() {
	deployStatusCmd.Flags().StringVar(&flagDeployStatusVersionID, "version-id", "", "Version ID to check")
	deployStatusCmd.Flags().BoolVar(&flagDeployStatusJSON, "json", false, "Output in JSON format")
	deployStatusCmd.MarkFlagRequired("version-id")
}

func runDeployStatus(cobraCmd *cobra.Command) error {
	applicationID, organizationID, _, err := getApplicationAndOrgID()
	if err != nil {
		return errors.WrapError("failed to get application ID", err)
	}

	apiClient := singletons.GetAPIClient()
	resp, err := apiClient.GetVersionStatus(applicationID, organizationID, flagDeployStatusVersionID)
	if err != nil {
		return errors.WrapError("failed to get deployment status", err)
	}

	type statusJSON struct {
		Status          string `json:"status"`
		DeploymentError string `json:"deploymentError,omitempty"`
		AppURL          string `json:"appUrl,omitempty"`
	}

	return utils.WriteJSON(cobraCmd, statusJSON{
		Status:          resp.Status,
		DeploymentError: resp.DeploymentError,
		AppURL:          resp.AppURL,
	})
}
