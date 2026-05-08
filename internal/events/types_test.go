package events

import "testing"

func TestEventTypes_HaveStablePrefix(t *testing.T) {
	types := []Type{
		TypeControllerStarted,
		TypeControllerStopping,
		TypeAppRegistered,
		TypeAppUnregistered,
		TypeAppSyncRequested,
		TypeAppSyncStarted,
		TypeAppGitPullFailed,
		TypeAppManifestPathMissing,
		TypeAppApplyFailed,
		TypeAppSyncSucceeded,
		TypeAppSyncNoChanges,
		TypeClusterRegistered,
		TypeClusterUnregistered,
		TypeClusterHealthCheckRequested,
		TypeClusterHealthCompleted,
	}

	for _, typ := range types {
		if len(typ) == 0 {
			t.Fatal("event type should not be empty")
		}
		if typ[0:3] != "io." {
			t.Fatalf("event type should start with io.: %s", typ)
		}
	}
	if SpecVersion != "1.0" {
		t.Fatalf("unexpected spec version: %s", SpecVersion)
	}
}
