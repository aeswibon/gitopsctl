package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestMetricsInitialization(t *testing.T) {
	// Increment a counter and check if it works
	AppSyncTotal.WithLabelValues("test-app", "test-cluster", "success").Inc()

	count := testutil.ToFloat64(AppSyncTotal.WithLabelValues("test-app", "test-cluster", "success"))
	if count != 1 {
		t.Errorf("Expected AppSyncTotal to be 1, got %v", count)
	}

	// Set a gauge
	ClusterStatus.WithLabelValues("test-cluster").Set(1)
	val := testutil.ToFloat64(ClusterStatus.WithLabelValues("test-cluster"))
	if val != 1 {
		t.Errorf("Expected ClusterStatus to be 1, got %v", val)
	}

	// Observe a duration
	AppSyncDuration.WithLabelValues("test-app", "test-cluster").Observe(0.5)
}
