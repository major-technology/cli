package projects

// bindings.json parsing plus the compile-time reference validation that hangs
// off it. bindings.json is the human-authored slot -> platform-id manifest at
// the project root, next to project.json: agents reference platform resources,
// applications, and skills by author-chosen slot names, and this file is where
// those slots get their concrete ids. Compile only validates that every
// reference resolves - it never bakes ids into agent configs; deploy resolves
// slots to ids server-side.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// BindingsFileName is the project-root manifest compile reads slots from.
const BindingsFileName = "bindings.json"

// rawBindings mirrors bindings.json. The schema (schemas/bindings.schema.json)
// enforces slot-name charset, uuid format, and non-empty resource types; this
// only has to carry the values through.
type rawBindings struct {
	Slots BindingSlots `json:"slots"`
}

// loadBindings reads and validates <dir>/bindings.json. A missing file is not
// an error (the manifest is optional): it returns nil bindings and no issues,
// and reference validation then tells the author to add the file only if some
// agent actually references a slot.
func loadBindings(dir string) (*BindingSlots, []Issue) {
	path := filepath.Join(dir, BindingsFileName)

	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, []Issue{{File: BindingsFileName, Message: "cannot read " + BindingsFileName + ": " + err.Error()}}
	}

	if issues := validateAgainst(BindingsSchema(), raw, BindingsFileName); len(issues) > 0 {
		return nil, issues
	}

	var parsed rawBindings
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, []Issue{{File: BindingsFileName, Message: "invalid JSON: " + err.Error()}}
	}

	return &parsed.Slots, nil
}

// checkSkillSlotCollisions rejects a bindings skills slot whose name is also a
// project-local skill slug. The server resolves the skill namespace by merging
// bindings slots with project skill slugs and letting the local slug win, so a
// collision would silently ignore the binding. Fail the compile instead and
// tell the author to rename the slot.
func checkSkillSlotCollisions(bindings *BindingSlots, skillSlugs map[string]bool) []Issue {
	if bindings == nil {
		return nil
	}

	var issues []Issue

	for _, slot := range idSlotNames(bindings.Skills) {
		if !skillSlugs[slot] {
			continue
		}

		issues = append(issues, Issue{
			File:    BindingsFileName,
			Path:    "/slots/skills/" + slot,
			Message: fmt.Sprintf("skills slot %q collides with the project skill of the same slug; rename the binding slot, because a project-local skill always wins over a bound platform skill", slot),
		})
	}

	return issues
}

// validateAgentReferences checks that every slot an agent references exists in
// the project's bindings, and that every bare skill string names a skill
// directory compiled in this run. bindings is nil when the project has no
// bindings.json, in which case any slot reference is reported as a missing
// manifest rather than a missing slot. srcDir only names the directory a
// missing project-local skill would have to live in.
func validateAgentReferences(agent AgentDefinition, bindings *BindingSlots, skillSlugs map[string]bool, srcDir string) []Issue {
	agentFile := filepath.Join(agent.Dir, "agent.json")

	var issues []Issue

	for i, connector := range agent.Connectors {
		path := fmt.Sprintf("/connectors/%d/slot", i)

		if issue := checkSlot(agentFile, path, connector.Slot, "resources", bindings, resourceSlotNames(bindings)); issue != nil {
			issues = append(issues, *issue)
		}
	}

	for i, app := range agent.Apps {
		path := fmt.Sprintf("/apps/%d/slot", i)

		if issue := checkSlot(agentFile, path, app.Slot, "applications", bindings, idSlotNames(applicationBindings(bindings))); issue != nil {
			issues = append(issues, *issue)
		}
	}

	for i, skill := range agent.Skills {
		if skill.Slot != "" {
			path := fmt.Sprintf("/skills/%d/slot", i)

			if issue := checkSlot(agentFile, path, skill.Slot, "skills", bindings, idSlotNames(skillBindings(bindings))); issue != nil {
				issues = append(issues, *issue)
			}

			continue
		}

		if !skillSlugs[skill.Slug] {
			issues = append(issues, Issue{
				File:    agentFile,
				Path:    fmt.Sprintf("/skills/%d", i),
				Message: fmt.Sprintf("skill %q is not a skill in this project; add %s/SKILL.md, or reference a platform skill with {\"slot\": ...}", skill.Slug, filepath.ToSlash(filepath.Join(srcDir, "skills", skill.Slug))),
			})
		}
	}

	return issues
}

// checkSlot reports the one right issue for a slot reference: the manifest is
// missing entirely, or the slot is absent from the given kind.
func checkSlot(agentFile, path, slot, kind string, bindings *BindingSlots, known []string) *Issue {
	if bindings == nil {
		return &Issue{
			File:    agentFile,
			Path:    path,
			Message: fmt.Sprintf("references binding slot %q but this project has no %s; add %s at the project root with slots.%s.%s", slot, BindingsFileName, BindingsFileName, kind, slot),
		}
	}

	for _, name := range known {
		if name == slot {
			return nil
		}
	}

	return &Issue{
		File:    agentFile,
		Path:    path,
		Message: fmt.Sprintf("unknown binding slot %q: %s declares no slots.%s.%s (declared: %s)", slot, BindingsFileName, kind, slot, describeSlots(known)),
	}
}

func describeSlots(names []string) string {
	if len(names) == 0 {
		return "none"
	}

	out := ""
	for i, name := range names {
		if i > 0 {
			out += ", "
		}
		out += fmt.Sprintf("%q", name)
	}

	return out
}

func resourceSlotNames(bindings *BindingSlots) []string {
	if bindings == nil {
		return nil
	}

	names := make([]string, 0, len(bindings.Resources))
	for name := range bindings.Resources {
		names = append(names, name)
	}
	sort.Strings(names)

	return names
}

func applicationBindings(bindings *BindingSlots) map[string]IDBinding {
	if bindings == nil {
		return nil
	}

	return bindings.Applications
}

func skillBindings(bindings *BindingSlots) map[string]IDBinding {
	if bindings == nil {
		return nil
	}

	return bindings.Skills
}

func idSlotNames(entries map[string]IDBinding) []string {
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	sort.Strings(names)

	return names
}
