package workflow

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/cmd/target"
	"github.com/major-technology/cli/singletons"
	"github.com/spf13/cobra"
)

type eventsResult struct{ *api.WorkflowConnectorProvider }

func (r eventsResult) String() string {
	lines := []string{fmt.Sprintf("%s  %s", r.ConnectorType, r.Label)}
	for _, event := range r.Events {
		lines = append(lines, fmt.Sprintf("%s  %s  %s", event.Type, event.Label, event.Description))
	}
	return strings.Join(lines, "\n")
}

type eventResult struct{ *api.WorkflowConnectorEvent }

func (r eventResult) String() string {
	data, err := json.MarshalIndent(r.WorkflowConnectorEvent, "", "  ")
	if err != nil {
		return fmt.Sprint(r.WorkflowConnectorEvent)
	}
	return string(data)
}

func newEventsCmd() *cobra.Command {
	var event string
	cmd := &cobra.Command{Use: "events <connectorType>", Short: "List a connector's workflow event types, or show one event's schemas", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		provider, err := singletons.GetAPIClient().GetWorkflowConnectorEvents(args[0])
		if err != nil {
			return err
		}
		if event == "" {
			return target.Output(cmd, eventsResult{provider})
		}
		for i := range provider.Events {
			if provider.Events[i].Type == event {
				return target.Output(cmd, eventResult{&provider.Events[i]})
			}
		}
		return fmt.Errorf("connector event %s:%s is not supported", args[0], event)
	}}
	cmd.Flags().StringVar(&event, "event", "", "Show this event type's schemas")
	return cmd
}
