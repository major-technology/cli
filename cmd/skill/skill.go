package skill

import (
	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/cmd/agent"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{Use: "skill", Short: "Manage skills", Args: cobra.NoArgs}

func output(c *cobra.Command, v api.Record, e error) error {
	if e != nil {
		return e
	}
	return utils.WriteJSON(c, v)
}
func init() {
	l := &cobra.Command{Use: "list", Args: cobra.NoArgs, RunE: func(c *cobra.Command, _ []string) error {
		a, _ := c.Flags().GetBool("editable")
		p, _ := c.Flags().GetBool("published")
		v, e := singletons.GetAPIClient().ListSkills(a, p)
		return output(c, v, e)
	}}
	l.Flags().Bool("editable", false, "Only editable skills")
	l.Flags().Bool("published", false, "Only published skills")
	Cmd.AddCommand(l)
	g := &cobra.Command{Use: "get [skill-id]", Args: cobra.MaximumNArgs(1), RunE: func(c *cobra.Command, a []string) error {
		x := ""
		if len(a) > 0 {
			x = a[0]
		}
		id, e := agent.ResolveSkillID(x)
		if e != nil {
			return e
		}
		v, e := singletons.GetAPIClient().GetSkill(id)
		return output(c, v, e)
	}}
	Cmd.AddCommand(g)
	create := &cobra.Command{Use: "create", Args: cobra.NoArgs, RunE: func(c *cobra.Command, _ []string) error {
		v, e := singletons.GetAPIClient().CreateSkill()
		return output(c, v, e)
	}}
	Cmd.AddCommand(create)
	for _, c := range []*cobra.Command{l, g, create} {
		c.Flags().Bool("json", false, "Output route fields as JSON")
	}
}
