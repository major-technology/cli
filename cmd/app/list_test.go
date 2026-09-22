package app

import (
	"bytes"
	"github.com/major-technology/cli/clients/api"
	"github.com/major-technology/cli/singletons"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListUsesTokenRouteAndPreservesJSON(t *testing.T) {
	t.Setenv("MAJOR_TOKEN", "injected-test-token")
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RequestURI() != "/apps?editable=true" {
			t.Errorf("path %s", r.URL.RequestURI())
		}
		w.Write([]byte(`{"applications":[{"id":"a","name":"A","description":"D","canEdit":true}]}`))
	}))
	defer s.Close()
	old := singletons.GetAPIClient()
	singletons.SetAPIClient(api.NewClient(s.URL))
	defer singletons.SetAPIClient(old)
	var out bytes.Buffer
	listCmd.SetOut(&out)
	listCmd.Flags().Set("json", "true")
	listCmd.Flags().Set("editable", "true")
	if err := listCmd.RunE(listCmd, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"canEdit":true`) {
		t.Fatal(out.String())
	}
}

func TestListDoesNotNeedNodeOrPnpm(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if err := Cmd.PersistentPreRunE(listCmd, nil); err != nil {
		t.Fatal(err)
	}
}
