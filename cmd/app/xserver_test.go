package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/major-technology/cli/clients/workspace"
)

const (
	xserverTargetID  = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
	xserverOutsideID = "cccccccc-cccc-4ccc-8ccc-cccccccccccc"
	xserverSecondID  = "dddddddd-dddd-4ddd-8ddd-dddddddddddd"
)

// xserverServer serves the current app's info and a callable-apps list seeded with initial; puts records
// the accepted PUT bodies. It serves no other app's info, as for a sandbox token, and it rejects a PUT
// naming an app from another org, as the server does.
func xserverServer(t *testing.T, initial []string) (puts *[]string) {
	t.Helper()
	puts = &[]string{}
	names := map[string]string{xserverTargetID: "orders", xserverSecondID: "billing"}
	ids := initial

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		info := func(id, org, name string) {
			fmt.Fprintf(w, `{"applicationId":%q,"organizationId":%q,"urlSlug":"x","name":%q,"deployStatus":"deployed","appUrl":"https://%s.example.com"}`, id, org, name, name)
		}

		switch {
		case r.URL.Path == "/applications/"+niAppID+"/info":
			info(niAppID, niOrgID, "self")
		case r.URL.Path == "/applications/"+niAppID+"/callable-apps" && r.Method == http.MethodGet:
			apps := []map[string]any{}
			for _, id := range ids {
				apps = append(apps, map[string]any{"id": id, "name": names[id], "appUrl": "https://" + names[id] + ".example.com"})
			}
			json.NewEncoder(w).Encode(map[string]any{"applicationIds": ids, "applications": apps})
		case r.URL.Path == "/applications/"+niAppID+"/callable-apps" && r.Method == http.MethodPut:
			body, _ := io.ReadAll(r.Body)
			if strings.Contains(string(body), xserverOutsideID) {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintf(w, `{"error":"Application %s not found in this organization"}`, xserverOutsideID)
				return
			}
			*puts = append(*puts, string(body))
			w.Write(body)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	dir := gitRepo(t)
	if err := workspace.Write(dir, workspace.Config{
		OrganizationID: niOrgID,
		Target:         workspace.Target{Kind: "app", ApplicationID: niAppID},
	}); err != nil {
		t.Fatal(err)
	}
	if err := workspace.IgnoreLocalConfig(dir); err != nil {
		t.Fatal(err)
	}
	restoreAPIClient(t, srv.URL)
	t.Chdir(dir)

	return puts
}

func TestCallsRegisteredOnAppCmd(t *testing.T) {
	for _, sub := range Cmd.Commands() {
		if sub.Name() == "xserver" {
			return
		}
	}

	t.Fatal("app calls not registered on the app command")
}

func TestCallsAddSameOrgPutsID(t *testing.T) {
	puts := xserverServer(t, nil)
	flagXserverAddID = xserverTargetID

	if err := xserverAddCmd.RunE(xserverAddCmd, nil); err != nil {
		t.Fatalf("add failed: %v", err)
	}

	if len(*puts) != 1 || !strings.Contains((*puts)[0], xserverTargetID) {
		t.Fatalf("PUT bodies = %v, want one containing %s", *puts, xserverTargetID)
	}
}

func TestCallsAddOtherOrgErrors(t *testing.T) {
	puts := xserverServer(t, nil)
	flagXserverAddID = xserverOutsideID

	if err := xserverAddCmd.RunE(xserverAddCmd, nil); err == nil {
		t.Fatal("expected the server's org error")
	}

	if len(*puts) != 0 {
		t.Fatalf("unexpected PUT: %v", *puts)
	}
}

func TestCallsAddSelfErrors(t *testing.T) {
	puts := xserverServer(t, nil)
	flagXserverAddID = niAppID

	if err := xserverAddCmd.RunE(xserverAddCmd, nil); err == nil || !strings.Contains(err.Error(), "itself") {
		t.Fatalf("err = %v, want self error", err)
	}

	if len(*puts) != 0 {
		t.Fatalf("unexpected PUT: %v", *puts)
	}
}

func TestCallsRemoveAbsentErrors(t *testing.T) {
	puts := xserverServer(t, nil)
	flagXserverRemoveID = xserverTargetID

	if err := xserverRemoveCmd.RunE(xserverRemoveCmd, nil); err == nil {
		t.Fatal("expected error removing an absent id")
	}

	if len(*puts) != 0 {
		t.Fatalf("unexpected PUT: %v", *puts)
	}
}

func TestCallsRemovePresentPutsWithoutID(t *testing.T) {
	puts := xserverServer(t, []string{xserverTargetID, xserverSecondID})
	flagXserverRemoveID = xserverTargetID

	if err := xserverRemoveCmd.RunE(xserverRemoveCmd, nil); err != nil {
		t.Fatalf("remove failed: %v", err)
	}

	if len(*puts) != 1 || strings.Contains((*puts)[0], xserverTargetID) || !strings.Contains((*puts)[0], xserverSecondID) {
		t.Fatalf("PUT bodies = %v, want the other id only", *puts)
	}
}

func TestCallsListPrintsIDsAndNames(t *testing.T) {
	xserverServer(t, []string{xserverTargetID})
	flagXserverListJSON = true
	t.Cleanup(func() { flagXserverListJSON = false })

	var out strings.Builder
	xserverListCmd.SetOut(&out)
	t.Cleanup(func() { xserverListCmd.SetOut(nil) })

	if err := xserverListCmd.RunE(xserverListCmd, nil); err != nil {
		t.Fatalf("list failed: %v", err)
	}

	if !strings.Contains(out.String(), xserverTargetID) || !strings.Contains(out.String(), "orders") || !strings.Contains(out.String(), "https://orders.example.com") {
		t.Fatalf("output = %q", out.String())
	}
}
