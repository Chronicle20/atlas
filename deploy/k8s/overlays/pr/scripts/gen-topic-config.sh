#!/usr/bin/env bash
# Emits Kustomize literal entries for every COMMAND_TOPIC_* / EVENT_TOPIC_*
# key in deploy/k8s/base/env-configmap.yaml, suffixed with a placeholder
# token. Output is consumed by an overlay's kustomization.yaml
# configMapGenerator block.
#
# Usage: gen-topic-config.sh [SUFFIX_TOKEN]
#
#   PLACEHOLDER_ATLAS_ENV (default)      — overlays/pr, isolated mode: this
#     environment gets its own topics, suffixed with its own env token.
#   PLACEHOLDER_BASELINE_ENVIRONMENT     — overlays/pr-sparse, sparse mode:
#     this environment shares the BASELINE's topics, so it must name them
#     the way the baseline names them. Note this is not the same as "no
#     suffix": the baseline overlay suffixes every topic with its own
#     environment id (`-main` today), so an unsuffixed name addresses a
#     topic nobody publishes to. That was the atlas-login crash-loop of
#     2026-08-20 — see docs/tasks/task-232-sparse-ephemeral-environments/
#     bug-sparse-baseline-scoping.md.
#
# Pipe directly into / inline in the target kustomization.yaml.
set -euo pipefail
export SUFFIX="-${1:-PLACEHOLDER_ATLAS_ENV}"
ROOT="$(git rev-parse --show-toplevel)"
yq -r '.data | to_entries | .[] | select(.key | test("^(COMMAND|EVENT)_TOPIC_")) | "      - " + .key + "=" + .value + strenv(SUFFIX)' \
    "$ROOT/deploy/k8s/base/env-configmap.yaml"
