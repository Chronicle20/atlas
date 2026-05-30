#!/usr/bin/env bash
set -euo pipefail

# -------------------------------
# Debug Stop Script
# -------------------------------
# Reverses a session started by debug-start.sh:
#   1. Removes the override key from ConfigMap atlas-ingress-debug-overrides
#      (deletes the CM if its data becomes empty).
#   2. Restores the backend Deployment to its original replica count
#      (read from the override CM's annotation).
#   3. If no more services in this env are being debugged, restores Argo
#      CD `syncPolicy.automated` on the target Application from the
#      annotation snapshot.
#   4. Restarts atlas-ingress so nginx drops the override include.
#
# Usage:
#   ./debug-stop.sh --namespace <ns> --service <svc>
#   ./debug-stop.sh --namespace <ns> --all
#   ./debug-stop.sh [--namespace <ns>] --status
#   ./debug-stop.sh --recover         (re-enable Argo on every atlas env
#                                      where the override CM is gone but
#                                      the automated-backup annotation
#                                      is still set — cleans up after a
#                                      crashed start.)

# -------------------------------
# Configuration
# -------------------------------
ARGOCD_NAMESPACE="argocd"
INGRESS_DEPLOYMENT="atlas-ingress"
OVERRIDE_CM="atlas-ingress-debug-overrides"
ANNOTATION_PREFIX="debug.atlas.io"
ARGO_BACKUP_ANNOTATION="${ANNOTATION_PREFIX}/automated-backup"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log()   { echo -e "${GREEN}==>${NC} $1"; }
warn()  { echo -e "${YELLOW}WARNING:${NC} $1" >&2; }
error() { echo -e "${RED}ERROR:${NC} $1" >&2; }
fail()  { error "$1"; exit 1; }

usage() {
  cat <<EOF
Usage: $(basename "$0") [OPTIONS]

Stop a debug session started by debug-start.sh and let Argo CD reconcile
the env back to its git-defined state.

OPTIONS:
  --namespace, -n <ns>    Target namespace (e.g. atlas-main, atlas-pr-1756)
  --service,   -s <svc>   Service to stop debugging
  --all,       -a         Stop every debug session in --namespace
  --status                Show active sessions (delegates to debug-start.sh)
  --recover               Find envs where the override CM is missing but
                          the Argo automated-backup annotation persists
                          (orphan from a crashed start) and restore Argo.
  --help,      -h         Show this help

EXAMPLES:
  $(basename "$0") --namespace atlas-main --service atlas-account
  $(basename "$0") --namespace atlas-pr-1756 --all
  $(basename "$0") --recover
EOF
}

check_prerequisites() {
  command -v kubectl >/dev/null 2>&1 || fail "kubectl not on PATH"
  command -v jq      >/dev/null 2>&1 || fail "jq not on PATH"
}

namespace_exists() {
  kubectl get namespace "$1" >/dev/null 2>&1
}

service_exists() {
  kubectl -n "$1" get deployment "$2" >/dev/null 2>&1
}

argo_app_exists() {
  kubectl -n "$ARGOCD_NAMESPACE" get application "$1" >/dev/null 2>&1
}

# -------------------------------
# State helpers
# -------------------------------
override_cm_exists() {
  kubectl -n "$1" get cm "$OVERRIDE_CM" >/dev/null 2>&1
}

get_cm_json() {
  kubectl -n "$1" get cm "$OVERRIDE_CM" -o json 2>/dev/null
}

debugged_services() {
  local ns="$1"
  local json
  json=$(get_cm_json "$ns")
  [[ -z "$json" ]] && return
  echo "$json" | jq -r '.data // {} | keys[]?' | sed 's/\.conf$//'
}

get_replicas_annotation() {
  local ns="$1" svc="$2"
  kubectl -n "$ns" get cm "$OVERRIDE_CM" \
    -o jsonpath="{.metadata.annotations.${ANNOTATION_PREFIX}/${svc}-replicas}" 2>/dev/null
}

remove_override_key() {
  local ns="$1" svc="$2"
  kubectl -n "$ns" patch configmap "$OVERRIDE_CM" \
    --type json -p "[{\"op\":\"remove\",\"path\":\"/data/${svc}.conf\"}]" \
    >/dev/null 2>&1 || true
  kubectl -n "$ns" annotate configmap "$OVERRIDE_CM" \
    "${ANNOTATION_PREFIX}/${svc}-replicas-" \
    "${ANNOTATION_PREFIX}/${svc}-target-" \
    >/dev/null 2>&1 || true
}

cm_data_empty() {
  local ns="$1"
  local count
  count=$(get_cm_json "$ns" | jq -r '.data // {} | length' 2>/dev/null || echo "0")
  [[ "$count" == "0" ]]
}

delete_override_cm() {
  kubectl -n "$1" delete configmap "$OVERRIDE_CM" --ignore-not-found >/dev/null
}

# -------------------------------
# Argo restore
# -------------------------------
argo_automated_spec() {
  kubectl -n "$ARGOCD_NAMESPACE" get application "$1" \
    -o jsonpath='{.spec.syncPolicy.automated}'
}

argo_backup_snapshot() {
  kubectl -n "$ARGOCD_NAMESPACE" get application "$1" \
    -o jsonpath="{.metadata.annotations.${ARGO_BACKUP_ANNOTATION//\//\\/}}" 2>/dev/null
}

argo_restore_automated() {
  local app="$1"
  local snapshot
  snapshot=$(argo_backup_snapshot "$app")
  if [[ -z "$snapshot" ]]; then
    warn "No backup annotation on Application/$app; skipping restore."
    return 0
  fi
  log "Restoring Argo automated on Application/$app ..."
  kubectl -n "$ARGOCD_NAMESPACE" patch application "$app" \
    --type merge -p "{\"spec\":{\"syncPolicy\":{\"automated\":${snapshot}}}}" >/dev/null
  kubectl -n "$ARGOCD_NAMESPACE" annotate application "$app" \
    "${ARGO_BACKUP_ANNOTATION}-" >/dev/null 2>&1 || true
}

# -------------------------------
# Scale + reload
# -------------------------------
scale_deployment() {
  kubectl -n "$1" scale deployment "$2" --replicas="$3" >/dev/null
}

reload_ingress() {
  log "Restarting atlas-ingress in $1 ..."
  kubectl -n "$1" rollout restart deployment "$INGRESS_DEPLOYMENT" >/dev/null
  kubectl -n "$1" rollout status  deployment "$INGRESS_DEPLOYMENT" --timeout=60s >/dev/null
}

# -------------------------------
# Stop a single service
# -------------------------------
stop_single() {
  local ns="$1" svc="$2"
  local replicas
  replicas=$(get_replicas_annotation "$ns" "$svc")
  if [[ -z "$replicas" ]]; then
    warn "No replicas annotation for $svc; assuming 1."
    replicas=1
  fi

  log "Removing override for $svc in $ns ..."
  remove_override_key "$ns" "$svc"

  if service_exists "$ns" "$svc"; then
    log "Scaling Deployment/$svc back to $replicas ..."
    scale_deployment "$ns" "$svc" "$replicas"
  fi
}

# -------------------------------
# Main stop flow
# -------------------------------
stop_action() {
  local ns="$1" svc="$2" all="$3"

  namespace_exists "$ns" || fail "Namespace $ns not found"
  override_cm_exists "$ns" || fail "No debug session in $ns (ConfigMap/$OVERRIDE_CM not found)"

  local targets=()
  if [[ "$all" -eq 1 ]]; then
    while read -r s; do [[ -n "$s" ]] && targets+=("$s"); done < <(debugged_services "$ns")
    [[ ${#targets[@]} -eq 0 ]] && fail "No active debug sessions in $ns"
  else
    [[ -z "$svc" ]] && fail "Missing --service (or use --all)"
    targets=("$svc")
  fi

  for s in "${targets[@]}"; do
    stop_single "$ns" "$s"
  done

  # If no more keys, delete CM and restore Argo.
  if cm_data_empty "$ns"; then
    log "No more debug sessions in $ns; cleaning up."
    delete_override_cm "$ns"
    if argo_app_exists "$ns"; then
      argo_restore_automated "$ns"
    fi
  else
    log "Other debug sessions remain in $ns; Argo stays disabled."
  fi

  reload_ingress "$ns"

  echo
  log "Done."
}

# -------------------------------
# Recover orphans
# -------------------------------
recover_action() {
  log "Scanning for orphan Argo backups ..."
  local apps
  apps=$(kubectl -n "$ARGOCD_NAMESPACE" get applications -o json \
    | jq -r ".items[] | select(.metadata.annotations[\"${ARGO_BACKUP_ANNOTATION}\"]) | .metadata.name")

  if [[ -z "$apps" ]]; then
    log "No orphan backups found."
    return
  fi

  local recovered=0
  while read -r app; do
    [[ -z "$app" ]] && continue
    # If the matching namespace still has an active override CM with keys,
    # this is NOT an orphan — leave it alone.
    if kubectl -n "$app" get cm "$OVERRIDE_CM" >/dev/null 2>&1; then
      local count
      count=$(kubectl -n "$app" get cm "$OVERRIDE_CM" -o json \
        | jq -r '.data // {} | length')
      if [[ "$count" != "0" ]]; then
        echo "  Skipping $app: ${count} active session(s)"
        continue
      fi
    fi
    echo "  Restoring Argo on $app ..."
    argo_restore_automated "$app" || true
    recovered=$((recovered + 1))
  done <<< "$apps"

  log "Recovered $recovered Application(s)."
}

# -------------------------------
# Argument Parsing
# -------------------------------
NAMESPACE=""
SERVICE=""
ALL=0
ACTION=""

while [[ $# -gt 0 ]]; do
  case $1 in
    --namespace|-n) NAMESPACE="$2"; shift 2;;
    --service|-s)   SERVICE="$2";   shift 2;;
    --all|-a)       ALL=1;          shift;;
    --status)       ACTION="status"; shift;;
    --recover)      ACTION="recover"; shift;;
    --help|-h)      usage; exit 0;;
    *) fail "Unknown option: $1 (use --help)";;
  esac
done

check_prerequisites

case "$ACTION" in
  status)
    # Delegate to debug-start.sh --status to avoid duplicating logic.
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    if [[ -n "$NAMESPACE" ]]; then
      exec "$SCRIPT_DIR/debug-start.sh" --status --namespace "$NAMESPACE"
    else
      exec "$SCRIPT_DIR/debug-start.sh" --status
    fi
    ;;
  recover)
    recover_action
    ;;
  "")
    [[ -z "$NAMESPACE" ]] && fail "Missing --namespace (use --help)"
    stop_action "$NAMESPACE" "$SERVICE" "$ALL"
    ;;
esac
