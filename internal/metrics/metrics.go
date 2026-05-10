package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// AppSyncTotal counts the total number of sync attempts, labeled by application name and status (success/failure/manual_approval_required).
	AppSyncTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gitopsctl_app_sync_total",
			Help: "Total number of application synchronization attempts",
		},
		[]string{"app", "cluster", "status"},
	)

	// ClusterStatus indicates the health of a cluster (1 for healthy/reachable, 0 for unhealthy/unreachable).
	ClusterStatus = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gitopsctl_cluster_status",
			Help: "Current connectivity status of a registered cluster (1=Healthy, 0=Error)",
		},
		[]string{"cluster"},
	)

	// AppSyncDuration tracks how long a successful sync takes.
	AppSyncDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gitopsctl_app_sync_duration_seconds",
			Help:    "Time taken to synchronize an application in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"app", "cluster"},
	)

	// AppHealthStatus indicates the actual health of the application's resources (1=Healthy, 0.5=Progressing, 0=Degraded/Error).
	AppHealthStatus = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gitopsctl_app_health_status",
			Help: "Current health status of application resources (1=Healthy, 0.5=Progressing, 0=Degraded/Error)",
		},
		[]string{"app", "cluster"},
	)

	// K8sApplyTotal counts individual Kubernetes resource apply operations.
	K8sApplyTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gitopsctl_k8s_apply_total",
			Help: "Total number of Kubernetes resource apply operations",
		},
		[]string{"app", "cluster", "kind", "status"},
	)

	// GitPullTotal counts git clone/pull operations.
	GitPullTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gitopsctl_git_pull_total",
			Help: "Total number of git clone/pull operations",
		},
		[]string{"app", "status"},
	)
)
