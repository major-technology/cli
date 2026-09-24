package api

import (
	"errors"
	"strings"
	"testing"

	clierrors "github.com/major-technology/cli/errors"
)

func TestToCLIErrorUsesServerMessageForUnmappedCodes(t *testing.T) {
	for _, code := range []int{ErrorCodeTokenTypeNotAllowed, 9876} {
		err := ToCLIError(&ErrorResponse{Error: &AppErrorDetail{InternalCode: code, StatusCode: 403, ErrorString: "token_type_not_allowed", Message: "Use create_agent instead"}})
		if !strings.Contains(err.Error(), "Use create_agent instead") {
			t.Fatalf("code %d: %v", code, err)
		}
	}
}

func TestToCLIErrorKeepsMappedSentinels(t *testing.T) {
	err := ToCLIError(&ErrorResponse{Error: &AppErrorDetail{InternalCode: ErrorCodeAuthorizationPending, StatusCode: 400, ErrorString: "authorization_pending", Message: "Authorization pending"}})
	if !errors.Is(err, clierrors.ErrorAuthorizationPending) {
		t.Fatalf("expected authorization pending sentinel, got %v", err)
	}
}
