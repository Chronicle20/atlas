#!/usr/bin/env bash
set -euo pipefail

# -------------------------------
# Debug Start Script
# -------------------------------
# Enables local debugging of a Kubernetes service by:
# 1. Scaling the deployment to 0
# 2. Redirecting nginx traffic to developer's local machine
# 3. Storing state for restoration
#
# Usage: ./debug-start.sh --service <name> --target <ip:port>
#        ./debug-start.sh --list
#        ./debug-start.sh --status

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

Enable local debugging of a Kubernetes service by redirecting traffic
from the cluster to your local machine.

OPTIONS:
  --service, -s <name>    Service name to debug (e.g., atlas-account)
  --target, -t <ip:port>  Target address for traffic (e.g., 192.168.1.100:8080)
  --list, -l              List all available services
  --status                Show currently debugged services
  --help, -h              Show this help message

EXAMPLES:
  # Start debugging atlas-account, redirect to local machine
  $(basename "$0") --service atlas-account --target 192.168.1.100:8080

  # List available services
  $(basename "$0") --list

  # Check what's currently being debugged
  $(basename "$0") --status

NOTES:
  - Your local machine must be reachable from the Kubernetes cluster
  - The service will be scaled to 0 replicas while debugging
  - Use debug-stop.sh to restore the service when done
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
# Service Discovery
# -------------------------------
get_all_services() {
  kubectl get deployments -n "$NAMESPACE" -o jsonpath='{.items[*].metadata.name}' | tr ' ' '\n' | grep -v "^${INGRESS_DEPLOYMENT}$" | sort
}

service_exists() {
  local service="$1"
  kubectl get deployment "$service" -n "$NAMESPACE" >/dev/null 2>&1
}

list_services() {
  log "Available services in namespace '$NAMESPACE':"
  echo
  get_all_services | while read -r svc; do
    echo "  - $svc"
  done
}

# -------------------------------
# State Management
# -------------------------------
get_debug_state() {
  local service="$1"
  kubectl get configmap "$CONFIGMAP_NAME" -n "$NAMESPACE" \
    -o jsonpath="{.metadata.annotations.${ANNOTATION_PREFIX}/${service}}" 2>/dev/null || echo ""
}

save_debug_state() {
  local service="$1"
  local replicas="$2"
  local target="$3"

  kubectl annotate configmap "$CONFIGMAP_NAME" -n "$NAMESPACE" \
    "${ANNOTATION_PREFIX}/${service}=${replicas}|${target}" \
    --overwrite >/dev/null
}

get_all_debug_states() {
  kubectl get configmap "$CONFIGMAP_NAME" -n "$NAMESPACE" -o json 2>/dev/null | \
    grep -oP "\"${ANNOTATION_PREFIX}/[^\"]+\":\s*\"[^\"]+\"" | \
    sed "s/\"${ANNOTATION_PREFIX}\///" | \
    sed 's/":\s*"/: /' | \
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
get_current_replicas() {
  local service="$1"
  kubectl get deployment "$service" -n "$NAMESPACE" \
    -o jsonpath='{.spec.replicas}'
}

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
  log "Reloading nginx configuration..."

  # Get the nginx pod name
  local pod
  pod=$(kubectl get pods -n "$NAMESPACE" -l app="$INGRESS_DEPLOYMENT" \
    -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)

  if [[ -z "$pod" ]]; then
    fail "Could not find nginx ingress pod"
  fi

  # Test config first
  if ! kubectl exec -n "$NAMESPACE" "$pod" -- nginx -t >/dev/null 2>&1; then
    fail "Nginx configuration test failed"
  fi

  # Reload nginx
  kubectl exec -n "$NAMESPACE" "$pod" -- nginx -s reload >/dev/null 2>&1
}

# -------------------------------
# Main Debug Logic
# -------------------------------
start_debug() {
  local service="$1"
  local target="$2"

  # Check if service exists
  if ! service_exists "$service"; then
    error "Service '$service' not found in namespace '$NAMESPACE'"
    echo
    echo "Available services:"
    get_all_services | head -10 | while read -r svc; do
      echo "  - $svc"
    done
    echo "  ... use --list to see all"
    exit 1
  fi

  # Check if already debugging
  local existing_state
  existing_state=$(get_debug_state "$service")

  if [[ -n "$existing_state" ]]; then
    local existing_target
    existing_target=$(echo "$existing_state" | cut -d'|' -f2)

    if [[ "$existing_target" == "$target" ]]; then
      log "Service '$service' is already being debugged with target '$target'"
      exit 0
    fi

    warn "Service '$service' is already being debugged (target: $existing_target)"
    log "Updating debug target to '$target'..."
  fi

  log "Starting debug session for '$service'"
  echo "  Target: $target"

  # Get current replicas (only if not already debugging)
  local original_replicas
  if [[ -z "$existing_state" ]]; then
    original_replicas=$(get_current_replicas "$service")
    echo "  Current replicas: $original_replicas"
  else
    original_replicas=$(echo "$existing_state" | cut -d'|' -f1)
    echo "  Original replicas (from state): $original_replicas"
  fi

  # Get and modify nginx config
  log "Updating nginx configuration..."
  local nginx_config
  nginx_config=$(get_nginx_config)

  # Find and replace all proxy_pass directives for this service
  local service_url="http://${service}${SERVICE_DNS_SUFFIX}:8080"
  local target_url="http://${target}"

  # Count occurrences
  local occurrences
  occurrences=$(echo "$nginx_config" | grep -c "$service_url" || echo "0")

  if [[ "$occurrences" == "0" ]]; then
    fail "No proxy_pass directives found for service '$service' in nginx config"
  fi

  echo "  Found $occurrences route(s) for '$service'"

  # Perform replacement
  local modified_config
  modified_config=$(echo "$nginx_config" | sed "s|${service_url}|${target_url}|g")

  # Patch the ConfigMap
  patch_nginx_config "$modified_config"

  # Save state
  log "Saving debug state..."
  save_debug_state "$service" "$original_replicas" "$target"

  # Scale down deployment (only if not already at 0)
  local current_replicas
  current_replicas=$(get_current_replicas "$service")

  if [[ "$current_replicas" != "0" ]]; then
    log "Scaling down deployment..."
    scale_deployment "$service" 0
  fi

  # Reload nginx
  reload_nginx

  echo
  log "Debug session started successfully!"
  echo
  echo "  Service:  $service"
  echo "  Target:   $target"
  echo "  Replicas: $original_replicas -> 0"
  echo
  echo "To stop debugging, run:"
  echo "  ./tools/debug-stop.sh --service $service"
}

# -------------------------------
# Argument Parsing
# -------------------------------
SERVICE=""
TARGET=""
ACTION=""

while [[ $# -gt 0 ]]; do
  case $1 in
    --service|-s)
      SERVICE="$2"
      shift 2
      ;;
    --target|-t)
      TARGET="$2"
      shift 2
      ;;
    --list|-l)
      ACTION="list"
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
  list)
    list_services
    exit 0
    ;;
  status)
    show_status
    exit 0
    ;;
  "")
    # Default action: start debug
    if [[ -z "$SERVICE" ]]; then
      fail "Missing required option: --service. Use --help for usage."
    fi
    if [[ -z "$TARGET" ]]; then
      fail "Missing required option: --target. Use --help for usage."
    fi

    # Validate target format
    if ! [[ "$TARGET" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+:[0-9]+$ ]]; then
      fail "Invalid target format. Expected: IP:PORT (e.g., 192.168.1.100:8080)"
    fi

    start_debug "$SERVICE" "$TARGET"
    ;;
esac
