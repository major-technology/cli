package target

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/major-technology/cli/clients/workspace"
)

// Nester is an optional Target method for a kind whose workspace holds other
// workspaces. It maps a directory under the root to the kind of workspace each
// of its subfolders is, e.g. a workflow's "functions" holds function workspaces.
type Nester interface {
	NestedDirs() map[string]string
}

// NestedResult is one nested workspace's outcome within its parent's result.
type NestedResult struct {
	Kind   string `json:"kind"`
	ID     string `json:"id"`
	Path   string `json:"path"`
	Result any    `json:"result"`
}

// withNested is a parent's result with its nested workspaces' results.
type withNested struct {
	parent any
	nested []NestedResult
}

func (r withNested) String() string {
	lines := make([]string, 0, len(r.nested)+1)
	for _, item := range r.nested {
		lines = append(lines, fmt.Sprintf("%s: %v", item.Path, item.Result))
	}
	lines = append(lines, fmt.Sprint(r.parent))
	return strings.Join(lines, "\n")
}

// MarshalJSON is the parent's own JSON object with a "nested" list added.
func (r withNested) MarshalJSON() ([]byte, error) {
	parent, err := json.Marshal(r.parent)
	if err != nil {
		return nil, err
	}
	fields := map[string]json.RawMessage{}
	if err := json.Unmarshal(parent, &fields); err != nil {
		return nil, err
	}
	if fields["nested"], err = json.Marshal(r.nested); err != nil {
		return nil, err
	}
	return json.Marshal(fields)
}

type nestedWorkspace struct {
	rel    string
	root   string
	config *workspace.Config
	target Target
}

// findNested lists the workspaces under each of the target's nested
// directories. Every subfolder there must be a workspace of the declared kind.
func findNested(root string, target Target) ([]nestedWorkspace, error) {
	nester, ok := target.(Nester)
	if !ok {
		return nil, nil
	}
	var found []nestedWorkspace
	for dir, kind := range nester.NestedDirs() {
		entries, err := os.ReadDir(filepath.Join(root, dir))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if !entry.IsDir() || SkippedDirs[entry.Name()] {
				continue
			}
			rel := dir + "/" + entry.Name()
			sub := filepath.Join(root, dir, entry.Name())
			notWorkspace := &InvalidBundleError{Err: fmt.Errorf("%s is not a %s workspace; run major %s create %s", rel, kind, kind, rel)}
			subRoot, cfg, err := workspace.Locate(sub)
			if errors.Is(err, workspace.ErrNotFound) || (err == nil && subRoot != sub) {
				return nil, notWorkspace
			}
			if err != nil {
				return nil, err
			}
			if cfg.Target.Kind != kind {
				return nil, notWorkspace
			}
			found = append(found, nestedWorkspace{rel: rel, root: sub, config: cfg, target: registry[kind]})
		}
	}
	sort.Slice(found, func(i, j int) bool { return found[i].rel < found[j].rel })
	return found, nil
}

// runWithNested runs a push, validate, or publish on every nested workspace in
// parallel, then on the workspace itself once all of them succeed. A pull does
// not descend: the parent's own bundle restores the nested workspaces.
func runWithNested(ctx *TargetContext, target Target, action string) (any, error) {
	if action == "pull" {
		return runAction(ctx, target, action)
	}
	nested, err := findNested(ctx.Root, target)
	if err != nil {
		return nil, err
	}
	if len(nested) == 0 {
		return runAction(ctx, target, action)
	}
	results := make([]NestedResult, len(nested))
	failures := make([]error, len(nested))
	var wg sync.WaitGroup
	for i, item := range nested {
		wg.Add(1)
		go func() {
			defer wg.Done()
			child := &TargetContext{Root: item.root, Config: item.config, API: ctx.API, Command: ctx.Command, Notes: ctx.Notes}
			result, err := runAction(child, item.target, action)
			if err != nil {
				failures[i] = fmt.Errorf("%s: %w", item.rel, err)
				return
			}
			results[i] = NestedResult{Kind: item.config.Target.Kind, ID: item.config.Target.ID(), Path: item.rel, Result: result}
		}()
	}
	wg.Wait()
	if err := errors.Join(failures...); err != nil {
		return nil, err
	}
	parent, err := runAction(ctx, target, action)
	if err != nil {
		return nil, err
	}
	return withNested{parent: parent, nested: results}, nil
}
