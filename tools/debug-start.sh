#!/usr/bin/env bash
set -euo pipefail

# -------------------------------
# Debug Start Script
# -------------------------------
# Enables local debugging of a Kubernetes service inside an Argo-CD-managed
# Atlas environment (atlas-main or atlas-pr-<N>) by:
#   1. Writing an nginx override into the unmanaged ConfigMap
#      `atlas-ingress-debug-overrides` that redirects the service's /api
#      prefix to the developer's machine. The base atlas-ingress nginx
#      conf includes /etc/nginx/conf.d/debug-overrides/*.conf BEFORE the
#      generated routes, so override location blocks win first-regex-match.
#   2. Disabling Argo CD `syncPolicy.automated` on the target Application
#      (one-shot per env; restored only when the last service stops).
#   3. Scaling the backend Deployment to 0 so it stops consuming Kafka in
#      parallel with the developer's local instance.
#   4. Restarting atlas-ingress so nginx reloads.
#
# Usage:
#   ./debug-start.sh --namespace <ns> --service <svc> --target <ip:port>
#   ./debug-start.sh --namespace <ns> --list
#   ./debug-start.sh [--namespace <ns>] --status
#
# Crash safety:
#   If the script dies between disabling Argo and committing the debug
#   session, an EXIT trap re-enables Argo automated and removes the
#   override key. If the trap also fails, debug-stop.sh --recover can
#   reconcile orphan state. See `--help` for manual recovery commands.

# -------------------------------
# Configuration
# -------------------------------
ARGOCD_NAMESPACE="argocd"
INGRESS_DEPLOYMENT="atlas-ingress"
OVERRIDE_CM="atlas-ingress-debug-overrides"
ROUTES_CM_PREFIX="atlas-ingress-routes-"
ANNOTATION_PREFIX="debug.atlas.io"
ARGO_BACKUP_ANNOTATION="${ANNOTATION_PREFIX}/automated-backup"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# -------------------------------
# Helpers
# -------------------------------
log()   { echo -e "${GREEN}==>${NC} $1"; }
warn()  { echo -e "${YELLOW}WARNING:${NC} $1" >&2; }
error() { echo -e "${RED}ERROR:${NC} $1" >&2; }
fail()  { error "$1"; exit 1; }

usage() {
  cat <<EOF
Usage: $(basename "$0") [OPTIONS]

Redirect a service's HTTP traffic in an Argo-managed Atlas namespace to a
developer's local machine, and silence the in-cluster pod's Kafka
consumption by scaling it to 0.

OPTIONS:
  --namespace, -n <ns>    Target namespace (e.g. atlas-main, atlas-pr-1756)
  --service,   -s <svc>   Backend service to redirect (e.g. atlas-account)
  --target,    -t <ip:p>  Local target address (e.g. 192.168.1.100:8080)
  --list,      -l         List backend services in the namespace
  --status                Show active debug sessions (all namespaces, or --namespace)
  --help,      -h         Show this help

EXAMPLES:
  $(basename "$0") --namespace atlas-main --service atlas-account --target 192.168.1.100:8080
  $(basename "$0") --namespace atlas-pr-1756 --service atlas-channel --target 10.0.0.5:8080
  $(basename "$0") --status

MANUAL RECOVERY (if both the script and its trap fail mid-flow):
  # Re-enable Argo auto-sync for an env:
  kubectl -n ${ARGOCD_NAMESPACE} patch application <ns> --type merge \\
    -p '{"spec":{"syncPolicy":{"automated":{"selfHeal":true,"prune":<true|false>}}}}'
  # (prune: false for atlas-main, true for atlas-pr-*)

  # Remove a stale override key (causes nginx to stop redirecting):
  kubectl -n <ns> patch configmap ${OVERRIDE_CM} \\
    --type json -p '[{"op":"remove","path":"/data/<svc>.conf"}]'

NOTES:
  - The override ConfigMap and the base nginx hook are namespace-scoped,
    so concurrent debug sessions in different envs do not collide.
  - Argo automated is disabled at most once per env (counted by override
    keys). The original automated spec is snapshotted onto the Argo
    Application as annotation ${ARGO_BACKUP_ANNOTATION}.
EOF
}

# -------------------------------
# Preconditions
# -------------------------------
check_prerequisites() {
  command -v kubectl >/dev/null 2>&1 || fail "kubectl not on PATH"
  command -v jq      >/dev/null 2>&1 || fail "jq not on PATH"
}

namespace_exists() {
  local ns="$1"
  kubectl get namespace "$ns" >/dev/null 2>&1
}

service_exists() {
  local ns="$1" svc="$2"
  kubectl -n "$ns" get deployment "$svc" >/dev/null 2>&1
}

argo_app_exists() {
  local app="$1"
  kubectl -n "$ARGOCD_NAMESPACE" get application "$app" >/dev/null 2>&1
}

# -------------------------------
# Service Discovery
# -------------------------------
list_backend_services() {
  local ns="$1"
  kubectl -n "$ns" get deployments \
    -o jsonpath='{.items[*].metadata.name}' \
    | tr ' ' '\n' \
    | grep -v "^${INGRESS_DEPLOYMENT}$" \
    | sort
}

find_routes_cm() {
  local ns="$1"
  kubectl -n "$ns" get cm -o name 2>/dev/null \
    | grep -E "^configmap/${ROUTES_CM_PREFIX}" \
    | head -1 \
    | sed 's|^configmap/||'
}

# Extract all location blocks for a given service from the routes template.
# Produces an override-friendly version (no $u indirection, direct proxy_pass
# to the developer's target). Each block is copied verbatim from the source
# except that `set $u "<svc>...";` is dropped and `proxy_pass http://$u...`
# is rewritten to `proxy_pass http://<target>...`.
generate_overrides() {
  local routes_text="$1" svc="$2" target="$3"

  awk -v svc="$svc" -v target="$target" '
    BEGIN { in_block=0; matched=0; buffer=""; this_matches=0 }

    /^location[[:space:]]+~/ {
      in_block=1
      this_matches=0
      buffer=$0 "\n"
      next
    }

    in_block==1 {
      # Detect ownership: lines like  set $u "atlas-account.${POD_NAMESPACE}..."
      if ($0 ~ "set[[:space:]]+\\$u[[:space:]]+\"" svc "\\.") {
        this_matches=1
        # Drop the set line entirely; rewrite proxy_pass next.
        next
      }
      # Skip every other set $u line (different service in same block? — none today).
      if ($0 ~ /set[[:space:]]+\$u[[:space:]]+"/) {
        next
      }
      # Rewrite proxy_pass when this block is a match.
      if (this_matches==1 && $0 ~ /proxy_pass[[:space:]]+http:\/\/\$u/) {
        sub(/http:\/\/\$u/, "http://" target, $0)
      }
      buffer = buffer $0 "\n"
      if ($0 ~ /^\}/) {
        if (this_matches==1) { printf "%s", buffer; matched++ }
        in_block=0
        buffer=""
        this_matches=0
      }
      next
    }
  ' <<< "$routes_text"
}

# -------------------------------
# Override ConfigMap
# -------------------------------
ensure_override_cm() {
  local ns="$1"
  if ! kubectl -n "$ns" get cm "$OVERRIDE_CM" >/dev/null 2>&1; then
    kubectl -n "$ns" create configmap "$OVERRIDE_CM" >/dev/null
  fi
}

set_override_key() {
  local ns="$1" svc="$2" content="$3"
  local tmpfile
  tmpfile=$(mktemp); trap "rm -f '$tmpfile'" RETURN
  cat > "$tmpfile" <<EOF
{
  "data": {
    "$svc.conf": $(jq -Rs . <<< "$content")
  }
}
EOF
  kubectl -n "$ns" patch configmap "$OVERRIDE_CM" \
    --type merge --patch-file "$tmpfile" >/dev/null
}

remove_override_key() {
  local ns="$1" svc="$2"
  # Use --type=json with a "remove" op so absence is tolerated via a "test"
  # would still error; instead, fetch + edit + apply path for safety.
  if ! kubectl -n "$ns" get cm "$OVERRIDE_CM" -o jsonpath="{.data.${svc}\\.conf}" >/dev/null 2>&1; then
    return 0
  fi
  kubectl -n "$ns" patch configmap "$OVERRIDE_CM" \
    --type json -p "[{\"op\":\"remove\",\"path\":\"/data/${svc}.conf\"}]" \
    >/dev/null 2>&1 || true
}

annotate_cm_replicas() {
  local ns="$1" svc="$2" replicas="$3" target="$4"
  kubectl -n "$ns" annotate configmap "$OVERRIDE_CM" \
    "${ANNOTATION_PREFIX}/${svc}-replicas=${replicas}" \
    "${ANNOTATION_PREFIX}/${svc}-target=${target}" \
    --overwrite >/dev/null
}

override_key_count() {
  local ns="$1"
  kubectl -n "$ns" get cm "$OVERRIDE_CM" -o json 2>/dev/null \
    | jq -r '.data | length'
}

# -------------------------------
# Argo CD opt-out
# -------------------------------
argo_automated_spec() {
  local app="$1"
  kubectl -n "$ARGOCD_NAMESPACE" get application "$app" \
    -o jsonpath='{.spec.syncPolicy.automated}'
}

argo_disable_automated() {
  local app="$1"
  # Snapshot the current automated block onto the Application as an annotation
  # (only if not already snapshotted by a prior debug session in this env).
  local existing
  existing=$(kubectl -n "$ARGOCD_NAMESPACE" get application "$app" \
    -o jsonpath="{.metadata.annotations.${ARGO_BACKUP_ANNOTATION//\//\\/}}" 2>/dev/null || true)

  if [[ -z "$existing" ]]; then
    local snapshot
    snapshot=$(argo_automated_spec "$app")
    if [[ -z "$snapshot" ]]; then
      snapshot='{"selfHeal":true,"prune":false}'  # safe default if missing
    fi
    kubectl -n "$ARGOCD_NAMESPACE" annotate application "$app" \
      "${ARGO_BACKUP_ANNOTATION}=${snapshot}" --overwrite >/dev/null
  fi

  # Strategic merge with null removes the field. Use JSON merge.
  kubectl -n "$ARGOCD_NAMESPACE" patch application "$app" \
    --type merge -p '{"spec":{"syncPolicy":{"automated":null}}}' >/dev/null
}

argo_restore_automated() {
  local app="$1"
  local snapshot
  snapshot=$(kubectl -n "$ARGOCD_NAMESPACE" get application "$app" \
    -o jsonpath="{.metadata.annotations.${ARGO_BACKUP_ANNOTATION//\//\\/}}" 2>/dev/null || true)

  if [[ -z "$snapshot" ]]; then
    warn "No automated-backup annotation on Application/${app}; restoring to {selfHeal:true,prune:false} as fallback."
    snapshot='{"selfHeal":true,"prune":false}'
  fi

  kubectl -n "$ARGOCD_NAMESPACE" patch application "$app" \
    --type merge -p "{\"spec\":{\"syncPolicy\":{\"automated\":${snapshot}}}}" >/dev/null

  kubectl -n "$ARGOCD_NAMESPACE" annotate application "$app" \
    "${ARGO_BACKUP_ANNOTATION}-" >/dev/null 2>&1 || true
}

argo_is_disabled() {
  local app="$1"
  local current
  current=$(argo_automated_spec "$app")
  [[ -z "$current" ]]
}

# -------------------------------
# Deployment scaling
# -------------------------------
get_replicas() {
  local ns="$1" svc="$2"
  kubectl -n "$ns" get deployment "$svc" -o jsonpath='{.spec.replicas}'
}

scale_deployment() {
  local ns="$1" svc="$2" replicas="$3"
  kubectl -n "$ns" scale deployment "$svc" --replicas="$replicas" >/dev/null
}

# -------------------------------
# Ingress reload
# -------------------------------
reload_ingress() {
  local ns="$1"
  log "Restarting atlas-ingress in $ns ..."
  kubectl -n "$ns" rollout restart deployment "$INGRESS_DEPLOYMENT" >/dev/null
  kubectl -n "$ns" rollout status  deployment "$INGRESS_DEPLOYMENT" --timeout=60s >/dev/null
}

# -------------------------------
# Status / list actions
# -------------------------------
list_action() {
  local ns="$1"
  [[ -z "$ns" ]] && fail "--list requires --namespace"
  namespace_exists "$ns" || fail "Namespace $ns not found"
  log "Backend services in $ns:"
  echo
  list_backend_services "$ns" | sed 's/^/  - /'
}

status_action() {
  local ns_filter="$1"
  local namespaces
  if [[ -n "$ns_filter" ]]; then
    namespaces="$ns_filter"
  else
    namespaces=$(kubectl get ns -o jsonpath='{.items[*].metadata.name}' \
      | tr ' ' '\n' | grep -E '^atlas-(main|pr-)' | sort)
  fi

  local any=0
  while read -r ns; do
    [[ -z "$ns" ]] && continue
    local cm_json
    cm_json=$(kubectl -n "$ns" get cm "$OVERRIDE_CM" -o json 2>/dev/null || true)
    [[ -z "$cm_json" ]] && continue

    local keys
    keys=$(echo "$cm_json" | jq -r '.data // {} | keys[]?' 2>/dev/null || true)
    [[ -z "$keys" ]] && continue

    any=1
    echo
    echo -e "${BLUE}${ns}${NC}"

    local argo_state="unknown"
    if argo_app_exists "$ns"; then
      if argo_is_disabled "$ns"; then argo_state="DISABLED"; else argo_state="enabled"; fi
    fi
    echo "  Argo automated: $argo_state"

    while read -r key; do
      local svc="${key%.conf}"
      local target replicas
      target=$(echo "$cm_json"   | jq -r ".metadata.annotations[\"${ANNOTATION_PREFIX}/${svc}-target\"] // \"?\"")
      replicas=$(echo "$cm_json" | jq -r ".metadata.annotations[\"${ANNOTATION_PREFIX}/${svc}-replicas\"] // \"?\"")
      echo "  - ${svc}  →  ${target}  (original replicas: ${replicas})"
    done <<< "$keys"
  done <<< "$namespaces"

  [[ $any -eq 0 ]] && log "No active debug sessions."
}

# -------------------------------
# Start
# -------------------------------
COMMITTED=0
ROLLBACK_NS=""
ROLLBACK_SVC=""
ROLLBACK_DISABLED_ARGO=0

on_exit() {
  local rc=$?
  if [[ $COMMITTED -eq 1 || -z "$ROLLBACK_NS" ]]; then
    exit $rc
  fi
  warn "Aborted mid-setup; attempting best-effort rollback ..."
  if [[ -n "$ROLLBACK_SVC" ]]; then
    remove_override_key "$ROLLBACK_NS" "$ROLLBACK_SVC" || true
  fi
  if [[ $ROLLBACK_DISABLED_ARGO -eq 1 ]]; then
    argo_restore_automated "$ROLLBACK_NS" || true
  fi
  warn "Rollback complete. See --help for manual recovery if anything looks stuck."
  exit $rc
}
trap on_exit EXIT INT TERM

start_action() {
  local ns="$1" svc="$2" target="$3"

  namespace_exists "$ns"   || fail "Namespace $ns not found"
  service_exists   "$ns" "$svc" || fail "Deployment $svc not found in $ns"
  argo_app_exists  "$ns"   || warn "No Argo Application named $ns in $ARGOCD_NAMESPACE — skipping auto-sync opt-out."

  log "Starting debug session"
  echo "  Namespace: $ns"
  echo "  Service:   $svc"
  echo "  Target:    $target"

  # 1. Resolve hash-suffixed routes ConfigMap.
  local routes_cm
  routes_cm=$(find_routes_cm "$ns")
  [[ -z "$routes_cm" ]] && fail "No ${ROUTES_CM_PREFIX}* ConfigMap in $ns"
  log "Found routes ConfigMap: $routes_cm"

  local routes_text
  routes_text=$(kubectl -n "$ns" get cm "$routes_cm" \
    -o jsonpath='{.data.routes\.conf\.template}')

  # 2. Build the override location blocks for this service.
  local overrides
  overrides=$(generate_overrides "$routes_text" "$svc" "$target")
  if [[ -z "$overrides" ]]; then
    fail "Found no location blocks in $routes_cm whose 'set \$u' references '$svc.'"
  fi
  local block_count
  block_count=$(grep -c "^location" <<< "$overrides")
  echo "  Override blocks: $block_count"

  # 3. Patch the override ConfigMap (creates it if absent).
  log "Writing override into ${OVERRIDE_CM}/${svc}.conf ..."
  ensure_override_cm "$ns"
  set_override_key "$ns" "$svc" "$overrides"
  ROLLBACK_NS="$ns"
  ROLLBACK_SVC="$svc"

  # 4. Capture original replicas (unless already debugging this svc).
  local existing_replicas
  existing_replicas=$(kubectl -n "$ns" get cm "$OVERRIDE_CM" \
    -o jsonpath="{.metadata.annotations.${ANNOTATION_PREFIX}/${svc}-replicas}" 2>/dev/null || true)

  local original_replicas
  if [[ -n "$existing_replicas" ]]; then
    original_replicas="$existing_replicas"
    echo "  Updating target for $svc (original replicas already captured: $original_replicas)"
  else
    original_replicas=$(get_replicas "$ns" "$svc")
    [[ -z "$original_replicas" ]] && original_replicas=1
  fi
  annotate_cm_replicas "$ns" "$svc" "$original_replicas" "$target"

  # 5. Disable Argo automated (once per env).
  if argo_app_exists "$ns"; then
    if ! argo_is_disabled "$ns"; then
      log "Disabling Argo automated on Application/$ns ..."
      argo_disable_automated "$ns"
      ROLLBACK_DISABLED_ARGO=1
    else
      log "Argo automated already disabled on Application/$ns (other debug session active)."
    fi
  fi

  # 6. Scale backend to 0 (idempotent).
  local current
  current=$(get_replicas "$ns" "$svc")
  if [[ "$current" != "0" ]]; then
    log "Scaling Deployment/$svc to 0 ..."
    scale_deployment "$ns" "$svc" 0
  fi

  # 7. Reload nginx.
  reload_ingress "$ns"

  COMMITTED=1
  echo
  log "Debug session active."
  echo "  HTTP traffic for $svc in $ns now redirects to $target"
  echo "  Stop with: ./tools/debug-stop.sh --namespace $ns --service $svc"
}

# -------------------------------
# Argument Parsing
# -------------------------------
NAMESPACE=""
SERVICE=""
TARGET=""
ACTION=""

while [[ $# -gt 0 ]]; do
  case $1 in
    --namespace|-n) NAMESPACE="$2"; shift 2;;
    --service|-s)   SERVICE="$2";   shift 2;;
    --target|-t)    TARGET="$2";    shift 2;;
    --list|-l)      ACTION="list";  shift;;
    --status)       ACTION="status"; shift;;
    --help|-h)      usage; exit 0;;
    *) fail "Unknown option: $1 (use --help)";;
  esac
done

check_prerequisites

case "$ACTION" in
  list)   list_action   "$NAMESPACE";;
  status) status_action "$NAMESPACE";;
  "")
    [[ -z "$NAMESPACE" ]] && fail "Missing --namespace (use --help)"
    [[ -z "$SERVICE"   ]] && fail "Missing --service (use --help)"
    [[ -z "$TARGET"    ]] && fail "Missing --target (use --help)"
    [[ "$TARGET" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+:[0-9]+$ ]] \
      || fail "Invalid --target format. Expected IP:PORT (e.g. 192.168.1.100:8080)"
    start_action "$NAMESPACE" "$SERVICE" "$TARGET"
    ;;
esac
