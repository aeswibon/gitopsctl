package cluster

import "testing"

func TestListHandlerMethod_Exists(t *testing.T) {
	_ = (*Handler).List
}
