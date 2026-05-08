package cluster

import "testing"

func TestUnregisterHandlerMethod_Exists(t *testing.T) {
	_ = (*Handler).Unregister
}
