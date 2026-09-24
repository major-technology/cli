package api

import (
	"strings"
	"testing"
)

func TestToCLIErrorUsesServerMessage(t *testing.T) {
	for _, code := range []int{2007, 9876, ErrorCodeApplicationNotFound} {
		err := ToCLIError(&ErrorResponse{Error: &AppErrorDetail{InternalCode: code, StatusCode: 403, ErrorString: "token_type_not_allowed", Message: "Use create_agent instead"}})
		if !strings.Contains(err.Error(), "Use create_agent instead") {
			t.Fatalf("code %d: %v", code, err)
		}
	}
}
