package agent

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/cmd/target"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

var runSources = []string{"web", "slack", "external_channel", "app", "schedule", "manual", "agent", "workflow"}

type runStartResult struct{ *api.AgentRunStartResponse }

func (r runStartResult) String() string {
	return fmt.Sprintf("Started run %s.\nFollow it with: major agent run content %s", r.RunID, r.RunID)
}

type runListResult struct {
	*api.AgentRunListResponse
	nextOffset int
}

func (r runListResult) String() string {
	if len(r.Runs) == 0 {
		return "No runs found."
	}
	lines := make([]string, 0, len(r.Runs)+1)
	for _, run := range r.Runs {
		lines = append(lines, fmt.Sprintf("%s  %s  %s  %s  %s", run.RunID, run.Status, run.Source, run.CreatedAt, run.Title))
	}
	if r.HasMore {
		lines = append(lines, fmt.Sprintf("More runs: pass --offset %d.", r.nextOffset))
	}
	return strings.Join(lines, "\n")
}

type runStatusResult struct {
	*api.AgentRunStatusResponse
	text string
}

func (r runStatusResult) String() string { return r.text }

type runContentResult struct{ *api.AgentRunMessagesResponse }

func (r runContentResult) String() string {
	lines := make([]string, 0, len(r.Messages)+1)
	for _, message := range r.Messages {
		lines = append(lines, fmt.Sprintf("%s  %s  %s  %s", message.Timestamp, message.Role, message.Type, compactJSON(message.Content)))
	}
	if r.NextToken != nil && *r.NextToken != "" {
		lines = append(lines, fmt.Sprintf("Older messages: pass --next-token '%s'.", strings.ReplaceAll(*r.NextToken, "'", `'\''`)))
	}
	return strings.Join(lines, "\n")
}

func compactJSON(content any) string {
	data, err := json.Marshal(content)
	if err != nil {
		return fmt.Sprint(content)
	}
	return string(data)
}

func newRunCmd() *cobra.Command {
	var id, prompt, name string
	cmd := &cobra.Command{Use: "run", Short: "Start a run of the agent's published version", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if prompt == "" {
			return fmt.Errorf("--prompt is required")
		}
		agentID, err := resolveAgentID(id)
		if err != nil {
			return err
		}
		result, err := singletons.GetAPIClient().StartAgentRun(agentID, prompt, name)
		if err != nil {
			return err
		}
		return target.Output(cmd, runStartResult{result})
	}}
	cmd.Flags().StringVar(&id, "id", "", "Agent ID (defaults to the agent folder's)")
	cmd.Flags().StringVarP(&prompt, "prompt", "p", "", "Initial prompt for the run")
	cmd.Flags().StringVar(&name, "name", "", "Run title shown in Major")
	cmd.AddCommand(newRunListCmd(), newRunSendCmd(), newRunStopCmd(), newRunContentCmd())
	return cmd
}

func newRunListCmd() *cobra.Command {
	var id string
	var filters api.AgentRunListFilters
	cmd := &cobra.Command{Use: "list", Short: "List the agent's runs, newest first", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if filters.Source != "" && !slices.Contains(runSources, filters.Source) {
			return fmt.Errorf("invalid --source %q: use one of %s", filters.Source, strings.Join(runSources, ", "))
		}
		agentID, err := resolveAgentID(id)
		if err != nil {
			return err
		}
		result, err := singletons.GetAPIClient().ListAgentRuns(agentID, filters)
		if err != nil {
			return err
		}
		return target.Output(cmd, runListResult{AgentRunListResponse: result, nextOffset: filters.Offset + filters.Limit})
	}}
	cmd.Flags().StringVar(&id, "id", "", "Agent ID (defaults to the agent folder's)")
	cmd.Flags().BoolVar(&filters.Mine, "mine", false, "Only runs you started")
	cmd.Flags().BoolVar(&filters.Live, "live", false, "Only runs with a live session")
	cmd.Flags().StringVar(&filters.Source, "source", "", "Only runs from this source: "+strings.Join(runSources, ", "))
	cmd.Flags().IntVar(&filters.Limit, "limit", 20, "Maximum runs to return")
	cmd.Flags().IntVar(&filters.Offset, "offset", 0, "Runs to skip")
	return cmd
}

func newRunSendCmd() *cobra.Command {
	var message string
	cmd := &cobra.Command{Use: "send <runId>", Short: "Send a message to a run, resuming it if it finished", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if message == "" {
			return fmt.Errorf("--message is required")
		}
		result, err := singletons.GetAPIClient().SendAgentRunMessage(args[0], message)
		if err != nil {
			return err
		}
		return target.Output(cmd, runStatusResult{AgentRunStatusResponse: result, text: "Message queued."})
	}}
	cmd.Flags().StringVarP(&message, "message", "m", "", "Message to send")
	return cmd
}

func newRunStopCmd() *cobra.Command {
	return &cobra.Command{Use: "stop <runId>", Short: "Stop a run", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		result, err := singletons.GetAPIClient().StopAgentRun(args[0])
		if err != nil {
			return err
		}
		return target.Output(cmd, runStatusResult{AgentRunStatusResponse: result, text: "Run stopped."})
	}}
}

func newRunContentCmd() *cobra.Command {
	var nextToken string
	cmd := &cobra.Command{Use: "content <runId>", Short: "Read a run's messages", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		result, err := singletons.GetAPIClient().GetAgentRunMessages(args[0], nextToken)
		if err != nil {
			return err
		}
		return target.Output(cmd, runContentResult{result})
	}}
	cmd.Flags().StringVar(&nextToken, "next-token", "", "Token from a previous page, for older messages")
	return cmd
}
