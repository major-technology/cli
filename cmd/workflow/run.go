package workflow

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/cmd/target"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

type runListResult struct{ *api.WorkflowRunListResponse }

func (r runListResult) String() string {
	if len(r.Runs) == 0 {
		return "No runs found."
	}
	lines := make([]string, 0, len(r.Runs))
	for _, run := range r.Runs {
		lines = append(lines, runLine(run))
	}
	return strings.Join(lines, "\n")
}

func runLine(run api.WorkflowRun) string {
	line := fmt.Sprintf("%s  %s  %s  v%d  %s", run.ID, run.Status, run.TriggerType, run.Version, run.StartedAt)
	if run.Error != nil && *run.Error != "" {
		line += "  " + *run.Error
	}
	return line
}

type runGetResult struct{ *api.WorkflowRunResponse }

func (r runGetResult) String() string {
	run := r.Run
	header := fmt.Sprintf("%s  %s  v%d", run.ID, run.Status, run.Version)
	if run.Error != nil && *run.Error != "" {
		header += "  " + *run.Error
	}
	if r.Document == nil {
		return header + "\nThe run has not started writing its trace yet."
	}
	executions := r.Document.Executions
	keys := make([]string, 0, len(executions))
	for key := range executions {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := executions[keys[i]], executions[keys[j]]
		if a.TriggeredAt != b.TriggeredAt {
			return a.TriggeredAt < b.TriggeredAt
		}
		return keys[i] < keys[j]
	})
	lines := []string{header}
	for _, key := range keys {
		execution := executions[key]
		line := fmt.Sprintf("%s  %s", key, execution.Status)
		if execution.Error != nil && *execution.Error != "" {
			line += "  " + *execution.Error
		}
		lines = append(lines, line)
		if execution.Input != nil {
			lines = append(lines, "  input: "+compactJSON(execution.Input))
		}
		if execution.Output != nil {
			lines = append(lines, "  output: "+compactJSON(execution.Output))
		}
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
	cmd := &cobra.Command{Use: "run", Short: "Inspect the workflow's runs"}
	cmd.AddCommand(newRunListCmd(), newRunGetCmd())
	return cmd
}

func newRunListCmd() *cobra.Command {
	var id string
	cmd := &cobra.Command{Use: "list", Short: "List the workflow's runs, newest first", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		workflowID, err := resolveWorkflowID(id)
		if err != nil {
			return err
		}
		result, err := singletons.GetAPIClient().ListWorkflowRuns(workflowID)
		if err != nil {
			return err
		}
		return target.Output(cmd, runListResult{result})
	}}
	cmd.Flags().StringVar(&id, "id", "", "Workflow ID (defaults to the workflow folder's)")
	return cmd
}

func newRunGetCmd() *cobra.Command {
	var id string
	cmd := &cobra.Command{Use: "get <runId>", Short: "Show a run's per-node trace", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		workflowID, err := resolveWorkflowID(id)
		if err != nil {
			return err
		}
		result, err := singletons.GetAPIClient().GetWorkflowRun(workflowID, args[0])
		if err != nil {
			return err
		}
		return target.Output(cmd, runGetResult{result})
	}}
	cmd.Flags().StringVar(&id, "id", "", "Workflow ID (defaults to the workflow folder's)")
	return cmd
}
