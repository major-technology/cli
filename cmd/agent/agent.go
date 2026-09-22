package agent

import (
	"errors"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/clients/workspace"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{Use: "agent", Short: "Manage agents", Args: cobra.NoArgs}

func target(id, kind string) (string, error) {
	if id != "" {
		return id, nil
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	cfg, err := workspace.Load(dir)
	if err != nil {
		if errors.Is(err, workspace.ErrNotFound) {
			return "", fmt.Errorf("%s ID required outside a %s workspace", kind, kind)
		}
		return "", err
	}
	if cfg.Target.Kind != kind {
		return "", fmt.Errorf("%s ID required: workspace target is %s", kind, cfg.Target.Kind)
	}
	if kind == "agent" {
		return cfg.Target.AgentID, nil
	}
	return cfg.Target.SkillID, nil
}
func command(use string, min, max int, invoke func(*cobra.Command, []string) (api.Record, error)) *cobra.Command {
	c := &cobra.Command{Use: use, Args: cobra.RangeArgs(min, max), RunE: func(cmd *cobra.Command, args []string) error {
		v, e := invoke(cmd, args)
		if e != nil {
			return e
		}
		if j, _ := cmd.Flags().GetBool("json"); j {
			return utils.WriteJSON(cmd, v)
		}
		printResult(cmd, v)
		return nil
	}}
	c.Flags().Bool("json", false, "Output route fields as JSON")
	return c
}
func printResult(cmd *cobra.Command, v api.Record) {
	if runs, ok := v["runs"].([]any); ok {
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "Run ID\tAgent ID\tTitle\tStarted by\tStatus")
		for _, item := range runs {
			r, _ := item.(map[string]any)
			user := "—"
			if u, ok := r["user"].(map[string]any); ok {
				user = fmt.Sprint(u["name"])
			}
			fmt.Fprintf(w, "%v\t%v\t%v\t%s\t%v\n", r["threadId"], r["agentId"], r["title"], user, r["status"])
		}
		w.Flush()
		return
	}
	utils.WriteJSON(cmd, v)
}
func id(args []string) (string, error) {
	s := ""
	if len(args) > 0 {
		s = args[0]
	}
	return target(s, "agent")
}
func init() {
	list := command("list", 0, 0, func(c *cobra.Command, _ []string) (api.Record, error) {
		b, _ := c.Flags().GetBool("editable")
		return singletons.GetAPIClient().ListAgents(b)
	})
	list.Flags().Bool("editable", false, "Only editable agents")
	Cmd.AddCommand(list)
	get := command("get [agent-id]", 0, 1, func(_ *cobra.Command, a []string) (api.Record, error) {
		x, e := id(a)
		if e != nil {
			return nil, e
		}
		return singletons.GetAPIClient().GetAgent(x)
	})
	Cmd.AddCommand(get)
	create := command("create", 0, 0, func(c *cobra.Command, _ []string) (api.Record, error) {
		n, _ := c.Flags().GetString("name")
		d, _ := c.Flags().GetString("description")
		return singletons.GetAPIClient().CreateAgent(n, d)
	})
	create.Flags().String("name", "", "Agent name")
	create.MarkFlagRequired("name")
	create.Flags().String("description", "", "Description")
	Cmd.AddCommand(create)
	run := command("run [agent-id]", 0, 1, func(c *cobra.Command, a []string) (api.Record, error) {
		x, e := id(a)
		if e != nil {
			return nil, e
		}
		p, _ := c.Flags().GetString("prompt")
		n, _ := c.Flags().GetString("name")
		return singletons.GetAPIClient().StartAgentRun(x, p, n)
	})
	run.Flags().String("prompt", "", "Run prompt")
	run.MarkFlagRequired("prompt")
	run.Flags().String("name", "", "Run title")
	Cmd.AddCommand(run)
	runs := &cobra.Command{Use: "runs", Short: "Manage independent agent runs"}
	Cmd.AddCommand(runs)
	rl := command("list", 0, 0, func(c *cobra.Command, _ []string) (api.Record, error) {
		a, _ := c.Flags().GetString("agent")
		all, _ := c.Flags().GetBool("all-users")
		return singletons.GetAPIClient().ListAgentRuns(a, all)
	})
	rl.Flags().String("agent", "", "Filter by agent ID")
	rl.Flags().Bool("all-users", false, "Include editable agents' runs started by others")
	runs.AddCommand(rl)
	content := command("content <run-id>", 1, 1, func(c *cobra.Command, a []string) (api.Record, error) {
		n, _ := c.Flags().GetInt("limit")
		if n < 0 || (c.Flags().Changed("limit") && n == 0) {
			return nil, fmt.Errorf("limit must be positive")
		}
		return singletons.GetAPIClient().GetAgentRunContent(a[0], n)
	})
	content.Flags().Int("limit", 0, "Maximum messages")
	runs.AddCommand(content)
	send := command("send <run-id>", 1, 1, func(c *cobra.Command, a []string) (api.Record, error) {
		m, _ := c.Flags().GetString("message")
		return singletons.GetAPIClient().SendAgentRunMessage(a[0], m)
	})
	send.Flags().String("message", "", "Message")
	send.MarkFlagRequired("message")
	runs.AddCommand(send)
	runs.AddCommand(command("stop <run-id>", 1, 1, func(_ *cobra.Command, a []string) (api.Record, error) {
		return singletons.GetAPIClient().StopAgentRun(a[0])
	}))
	channel := &cobra.Command{Use: "channel", Short: "Manage Slack channel"}
	Cmd.AddCommand(channel)
	for _, action := range []string{"connect", "pause", "resume", "delete"} {
		action := action
		c := command(action+" [agent-id]", 0, 1, func(c *cobra.Command, a []string) (api.Record, error) {
			typ, _ := c.Flags().GetString("type")
			if typ != "slack" {
				return nil, fmt.Errorf("only slack is supported")
			}
			x, e := id(a)
			if e != nil {
				return nil, e
			}
			return singletons.GetAPIClient().AgentChannel(x, action)
		})
		c.Flags().String("type", "", "Channel type (slack)")
		c.MarkFlagRequired("type")
		channel.AddCommand(c)
	}
	permissions := &cobra.Command{Use: "permissions", Short: "Inspect published permissions"}
	Cmd.AddCommand(permissions)
	for _, kind := range []string{"resource", "app"} {
		kind := kind
		c := command(kind+" [agent-id] <"+kind+"-id>", 1, 2, func(_ *cobra.Command, a []string) (api.Record, error) {
			agentID := ""
			other := a[0]
			if len(a) == 2 {
				agentID = a[0]
				other = a[1]
			}
			x, e := target(agentID, "agent")
			if e != nil {
				return nil, e
			}
			if kind == "resource" {
				return singletons.GetAPIClient().AgentResourcePermissions(x, other)
			}
			return singletons.GetAPIClient().AgentAppPermissions(x, other)
		})
		permissions.AddCommand(c)
	}
}
