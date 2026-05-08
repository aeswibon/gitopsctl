package app

import (
	"net/http"
	"testing"
)

func TestApprove_RejectsMissingCommit(t *testing.T) {
	h, e, _, _ := newTestHandler()
	c, _ := newJSONContext(e, http.MethodPost, "/applications/a1/approve", `{}`)
	c.SetParamNames("name")
	c.SetParamValues("a1")

	err := h.Approve(c)
	if err == nil {
		t.Fatal("expected bad request error when commitHash is missing")
	}
}
