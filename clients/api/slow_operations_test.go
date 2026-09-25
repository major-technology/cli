package api

import "testing"

func TestListSlowOperationsUsesInvocationAggregatesRoute(t *testing.T) {
	for _, tc := range []struct {
		name       string
		includeAll bool
		path       string
	}{
		{"slow", false, "/applications/app-1/resource-invocation-aggregates?executionEnvironment=coding-session&windowDays=30"},
		{"all", true, "/applications/app-1/resource-invocation-aggregates?executionEnvironment=coding-session&minP95Ms=0&windowDays=30"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, client := newTestServer(t, "GET", tc.path, 200, ListSlowOperationsResponse{Operations: []SlowOperation{}})
			_, err := client.ListSlowOperations("app-1", ListSlowOperationsRequest{
				ExecutionEnvironment: "coding-session",
				WindowDays:           30,
				IncludeAll:           tc.includeAll,
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
