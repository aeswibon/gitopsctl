package cluster

import "testing"

func TestGetHandlerMethod_Exists(t *testing.T) {
	_ = (*Handler).Get
}
