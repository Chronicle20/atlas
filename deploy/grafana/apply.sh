#!/usr/bin/env bash
set -euo pipefail
HERE="$(cd "$(dirname "$0")" && pwd)"
kubectl create configmap grafana-dashboards-atlas \
  -n observability \
  --from-file=dashboards-provider.yaml="$HERE/dashboards-provider.yaml" \
  --from-file=atlas-latency.json="$HERE/dashboards/atlas-latency.json" \
  --dry-run=client -o yaml | kubectl apply -f -
kubectl rollout restart deployment/grafana -n observability
