package agent

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/cmd/target"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

type slackConnectResult struct{ *api.AgentSlackConnectResponse }

func (r slackConnectResult) String() string {
	return "Open this URL to install the agent's Slack app: " + r.InstallURL
}

type slackStatusResult struct {
	*api.AgentSuccessResponse
	text string
}

func (r slackStatusResult) String() string { return r.text }

func newSlackCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "slack", Short: "Manage the agent's own Slack app"}
	cmd.AddCommand(
		newSlackConnectCmd(),
		newSlackStatusCmd("pause", "Silence the agent's Slack app", "Slack paused.", func(c *api.Client, id string) (*api.AgentSuccessResponse, error) {
			return c.PauseAgentSlack(id)
		}),
		newSlackStatusCmd("resume", "Re-enable a paused Slack app", "Slack resumed.", func(c *api.Client, id string) (*api.AgentSuccessResponse, error) {
			return c.ResumeAgentSlack(id)
		}),
		newSlackDeleteCmd(),
	)
	return cmd
}

func newSlackConnectCmd() *cobra.Command {
	var id string
	cmd := &cobra.Command{Use: "connect", Short: "Provision the agent's Slack app and print its install URL", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		agentID, err := resolveAgentID(id)
		if err != nil {
			return err
		}
		result, err := singletons.GetAPIClient().ConnectAgentSlack(agentID)
		if err != nil {
			return err
		}
		return target.Output(cmd, slackConnectResult{result})
	}}
	cmd.Flags().StringVar(&id, "id", "", "Agent ID (defaults to the agent folder's)")
	return cmd
}

func newSlackStatusCmd(use, short, text string, call func(*api.Client, string) (*api.AgentSuccessResponse, error)) *cobra.Command {
	var id string
	cmd := &cobra.Command{Use: use, Short: short, Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		agentID, err := resolveAgentID(id)
		if err != nil {
			return err
		}
		result, err := call(singletons.GetAPIClient(), agentID)
		if err != nil {
			return err
		}
		return target.Output(cmd, slackStatusResult{AgentSuccessResponse: result, text: text})
	}}
	cmd.Flags().StringVar(&id, "id", "", "Agent ID (defaults to the agent folder's)")
	return cmd
}

func newSlackDeleteCmd() *cobra.Command {
	var id string
	var yes bool
	cmd := &cobra.Command{Use: "delete", Short: "Permanently delete the agent's Slack app", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		agentID, err := resolveAgentID(id)
		if err != nil {
			return err
		}
		if !yes {
			if utils.IsNonInteractive(cmd) {
				return fmt.Errorf("major agent slack delete requires --yes in non-interactive mode")
			}
			var confirmed bool
			if err := huh.NewConfirm().Title("Delete this agent's Slack app? This can't be undone.").Value(&confirmed).Run(); err != nil {
				return err
			}
			if !confirmed {
				return fmt.Errorf("delete cancelled")
			}
		}
		result, err := singletons.GetAPIClient().DeleteAgentSlack(agentID)
		if err != nil {
			return err
		}
		return target.Output(cmd, slackStatusResult{AgentSuccessResponse: result, text: "Slack app deleted."})
	}}
	cmd.Flags().StringVar(&id, "id", "", "Agent ID (defaults to the agent folder's)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation")
	return cmd
}
