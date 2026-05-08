package common

import "testing"

func TestErrorResponse_Fields(t *testing.T) {
	errResp := ErrorResponse{
		Message: "failure",
		Details: "reason",
	}
	if errResp.Message != "failure" || errResp.Details != "reason" {
		t.Fatalf("unexpected error response: %+v", errResp)
	}
}
