package cluster

import "testing"

func TestRegisterHandlerMethod_Exists(t *testing.T) {
	_ = (*Handler).Register
}
