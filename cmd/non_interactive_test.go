package cmd

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/major-technology/cli/clients/api"
	mjrToken "github.com/major-technology/cli/clients/token"
	"github.com/major-technology/cli/clients/workspace"
	"github.com/major-technology/cli/singletons"
	"github.com/major-technology/cli/utils"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

const (
	tableAppID = "11111111-1111-4111-8111-111111111111"
	tableOrgID = "22222222-2222-4222-8222-222222222222"
)

func TestRootHelpRegistersNonInteractiveFlag(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("non-interactive")
	if flag == nil {
		t.Fatal("root must inherit --non-interactive")
	}
	if flag.DefValue != "false" {
		t.Fatalf("default = %q, want false (must not imply --yes)", flag.DefValue)
	}
}

func TestLoginNonInteractiveFailsBeforeDeviceFlow(t *testing.T) {
	keyring.MockInit()
	t.Setenv("MAJOR_TOKEN", "")

	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		t.Errorf("unexpected HTTP call: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("MAJOR_API_URL", srv.URL+"/cli")

	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(errBuf)
	rootCmd.SetIn(strings.NewReader(""))
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetIn(nil)
		rootCmd.SetArgs(nil)
	})

	rootCmd.SetArgs([]string{"--non-interactive", "user", "login"})
	err := runWithDeadline(t, 8*time.Second, rootCmd.Execute)
	if err == nil {
		t.Fatal("non-interactive login must fail")
	}
	msg := err.Error() + outBuf.String() + errBuf.String()
	if !strings.Contains(msg, "MAJOR_TOKEN") {
		t.Fatalf("error must name MAJOR_TOKEN, got %v stdout=%q stderr=%q", err, outBuf.String(), errBuf.String())
	}
	if !strings.Contains(strings.ToLower(msg), "login") {
		t.Fatalf("error must mention prior human login, got %v", err)
	}
	if got := requests.Load(); got != 0 {
		t.Fatalf("HTTP calls = %d, want 0 (must not start device login)", got)
	}
}

func TestDocsNonInteractivePrintsURLWithoutOpening(t *testing.T) {
	orig := utils.BrowserStart
	opened := []string{}
	utils.BrowserStart = func(url string) error {
		opened = append(opened, url)
		return nil
	}
	t.Cleanup(func() { utils.BrowserStart = orig })

	outBuf := &bytes.Buffer{}
	docsCmd.SetOut(outBuf)
	docsCmd.SetErr(outBuf)
	docsCmd.SetIn(strings.NewReader(""))
	t.Cleanup(func() {
		docsCmd.SetOut(nil)
		docsCmd.SetErr(nil)
		docsCmd.SetIn(nil)
	})

	err := runWithDeadline(t, 8*time.Second, func() error {
		return docsCmd.RunE(docsCmd, nil)
	})
	if err != nil {
		t.Fatalf("docs in non-interactive mode: %v", err)
	}
	if len(opened) != 0 {
		t.Fatalf("opened browser: %v", opened)
	}
	if !strings.Contains(outBuf.String(), "https://docs.major.build/") {
		t.Fatalf("output must print destination, got %q", outBuf.String())
	}
}

func TestCreateHelpShowsCompleteNonInteractiveInvocation(t *testing.T) {
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})
	rootCmd.SetArgs([]string{"app", "create", "--help"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	help := out.String()
	for _, want := range []string{"--name", "--description", "--theme-id", "--github-user"} {
		if !strings.Contains(help, want) {
			t.Errorf("app create help missing %q\n%s", want, help)
		}
	}
}

func TestNonInteractiveIncompleteChoicesRefuseWithoutMutation(t *testing.T) {
	type choiceCase struct {
		name  string
		path  []string
		args  []string
		want  []string
		setup func(t *testing.T, cmd *cobra.Command) *atomic.Int32
	}

	cases := []choiceCase{
		{
			name: "app create missing name and description",
			path: []string{"app", "create"},
			want: []string{"--name", "--description"},
			setup: func(t *testing.T, cmd *cobra.Command) *atomic.Int32 {
				keyring.MockInit()
				if err := mjrToken.StoreDefaultOrg(tableOrgID, "Test Org"); err != nil {
					t.Fatal(err)
				}
				t.Setenv("MAJOR_TOKEN", "test-injected-token")
				_ = cmd.Flags().Set("name", "")
				_ = cmd.Flags().Set("description", "")
				_ = cmd.Flags().Set("theme-id", "")
				var writes atomic.Int32
				restoreAPIClient(t, mutationCountingServer(t, &writes, func(w http.ResponseWriter, r *http.Request) bool {
					if strings.HasPrefix(r.URL.Path, "/themes") {
						w.Header().Set("Content-Type", "application/json")
						fmt.Fprint(w, `{"themes":[]}`)
						return true
					}
					return false
				}))
				return &writes
			},
		},
		{
			name: "app clone missing app-id",
			path: []string{"app", "clone"},
			want: []string{"--app-id"},
			setup: func(t *testing.T, cmd *cobra.Command) *atomic.Int32 {
				keyring.MockInit()
				if err := mjrToken.StoreDefaultOrg(tableOrgID, "Test Org"); err != nil {
					t.Fatal(err)
				}
				t.Setenv("MAJOR_TOKEN", "test-injected-token")
				_ = cmd.Flags().Set("app-id", "")
				var writes atomic.Int32
				restoreAPIClient(t, mutationCountingServer(t, &writes, func(w http.ResponseWriter, r *http.Request) bool {
					if r.URL.Path == "/organizations/applications" {
						w.Header().Set("Content-Type", "application/json")
						fmt.Fprint(w, `{"applications":[{"id":"a1","name":"One","githubRepositoryName":"one","cloneUrlSsh":"git@github.com:o/one.git","cloneUrlHttps":"https://github.com/o/one.git"},{"id":"a2","name":"Two","githubRepositoryName":"two","cloneUrlSsh":"git@github.com:o/two.git","cloneUrlHttps":"https://github.com/o/two.git"}]}`)
						return true
					}
					return false
				}))
				return &writes
			},
		},
		{
			name: "org select missing id",
			path: []string{"org", "select"},
			want: []string{"--id"},
			setup: func(t *testing.T, cmd *cobra.Command) *atomic.Int32 {
				keyring.MockInit()
				t.Setenv("MAJOR_TOKEN", "test-injected-token")
				_ = cmd.Flags().Set("id", "")
				var writes atomic.Int32
				restoreAPIClient(t, mutationCountingServer(t, &writes, func(w http.ResponseWriter, r *http.Request) bool {
					if r.URL.Path == "/organizations" && r.Method == http.MethodGet {
						w.Header().Set("Content-Type", "application/json")
						fmt.Fprint(w, `{"organizations":[{"id":"o1","name":"One"},{"id":"o2","name":"Two"}]}`)
						return true
					}
					return false
				}))
				return &writes
			},
		},
		{
			name: "user gitconfig missing username",
			path: []string{"user", "gitconfig"},
			want: []string{"--username"},
			setup: func(t *testing.T, cmd *cobra.Command) *atomic.Int32 {
				keyring.MockInit()
				_ = cmd.Flags().Set("username", "")
				var writes atomic.Int32
				return &writes
			},
		},
		{
			name: "user login missing token",
			path: []string{"user", "login"},
			want: []string{"MAJOR_TOKEN"},
			setup: func(t *testing.T, cmd *cobra.Command) *atomic.Int32 {
				keyring.MockInit()
				t.Setenv("MAJOR_TOKEN", "")
				var writes atomic.Int32
				restoreAPIClient(t, mutationCountingServer(t, &writes, nil))
				return &writes
			},
		},
		{
			name: "app deploy missing message",
			path: []string{"app", "deploy"},
			want: []string{"--message"},
			setup: func(t *testing.T, cmd *cobra.Command) *atomic.Int32 {
				prepareDirtyAppRepo(t)
				t.Setenv("MAJOR_TOKEN", "test-injected-token")
				_ = cmd.Flags().Set("message", "")
				_ = cmd.Flags().Set("slug", "")
				var writes atomic.Int32
				restoreAPIClient(t, mutationCountingServer(t, &writes, func(w http.ResponseWriter, r *http.Request) bool {
					if r.URL.Path == "/applications/"+tableAppID+"/info" {
						writeAppInfo(w, "prototype")
						return true
					}
					return false
				}))
				return &writes
			},
		},
		{
			name: "app deploy missing slug",
			path: []string{"app", "deploy"},
			want: []string{"--slug"},
			setup: func(t *testing.T, cmd *cobra.Command) *atomic.Int32 {
				prepareDirtyAppRepo(t)
				t.Setenv("MAJOR_TOKEN", "test-injected-token")
				_ = cmd.Flags().Set("message", "ship it")
				_ = cmd.Flags().Set("slug", "")
				var writes atomic.Int32
				restoreAPIClient(t, mutationCountingServer(t, &writes, func(w http.ResponseWriter, r *http.Request) bool {
					if r.URL.Path == "/applications/"+tableAppID+"/info" {
						w.Header().Set("Content-Type", "application/json")
						fmt.Fprintf(w, `{"applicationId":%q,"organizationId":%q,"urlSlug":null,"name":"Prototype","deployStatus":"not_deployed","appUrl":null}`, tableAppID, tableOrgID)
						return true
					}
					return false
				}))
				return &writes
			},
		},
		{
			name: "resource manage names add and remove",
			path: []string{"resource", "manage"},
			want: []string{"add --id", "remove --id"},
			setup: func(t *testing.T, cmd *cobra.Command) *atomic.Int32 {
				writeAppWorkspace(t)
				t.Setenv("MAJOR_TOKEN", "test-injected-token")
				var writes atomic.Int32
				restoreAPIClient(t, mutationCountingServer(t, &writes, func(w http.ResponseWriter, r *http.Request) bool {
					if r.URL.Path == "/applications/"+tableAppID+"/info" {
						writeAppInfo(w, "prototype")
						return true
					}
					if r.URL.Path == "/resources" {
						w.Header().Set("Content-Type", "application/json")
						fmt.Fprint(w, `{"resources":[{"id":"r1","name":"DB","type":"postgres","description":"db"}]}`)
						return true
					}
					return false
				}))
				return &writes
			},
		},
		{
			name: "resource env missing id",
			path: []string{"resource", "env"},
			want: []string{"--id", "env-list"},
			setup: func(t *testing.T, cmd *cobra.Command) *atomic.Int32 {
				writeAppWorkspace(t)
				t.Setenv("MAJOR_TOKEN", "test-injected-token")
				_ = cmd.Flags().Set("id", "")
				var writes atomic.Int32
				restoreAPIClient(t, mutationCountingServer(t, &writes, func(w http.ResponseWriter, r *http.Request) bool {
					switch {
					case r.URL.Path == "/applications/"+tableAppID+"/info":
						writeAppInfo(w, "prototype")
						return true
					case r.URL.Path == "/application/"+tableAppID+"/environment" && r.Method == http.MethodGet:
						w.Header().Set("Content-Type", "application/json")
						fmt.Fprint(w, `{"environmentId":"e1","environmentName":"dev"}`)
						return true
					case r.URL.Path == "/application/"+tableAppID+"/environments":
						w.Header().Set("Content-Type", "application/json")
						fmt.Fprint(w, `{"environments":[{"id":"e1","name":"dev"},{"id":"e2","name":"prod"}]}`)
						return true
					default:
						return false
					}
				}))
				return &writes
			},
		},
		{
			name: "vars unset missing yes",
			path: []string{"vars", "unset"},
			args: []string{"CLI_PROTOTYPE"},
			want: []string{"--yes"},
			setup: func(t *testing.T, cmd *cobra.Command) *atomic.Int32 {
				writeAppWorkspace(t)
				t.Setenv("MAJOR_TOKEN", "test-injected-token")
				_ = cmd.Flags().Set("yes", "false")
				_ = cmd.Flags().Set("env", "")
				var writes atomic.Int32
				restoreAPIClient(t, mutationCountingServer(t, &writes, func(w http.ResponseWriter, r *http.Request) bool {
					switch {
					case r.URL.Path == "/applications/"+tableAppID+"/info":
						writeAppInfo(w, "prototype")
						return true
					case r.URL.Path == "/application/"+tableAppID+"/environment" && r.Method == http.MethodGet:
						w.Header().Set("Content-Type", "application/json")
						fmt.Fprint(w, `{"environmentId":"e1","environmentName":"dev"}`)
						return true
					default:
						return false
					}
				}))
				return &writes
			},
		},
		{
			name: "project deploy missing yes",
			path: []string{"project", "deploy"},
			want: []string{"--yes"},
			setup: func(t *testing.T, cmd *cobra.Command) *atomic.Int32 {
				dir := t.TempDir()
				runTableGit(t, dir, "init")
				runTableGit(t, dir, "config", "user.email", "test@example.com")
				runTableGit(t, dir, "config", "user.name", "Test")
				runTableGit(t, dir, "remote", "add", "origin", "https://github.com/acme/demo.git")
				t.Chdir(dir)
				t.Setenv("MAJOR_TOKEN", "test-injected-token")
				_ = cmd.Flags().Set("yes", "false")
				var writes atomic.Int32
				restoreAPIClient(t, mutationCountingServer(t, &writes, func(w http.ResponseWriter, r *http.Request) bool {
					switch {
					case r.URL.Path == "/projects/from-repo":
						w.Header().Set("Content-Type", "application/json")
						fmt.Fprint(w, `{"projectId":"p-1","organizationId":"org-1"}`)
						return true
					case strings.HasPrefix(r.URL.Path, "/projects/p-1/versions"):
						w.Header().Set("Content-Type", "application/json")
						fmt.Fprint(w, `{"versions":[{"id":"v-1","commitHash":"aaaaaaaaaaaa","compileStatus":"compiled","createdAt":"2026-09-09T00:00:00Z"}]}`)
						return true
					case strings.Contains(r.URL.Path, "/deploy-plan"):
						w.Header().Set("Content-Type", "application/json")
						fmt.Fprint(w, `{"creates":[],"updates":[],"unchanged":[],"deletes":["old-agent"]}`)
						return true
					default:
						return false
					}
				}))
				return &writes
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd, _, err := rootCmd.Find(tc.path)
			if err != nil {
				t.Fatalf("Find %v: %v", tc.path, err)
			}
			writes := tc.setup(t, cmd)
			out := &bytes.Buffer{}
			cmd.SetOut(out)
			cmd.SetErr(out)
			cmd.SetIn(strings.NewReader(""))
			_ = rootCmd.PersistentFlags().Set("non-interactive", "true")
			t.Cleanup(func() {
				_ = rootCmd.PersistentFlags().Set("non-interactive", "false")
				cmd.SetOut(nil)
				cmd.SetErr(nil)
				cmd.SetIn(nil)
			})

			runErr := runWithDeadline(t, 8*time.Second, func() error {
				if cmd.RunE == nil {
					return fmt.Errorf("command %s has no RunE handler", cmd.Name())
				}
				return cmd.RunE(cmd, tc.args)
			})
			if runErr == nil {
				t.Fatalf("incomplete choice must fail, output=%q", out.String())
			}
			msg := runErr.Error() + out.String()
			for _, want := range tc.want {
				if !strings.Contains(msg, want) {
					t.Errorf("error must contain %q, got %v output=%q", want, runErr, out.String())
				}
			}
			if got := writes.Load(); got != 0 {
				t.Fatalf("mutation calls = %d, want 0", got)
			}
		})
	}
}

func mutationCountingServer(t *testing.T, writes *atomic.Int32, read func(http.ResponseWriter, *http.Request) bool) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if read != nil && read(w, r) {
			return
		}
		writes.Add(1)
		t.Errorf("unexpected mutation request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func restoreAPIClient(t *testing.T, baseURL string) {
	t.Helper()
	prev := singletons.GetAPIClient()
	singletons.SetAPIClient(api.NewClient(baseURL))
	t.Cleanup(func() { singletons.SetAPIClient(prev) })
}

func writeAppWorkspace(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	writeAppWorkspaceAt(t, dir)
	t.Chdir(dir)
}

func writeAppWorkspaceAt(t *testing.T, dir string) {
	t.Helper()
	if err := workspace.Write(dir, workspace.Config{
		OrganizationID: tableOrgID,
		Target:         workspace.Target{Kind: "app", ApplicationID: tableAppID},
	}); err != nil {
		t.Fatal(err)
	}
}

func prepareDirtyAppRepo(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	runTableGit(t, dir, "init")
	runTableGit(t, dir, "config", "user.email", "test@example.com")
	runTableGit(t, dir, "config", "user.name", "Test")
	runTableGit(t, dir, "commit", "--allow-empty", "-m", "base")
	writeAppWorkspaceAt(t, dir)
	if err := os.WriteFile(dir+"/dirty.txt", []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
}

func writeAppInfo(w http.ResponseWriter, slug string) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"applicationId":%q,"organizationId":%q,"urlSlug":%q,"name":"Prototype","deployStatus":"not_deployed","appUrl":null}`, tableAppID, tableOrgID, slug)
}

func runTableGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func runWithDeadline(t *testing.T, d time.Duration, fn func() error) error {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- fn() }()
	select {
	case err := <-done:
		return err
	case <-time.After(d):
		t.Fatalf("command did not complete within %s (possible prompt)", d)
		return nil
	}
}
