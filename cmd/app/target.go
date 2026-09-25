package app

import (
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/major-technology/cli/clients/git"
	"github.com/major-technology/cli/cmd/target"
	clierrors "github.com/major-technology/cli/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// appTarget serves `major pull|push|validate|publish` for an app workspace:
// git for the files and the existing deploy code for publish. It never forces.
type appTarget struct{}

func init() {
	target.Register("app", appTarget{})
}

type appGitResult struct {
	ApplicationID string `json:"applicationId"`
	Commit        string `json:"commit"`
	action        string
}

func (r appGitResult) String() string {
	return fmt.Sprintf("%s %s.", r.action, shortSHA(r.Commit))
}

type appValidateResult struct {
	Valid bool `json:"valid"`
}

func (appValidateResult) String() string { return "Lint passed." }

type appPublishResult struct {
	VersionID   string `json:"versionId"`
	VersionHash string `json:"versionHash"`
	Status      string `json:"status"`
	AppURL      string `json:"appUrl,omitempty"`
}

func (r appPublishResult) String() string {
	if r.AppURL == "" {
		return fmt.Sprintf("Deployed %s.", shortSHA(r.VersionHash))
	}
	return fmt.Sprintf("Deployed %s: %s", shortSHA(r.VersionHash), r.AppURL)
}

func (appTarget) AddFlags(action string, flags *pflag.FlagSet) {
	if action == "publish" {
		flags.String("slug", "", "URL slug for the first deploy of an app (ignored once the app has one)")
	}
}

func (appTarget) PublishPrompt() string {
	branch, err := git.RemoteDefaultBranch()
	if err != nil {
		return "Deploy the latest commit on the default branch?"
	}
	return fmt.Sprintf("Deploy the latest commit on %s?", branch)
}

// Pull runs `git pull --no-rebase`. A conflict leaves git's markers in place
// and returns git's output.
func (appTarget) Pull(ctx *target.TargetContext) (any, error) {
	if err := git.PullMerge(ctx.Root); err != nil {
		return nil, err
	}
	head, err := git.HeadSHA()
	if err != nil {
		return nil, err
	}
	return appGitResult{ApplicationID: ctx.Config.Target.ApplicationID, Commit: head, action: "Pulled"}, nil
}

// Push commits a dirty tree with -m, then pushes HEAD to the default branch
// with an ordinary push.
func (appTarget) Push(ctx *target.TargetContext) (any, error) {
	branch, err := git.RequireDefaultBranch()
	if err != nil {
		return nil, err
	}
	dirty, err := git.HasUncommittedChanges()
	if err != nil {
		return nil, err
	}
	if dirty {
		if strings.TrimSpace(ctx.Notes) == "" {
			return nil, fmt.Errorf("uncommitted changes: pass -m <message> to commit them")
		}
		if err := git.AddAll(); err != nil {
			return nil, err
		}
		if err := git.Commit(ctx.Notes); err != nil {
			return nil, clierrors.WrapError("failed to commit changes", err)
		}
	}
	if err := git.PushHead(branch); err != nil {
		return nil, fmt.Errorf("%w\nrun major pull, resolve conflicts, then major push", err)
	}
	head, err := git.HeadSHA()
	if err != nil {
		return nil, err
	}
	return appGitResult{ApplicationID: ctx.Config.Target.ApplicationID, Commit: head, action: "Pushed"}, nil
}

// Validate runs the app's lint script. Lint's output passes through and its
// exit code becomes the command's.
func (appTarget) Validate(ctx *target.TargetContext) (any, error) {
	data, err := os.ReadFile(filepath.Join(ctx.Root, "package.json"))
	if err != nil {
		return nil, fmt.Errorf("read package.json: %w", err)
	}
	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("parse package.json: %w", err)
	}
	if _, ok := pkg.Scripts["lint"]; !ok {
		return nil, fmt.Errorf("package.json has no lint script")
	}
	lint := exec.Command("pnpm", "run", "lint")
	lint.Dir = ctx.Root
	lint.Stdout = lintOutput(ctx.Command)
	lint.Stderr = ctx.Command.ErrOrStderr()
	if err := lint.Run(); err != nil {
		var exitErr *exec.ExitError
		if stderrors.As(err, &exitErr) {
			return nil, &clierrors.ExitCodeError{Code: exitErr.ExitCode(), Err: fmt.Errorf("lint failed with exit code %d", exitErr.ExitCode())}
		}
		return nil, fmt.Errorf("run pnpm run lint: %w", err)
	}
	return appValidateResult{Valid: true}, nil
}

// lintOutput keeps stdout for the one JSON result when --json is set.
func lintOutput(cmd *cobra.Command) io.Writer {
	if jsonOut, _ := cmd.Flags().GetBool("json"); jsonOut {
		return cmd.ErrOrStderr()
	}
	return cmd.OutOrStdout()
}

// Publish deploys the pushed default branch, the body of `major app deploy`
// after its push step, and waits for the result.
func (appTarget) Publish(ctx *target.TargetContext) (any, error) {
	branch, err := git.RequireDefaultBranch()
	if err != nil {
		return nil, err
	}
	state, err := git.RemoteHeadState(branch)
	if err != nil {
		return nil, err
	}
	switch state {
	case "unpushed":
		return nil, fmt.Errorf("local HEAD is not on origin/%s; run major push first", branch)
	case "behind":
		return nil, fmt.Errorf("origin/%s has commits you do not have; run major pull, then major publish", branch)
	}
	applicationID, organizationID, slug, err := getApplicationAndOrgIDFromDir(ctx.Root)
	if err != nil {
		return nil, clierrors.WrapError("failed to get application ID", err)
	}
	flagSlug, _ := ctx.Command.Flags().GetString("slug")
	slug, err = resolveDeploySlug(slug, flagSlug, func() (string, error) {
		return "", fmt.Errorf("this app has never been deployed; run major publish --slug <slug> to choose its URL")
	})
	if err != nil {
		return nil, err
	}
	head, err := git.HeadSHA()
	if err != nil {
		return nil, err
	}
	resp, statusHint, err := createPushedVersion(applicationID, slug, head)
	if err != nil {
		return nil, err
	}
	logf := func(format string, args ...any) { fmt.Fprintf(ctx.Command.ErrOrStderr(), format, args...) }
	status, deploymentError, appURL, err := pollDeploymentStatusSimple(logf, applicationID, organizationID, resp.VersionID)
	if err != nil {
		return nil, fmt.Errorf("failed to track deployment status: %w. Inspect with: %s", err, statusHint)
	}
	if status != "DEPLOYED" {
		return nil, deploymentFailedError(status, deploymentError)
	}
	return appPublishResult{VersionID: resp.VersionID, VersionHash: resp.VersionHash, Status: status, AppURL: appURL}, nil
}

func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}
