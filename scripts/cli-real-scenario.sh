#!/usr/bin/env bash
set -euo pipefail

# Strict real-environment scenario:
# - fails on any setup/runtime error
# - intended for local/manual release checks (not default CI)

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

APP_NAME="${APP_NAME:-guestbook-real}"
CLUSTER_NAME="${CLUSTER_NAME:-real}"
API_ADDR="${API_ADDR:-127.0.0.1:18082}"
API_URL="http://$API_ADDR"
APP_REPO="${APP_REPO:-https://github.com/kubernetes/examples.git}"
APP_PATH="${APP_PATH:-staging/guestbook}"
APP_INTERVAL="${APP_INTERVAL:-30s}"

if ! command -v kubectl >/dev/null 2>&1; then
  echo "kubectl is required for cli-real-scenario"
  exit 1
fi

# Verify cluster connectivity up front.
kubectl version --client >/dev/null
kubectl cluster-info >/dev/null

KCFG="${KUBECONFIG:-$HOME/.kube/config}"
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
  echo "No readable kubeconfig file found (resolved path: $KCFG)"
  exit 1
fi

TESTROOT="$(mktemp -d)"
mkdir -p "$TESTROOT/configs"
BIN="$TESTROOT/gitopsctl"

cleanup() {
  if [[ -n "${PID:-}" ]]; then
    kill "${PID}" >/dev/null 2>&1 || true
    wait "${PID}" >/dev/null 2>&1 || true
  fi
  if [[ "${KEEP_TESTROOT:-0}" != "1" ]]; then
    rm -rf "$TESTROOT"
  fi
}
trap cleanup EXIT

cd "$REPO_ROOT"
go build -o "$BIN" .

chmod +x "$TESTROOT/gitopsctl"
cp "$KCFG" "$TESTROOT/real.kubeconfig"

cd "$TESTROOT"

./gitopsctl register-cluster -n "$CLUSTER_NAME" -k "$TESTROOT/real.kubeconfig" --force >/dev/null
CLUSTERS_JSON="$(./gitopsctl list-clusters --output json)"
if ! printf "%s" "$CLUSTERS_JSON" | grep -Eq "\"name\"[[:space:]]*:[[:space:]]*\"$CLUSTER_NAME\""; then
  echo "cluster registration verification failed: '$CLUSTER_NAME' not present after register-cluster"
  echo "$CLUSTERS_JSON"
  exit 1
fi
./gitopsctl register-apps -n "$APP_NAME" -r "$APP_REPO" -p "$APP_PATH" -c "$CLUSTER_NAME" -i "$APP_INTERVAL" --force >/dev/null

./gitopsctl start --api-address "$API_ADDR" --events-file "$TESTROOT/configs/events.actual.jsonl" >"$TESTROOT/start.actual.log" 2>&1 &
PID=$!
cleanup() {
  kill "$PID" >/dev/null 2>&1 || true
  wait "$PID" >/dev/null 2>&1 || true
}
trap cleanup EXIT

for _ in $(seq 1 80); do
  if curl -fsS "$API_URL/health" >/dev/null 2>&1; then
    break
  fi
  sleep 0.25
done
curl -fsS "$API_URL/health" >/dev/null

# Subscribe first, then trigger events.
(curl -N "$API_URL/api/v1/events" --max-time 6 >"$TESTROOT/sse.actual.out" 2>/dev/null || true) &
CPID=$!
sleep 1

./gitopsctl --api-url "$API_URL" sync-app -n "$APP_NAME" >/dev/null

wait "$CPID" || true
sleep 2

test -s "$TESTROOT/configs/events.actual.jsonl"
test -s "$TESTROOT/sse.actual.out"

echo "CLI_REAL_SCENARIO_OK"
if [[ "${KEEP_TESTROOT:-0}" == "1" ]]; then
  echo "TESTROOT=$TESTROOT"
fi
