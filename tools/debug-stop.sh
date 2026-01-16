#!/usr/bin/env bash
set -euo pipefail

# -------------------------------
# Debug Stop Script
# -------------------------------
# Restores a Kubernetes service after local debugging by:
# 1. Restoring nginx routing to the original service
# 2. Scaling the deployment back to original replicas
# 3. Cleaning up debug state
#
# Usage: ./debug-stop.sh --service <name>
#        ./debug-stop.sh --all

# -------------------------------
# Configuration
# -------------------------------
NAMESPACE="atlas"
CONFIGMAP_NAME="atlas-ingress-configmap"
INGRESS_DEPLOYMENT="atlas-ingress"
ANNOTATION_PREFIX="debug.atlas.io"
SERVICE_DNS_SUFFIX=".atlas.svc.cluster.local"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# -------------------------------
# Helpers
# -------------------------------
log() {
  echo -e "${GREEN}==>${NC} $1"
}

warn() {
  echo -e "${YELLOW}WARNING:${NC} $1"
}

error() {
  echo -e "${RED}ERROR:${NC} $1" >&2
}

fail() {
  error "$1"
  exit 1
}

usage() {
  cat <<EOF
Usage: $(basename "$0") [OPTIONS]

Restore a Kubernetes service after local debugging by reverting
nginx routing and scaling the deployment back up.

OPTIONS:
  --service, -s <name>    Service name to restore (e.g., atlas-account)
  --all, -a               Restore all debugged services
  --status                Show currently debugged services
  --help, -h              Show this help message

EXAMPLES:
  # Stop debugging atlas-account and restore it
  $(basename "$0") --service atlas-account

  # Restore all services that are being debugged
  $(basename "$0") --all

  # Check what's currently being debugged
  $(basename "$0") --status

NOTES:
  - The service will be restored to its original replica count
  - Nginx will be reloaded automatically
EOF
}

# -------------------------------
# Preconditions
# -------------------------------
check_prerequisites() {
  if ! command -v kubectl >/dev/null 2>&1; then
    fail "kubectl is not installed or not in PATH"
  fi

  if ! kubectl auth can-i get configmaps -n "$NAMESPACE" >/dev/null 2>&1; then
    fail "No permission to access configmaps in namespace '$NAMESPACE'"
  fi

  if ! kubectl auth can-i update deployments -n "$NAMESPACE" >/dev/null 2>&1; then
    fail "No permission to update deployments in namespace '$NAMESPACE'"
  fi
}

# -------------------------------
# State Management
# -------------------------------
get_debug_state() {
  local service="$1"
  kubectl get configmap "$CONFIGMAP_NAME" -n "$NAMESPACE" \
    -o go-template='{{index .metadata.annotations "'"${ANNOTATION_PREFIX}/${service}"'"}}' 2>/dev/null || echo ""
}

remove_debug_state() {
  local service="$1"

  kubectl annotate configmap "$CONFIGMAP_NAME" -n "$NAMESPACE" \
    "${ANNOTATION_PREFIX}/${service}-" >/dev/null 2>&1 || true
}

get_all_debug_states() {
  kubectl get configmap "$CONFIGMAP_NAME" -n "$NAMESPACE" -o json 2>/dev/null | \
    grep -oP "\"${ANNOTATION_PREFIX}/[^\"]+\":\s*\"[^\"]+\"" | \
    sed "s/\"${ANNOTATION_PREFIX}\///" | \
    sed 's/":\s*"/: /' | \
    sed 's/"$//'
}

get_debugged_services() {
  kubectl get configmap "$CONFIGMAP_NAME" -n "$NAMESPACE" -o json 2>/dev/null | \
    grep -oP "\"${ANNOTATION_PREFIX}/[^\"]+\"" | \
    sed "s/\"${ANNOTATION_PREFIX}\///" | \
    sed 's/"$//'
}

show_status() {
  log "Currently debugged services:"
  echo

  local states
  states=$(get_all_debug_states)

  if [[ -z "$states" ]]; then
    echo "  No services currently in debug mode"
    return
  fi

  echo "$states" | while IFS=': ' read -r service state; do
    local replicas target
    replicas=$(echo "$state" | cut -d'|' -f1)
    target=$(echo "$state" | cut -d'|' -f2)
    echo -e "  ${BLUE}$service${NC}"
    echo "    Original replicas: $replicas"
    echo "    Debug target: $target"
    echo
  done
}

# -------------------------------
# Deployment Operations
# -------------------------------
scale_deployment() {
  local service="$1"
  local replicas="$2"

  kubectl scale deployment "$service" -n "$NAMESPACE" --replicas="$replicas" >/dev/null
}

# -------------------------------
# Nginx ConfigMap Operations
# -------------------------------
get_nginx_config() {
  kubectl get configmap "$CONFIGMAP_NAME" -n "$NAMESPACE" \
    -o jsonpath='{.data.nginx\.conf}'
}

patch_nginx_config() {
  local new_config="$1"

  # Create a temporary file for the patch
  local tmpfile
  tmpfile=$(mktemp)
  trap "rm -f '$tmpfile'" EXIT

  # Build the JSON patch
  cat > "$tmpfile" <<EOF
{
  "data": {
    "nginx.conf": $(echo "$new_config" | jq -Rs .)
  }
}
EOF

  kubectl patch configmap "$CONFIGMAP_NAME" -n "$NAMESPACE" \
    --type merge --patch-file "$tmpfile" >/dev/null
}

reload_nginx() {
  log "Restarting nginx ingress pod..."

  kubectl rollout restart deployment "$INGRESS_DEPLOYMENT" -n "$NAMESPACE" >/dev/null
  kubectl rollout status deployment "$INGRESS_DEPLOYMENT" -n "$NAMESPACE" --timeout=60s >/dev/null
}

# -------------------------------
# Main Restore Logic
# -------------------------------
stop_debug() {
  local service="$1"

  # Check if service is being debugged
  local debug_state
  debug_state=$(get_debug_state "$service")

  if [[ -z "$debug_state" ]]; then
    error "Service '$service' is not currently being debugged"
    echo
    echo "Currently debugged services:"
    local debugged
    debugged=$(get_debugged_services)
    if [[ -z "$debugged" ]]; then
      echo "  (none)"
    else
      echo "$debugged" | while read -r svc; do
        echo "  - $svc"
      done
    fi
    exit 1
  fi

  # Parse state
  local original_replicas debug_target
  original_replicas=$(echo "$debug_state" | cut -d'|' -f1)
  debug_target=$(echo "$debug_state" | cut -d'|' -f2)

  log "Stopping debug session for '$service'"
  echo "  Debug target was: $debug_target"
  echo "  Original replicas: $original_replicas"

  # Get and modify nginx config
  log "Restoring nginx configuration..."
  local nginx_config
  nginx_config=$(get_nginx_config)

  # Build the URLs
  local service_url="http://${service}${SERVICE_DNS_SUFFIX}:8080"
  local target_url="http://${debug_target}"

  # Count occurrences
  local occurrences
  occurrences=$(echo "$nginx_config" | grep -c "$target_url" || echo "0")

  if [[ "$occurrences" == "0" ]]; then
    warn "No proxy_pass directives found for debug target in nginx config"
    warn "Config may have been manually modified or already restored"
  else
    echo "  Found $occurrences route(s) to restore"

    # Perform replacement
    local modified_config
    modified_config=$(echo "$nginx_config" | sed "s|${target_url}|${service_url}|g")

    # Patch the ConfigMap
    patch_nginx_config "$modified_config"
  fi

  # Scale up deployment
  log "Scaling up deployment..."
  scale_deployment "$service" "$original_replicas"

  # Reload nginx
  reload_nginx

  # Remove state
  log "Cleaning up debug state..."
  remove_debug_state "$service"

  echo
  log "Debug session stopped successfully!"
  echo
  echo "  Service:  $service"
  echo "  Replicas: 0 -> $original_replicas"
}

stop_all_debug() {
  local debugged_services
  debugged_services=$(get_debugged_services)

  if [[ -z "$debugged_services" ]]; then
    log "No services currently in debug mode"
    exit 0
  fi

  log "Stopping all debug sessions..."
  echo

  # Process each service
  local failed=()
  local succeeded=()

  while read -r service; do
    echo "----------------------------------------"
    if stop_debug_single "$service"; then
      succeeded+=("$service")
    else
      failed+=("$service")
    fi
    echo
  done <<< "$debugged_services"

  # Summary
  echo "========================================"
  log "Summary:"
  echo "  Restored: ${#succeeded[@]}"
  if [[ ${#failed[@]} -gt 0 ]]; then
    echo "  Failed: ${#failed[@]}"
    for svc in "${failed[@]}"; do
      echo "    - $svc"
    done
    exit 1
  fi
}

# Helper for stop_all that doesn't exit on error
stop_debug_single() {
  local service="$1"

  # Check if service is being debugged
  local debug_state
  debug_state=$(get_debug_state "$service")

  if [[ -z "$debug_state" ]]; then
    warn "Service '$service' state not found, skipping"
    return 1
  fi

  # Parse state
  local original_replicas debug_target
  original_replicas=$(echo "$debug_state" | cut -d'|' -f1)
  debug_target=$(echo "$debug_state" | cut -d'|' -f2)

  log "Restoring '$service'"
  echo "  Debug target was: $debug_target"
  echo "  Original replicas: $original_replicas"

  # Get and modify nginx config
  local nginx_config
  nginx_config=$(get_nginx_config)

  # Build the URLs
  local service_url="http://${service}${SERVICE_DNS_SUFFIX}:8080"
  local target_url="http://${debug_target}"

  # Count occurrences
  local occurrences
  occurrences=$(echo "$nginx_config" | grep -c "$target_url" || echo "0")

  if [[ "$occurrences" != "0" ]]; then
    # Perform replacement
    local modified_config
    modified_config=$(echo "$nginx_config" | sed "s|${target_url}|${service_url}|g")
    patch_nginx_config "$modified_config"
  fi

  # Scale up deployment
  scale_deployment "$service" "$original_replicas"

  # Remove state
  remove_debug_state "$service"

  return 0
}

# -------------------------------
# Argument Parsing
# -------------------------------
SERVICE=""
ACTION=""

while [[ $# -gt 0 ]]; do
  case $1 in
    --service|-s)
      SERVICE="$2"
      shift 2
      ;;
    --all|-a)
      ACTION="all"
      shift
      ;;
    --status)
      ACTION="status"
      shift
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      fail "Unknown option: $1. Use --help for usage."
      ;;
  esac
done

# -------------------------------
# Main
# -------------------------------
check_prerequisites

case "$ACTION" in
  status)
    show_status
    exit 0
    ;;
  all)
    stop_all_debug
    # Reload nginx once after all changes
    reload_nginx
    exit 0
    ;;
  "")
    # Default action: stop debug for single service
    if [[ -z "$SERVICE" ]]; then
      fail "Missing required option: --service or --all. Use --help for usage."
    fi

    stop_debug "$SERVICE"
    ;;
esac
