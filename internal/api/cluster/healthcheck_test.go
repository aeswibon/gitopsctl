package cluster

import "testing"

func TestHealthCheckHandlerMethod_Exists(t *testing.T) {
	_ = (*Handler).HealthCheck
}
