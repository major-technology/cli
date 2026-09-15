package app

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/major-technology/cli/clients/api"
)

type GetApplicationInfoResponseForTest = api.GetApplicationInfoResponse

func TestInfoJSONCarriesDeployFields(t *testing.T) {
	payload := `{"applicationId":"a","name":"App","deployStatus":"deploy_failed","deployError":"next build exited 1","isPublic":false,"webhooksEnabled":true,"deployedHash":"abc123"}`

	var resp GetApplicationInfoResponseForTest
	if err := json.NewDecoder(strings.NewReader(payload)).Decode(&resp); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if resp.DeployError == nil || *resp.DeployError != "next build exited 1" {
		t.Fatalf("deployError not decoded: %#v", resp.DeployError)
	}

	if !resp.WebhooksEnabled {
		t.Fatal("webhooksEnabled not decoded")
	}

	if resp.DeployedHash == nil || *resp.DeployedHash != "abc123" {
		t.Fatalf("deployedHash not decoded: %#v", resp.DeployedHash)
	}
}
