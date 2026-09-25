package skill

import (
	"fmt"
	"path"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/clients/workspace"
	"github.com/major-technology/cli/cmd/target"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{Use: "skill", Short: "Manage file-based skills"}

// bundle is the skill kind: every regular file under the root, less the skip list.
var bundle = target.BundleTarget{
	Kind:             "skill",
	APIPath:          "skills",
	Files:            func(string) target.FileSet { return skillFiles },
	Pack:             target.PackZip,
	ValidateByUpload: true,
}

// skippedNames, with target.SkippedDirs, match the skill pod: never bundle content.
var skippedNames = map[string]bool{".DS_Store": true, "Thumbs.db": true}

var skillFiles = target.FileSet{
	Owns: func(rel string) bool {
		// The pod scaffolds a root tsconfig.json for its editor and never saves it.
		if rel == "tsconfig.json" || skippedNames[path.Base(rel)] {
			return false
		}
		for _, segment := range strings.Split(path.Dir(rel), "/") {
			if target.SkippedDirs[segment] {
				return false
			}
		}
		return true
	},
	Check: func(rels []string) error {
		for _, rel := range rels {
			if rel == "SKILL.md" {
				return nil
			}
		}
		return fmt.Errorf("skill workspace requires SKILL.md at the root")
	},
}

func init() {
	Cmd.PersistentFlags().Bool("json", false, "Print one JSON result")
	Cmd.AddCommand(newListCmd(), newCreateCmd(), newCloneCmd())
	target.Register("skill", bundle)
}

type listResult struct{ *api.SkillListResponse }

func (r listResult) String() string {
	if len(r.Skills) == 0 {
		return "No skills found."
	}
	lines := make([]string, 0, len(r.Skills))
	for _, item := range r.Skills {
		lines = append(lines, fmt.Sprintf("%s  %s  (%s)", item.SkillID, skillName(item), item.Status))
	}
	return strings.Join(lines, "\n")
}

// skillName is the slug, which a never-saved draft does not have yet.
func skillName(item api.SkillItem) string {
	if item.Slug == nil || *item.Slug == "" {
		return "(unnamed draft)"
	}
	return *item.Slug
}

func newListCmd() *cobra.Command {
	var readOnly bool
	cmd := &cobra.Command{Use: "list", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		org, err := target.OrganizationID()
		if err != nil {
			return err
		}
		result, err := singletons.GetAPIClient().ListSkills(org, readOnly)
		if err != nil {
			return err
		}
		return target.Output(cmd, listResult{result})
	}}
	cmd.Flags().BoolVar(&readOnly, "include-read-only", false, "Include skills you can see but not edit")
	return cmd
}

// newCreateCmd takes no name: a skill's name comes from SKILL.md frontmatter.
func newCreateCmd() *cobra.Command {
	return &cobra.Command{Use: "create <dir>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := target.RequireNewDir(args[0])
		if err != nil {
			return err
		}
		org, err := target.OrganizationID()
		if err != nil {
			return err
		}
		created, err := singletons.GetAPIClient().CreateSkill(org)
		if err != nil {
			return err
		}
		return bundle.Clone(cmd, org, workspace.Target{Kind: "skill", SkillID: created.SkillID}, dir)
	}}
}
func newCloneCmd() *cobra.Command {
	var id, dir string
	cmd := &cobra.Command{Use: "clone", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		org, err := target.OrganizationID()
		if err != nil {
			return err
		}
		if id == "" {
			if utils.IsNonInteractive(cmd) {
				return fmt.Errorf("major skill clone requires --id in non-interactive mode")
			}
			list, err := singletons.GetAPIClient().ListSkills(org, false)
			if err != nil {
				return err
			}
			if len(list.Skills) == 0 {
				return fmt.Errorf("no editable skills in this organization")
			}
			options := make([]huh.Option[string], 0, len(list.Skills))
			for _, item := range list.Skills {
				options = append(options, huh.NewOption(skillName(item)+" ("+item.SkillID+")", item.SkillID))
			}
			if err := huh.NewSelect[string]().Title("Choose a skill").Options(options...).Value(&id).Run(); err != nil {
				return err
			}
		}
		if dir == "" {
			dir = id
		}
		abs, err := target.RequireNewDir(dir)
		if err != nil {
			return err
		}
		return bundle.Clone(cmd, org, workspace.Target{Kind: "skill", SkillID: id}, abs)
	}}
	cmd.Flags().StringVar(&id, "id", "", "Skill ID")
	cmd.Flags().StringVar(&dir, "dir", "", "New directory (defaults to skill ID)")
	return cmd
}
