#!/usr/bin/env bash
# Emits Kustomize literal entries for every COMMAND_TOPIC_* / EVENT_TOPIC_*
# key in deploy/k8s/base/env-configmap.yaml, suffixed with
# -PLACEHOLDER_ATLAS_ENV. Output is consumed by the per-PR overlay's
# kustomization.yaml configMapGenerator block.
#
# Usage: pipe directly into / inline in kustomization.yaml.
set -euo pipefail
ROOT="$(git rev-parse --show-toplevel)"
yq -r '.data | to_entries | .[] | select(.key | test("^(COMMAND|EVENT)_TOPIC_")) | "      - " + .key + "=" + .value + "-PLACEHOLDER_ATLAS_ENV"' \
    "$ROOT/deploy/k8s/base/env-configmap.yaml"
