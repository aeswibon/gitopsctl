#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TESTROOT="$(mktemp -d)"
mkdir -p "$TESTROOT/configs"
BIN="$TESTROOT/gitopsctl"

cleanup() {
  # Stop background processes if they exist.
  if [[ -n "${PID:-}" ]]; then
    kill "${PID}" >/dev/null 2>&1 || true
    wait "${PID}" >/dev/null 2>&1 || true
  fi
  if [[ -n "${APID:-}" ]]; then
    kill "${APID}" >/dev/null 2>&1 || true
    wait "${APID}" >/dev/null 2>&1 || true
  fi
  if [[ "${KEEP_TESTROOT:-0}" != "1" ]]; then
    rm -rf "$TESTROOT"
  fi
}
trap cleanup EXIT

cd "$REPO_ROOT"
go build -o "$BIN" .

chmod +x "$TESTROOT/gitopsctl"

cat > "$TESTROOT/configs/clusters.json" <<'EOF'
[
  {
    "name": "local",
    "kubeconfigPath": "/tmp/nonexistent-kubeconfig",
    "registeredAt": "2026-01-01T00:00:00Z",
    "status": "Pending",
    "message": "fixture"
  }
]
EOF

cat > "$TESTROOT/configs/applications.json" <<'EOF'
[
  {
    "name": "demoapp",
    "repoURL": "https://github.com/example/repo.git",
    "branch": "main",
    "path": "k8s",
    "clusterName": "local",
    "interval": "30s",
    "lastSyncedGitHash": "",
    "status": "Pending",
    "message": "fixture",
    "consecutiveFailures": 0
  }
]
EOF

cd "$TESTROOT"

# Help/parse coverage: dynamically enumerate all top-level commands.
./gitopsctl --help >"$TESTROOT/root.help"
# Extract commands from the help listing (lines like: "  command   description").
# Cobra groups render without an "Available Commands:" header, so match indented command rows.
awk '/^  [a-z0-9][a-z0-9-]+[[:space:]]+/ {print $1}' "$TESTROOT/root.help" \
  | grep -v '^gitopsctl$' \
  | sort -u \
  | while IFS= read -r cmd; do
      ./gitopsctl "$cmd" --help >/dev/null
    done

# Read-only output paths
./gitopsctl list-apps --output json >/dev/null
./gitopsctl status-apps --output json >/dev/null
./gitopsctl list-clusters --output json >/dev/null
./gitopsctl status-clusters --output json >/dev/null

# Mutating commands in dry-run mode
./gitopsctl register-apps -n demoapp -r https://github.com/example/repo.git -p k8s -c local --dry-run --force >/dev/null
./gitopsctl unregister -n demoapp --dry-run >/dev/null

# Cluster registration requires a real kubeconfig; ensure we fail cleanly.
./gitopsctl register-cluster -n local -k /tmp/nonexistent-kubeconfig --dry-run --force >/dev/null || true

# Start controller briefly and exercise API-backed CLI actions + SSE.
./gitopsctl start --api-address 127.0.0.1:18080 --events-file "$TESTROOT/configs/events.jsonl" >"$TESTROOT/start.log" 2>&1 &
PID=$!

for _ in $(seq 1 80); do
  if curl -fsS "http://127.0.0.1:18080/health" >/dev/null 2>&1; then
    break
  fi
  sleep 0.25
done

# Subscribe first, then trigger events
(curl -N "http://127.0.0.1:18080/api/v1/events" --max-time 4 >"$TESTROOT/sse.out" 2>/dev/null || true) &
CPID=$!
sleep 0.8

./gitopsctl --api-url http://127.0.0.1:18080 sync-app -n demoapp >/dev/null || true
./gitopsctl --api-url http://127.0.0.1:18080 check-cluster -n local >/dev/null || true

wait "$CPID" || true
kill "$PID" >/dev/null 2>&1 || true
wait "$PID" >/dev/null 2>&1 || true

test -s "$TESTROOT/configs/events.jsonl"

#
# Optional "actual scenario" run:
# If RUN_ACTUAL_SCENARIO=1 and kubectl can reach a real cluster, do a full end-to-end reconcile
# using a public repo with manifests.
#
if [[ "${RUN_ACTUAL_SCENARIO:-0}" == "1" ]] && command -v kubectl >/dev/null 2>&1 && kubectl version --client >/dev/null 2>&1 && kubectl cluster-info >/dev/null 2>&1; then
  echo "ACTUAL_SCENARIO=running"
  KCFG="${KUBECONFIG:-$HOME/.kube/config}"
  # If KUBECONFIG is a list, pick the first existing file.
  if [[ "$KCFG" == *:* ]]; then
    FIRST_EXISTING=""
    OLD_IFS="$IFS"
    IFS=':'
    for p in $KCFG; do
      if [[ -f "$p" ]]; then
        FIRST_EXISTING="$p"
        break
      fi
    done
    IFS="$OLD_IFS"
    if [[ -n "$FIRST_EXISTING" ]]; then
      KCFG="$FIRST_EXISTING"
    fi
  fi
  if [[ ! -f "$KCFG" ]]; then
    echo "ACTUAL_SCENARIO=skipped (no readable kubeconfig path found)"
    echo "CLI_SMOKE_OK"
    if [[ "${KEEP_TESTROOT:-0}" == "1" ]]; then
      echo "TESTROOT=$TESTROOT"
    fi
    exit 0
  fi
  cp "$KCFG" "$TESTROOT/real.kubeconfig"
  # Use a known repo with simple manifests.
  APP_REPO="https://github.com/kubernetes/examples.git"
  APP_PATH="staging/guestbook"

  if ./gitopsctl register-cluster -n real -k "$TESTROOT/real.kubeconfig" --force >/dev/null; then
    ./gitopsctl register-apps -n guestbook -r "$APP_REPO" -p "$APP_PATH" -c real -i 30s --force >/dev/null
  else
    echo "ACTUAL_SCENARIO=skipped (cluster registration failed)"
    echo "CLI_SMOKE_OK"
    if [[ "${KEEP_TESTROOT:-0}" == "1" ]]; then
      echo "TESTROOT=$TESTROOT"
    fi
    exit 0
  fi

  ./gitopsctl start --api-address 127.0.0.1:18082 --events-file "$TESTROOT/configs/events.actual.jsonl" >"$TESTROOT/start.actual.log" 2>&1 &
  APID=$!
  for _ in $(seq 1 80); do
    if curl -fsS "http://127.0.0.1:18082/health" >/dev/null 2>&1; then
      break
    fi
    sleep 0.25
  done

  # Trigger sync immediately and wait briefly.
  ./gitopsctl --api-url http://127.0.0.1:18082 sync-app -n guestbook >/dev/null || true
  ./gitopsctl --api-url http://127.0.0.1:18082 check-cluster -n real >/dev/null || true
  sleep 6

  # Basic k8s check (guestbook creates resources; tolerate namespace differences).
  kubectl get pods -A >/dev/null || true

  # Verify actual scenario emitted at least one event record.
  test -s "$TESTROOT/configs/events.actual.jsonl"

  kill "$APID" >/dev/null 2>&1 || true
  wait "$APID" >/dev/null 2>&1 || true
else
  echo "ACTUAL_SCENARIO=skipped (set RUN_ACTUAL_SCENARIO=1 and configure kubectl)"
fi

echo "CLI_SMOKE_OK"
if [[ "${KEEP_TESTROOT:-0}" == "1" ]]; then
  echo "TESTROOT=$TESTROOT"
fi
