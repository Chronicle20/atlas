#!/usr/bin/env bash
# Regenerate every artifact derived from the topic.Token constants in
# services/ and libs/: libs/atlas-kafka/gen/topics.yaml, the marked topic
# block in deploy/k8s/base/env-configmap.yaml, deploy/k8s/base/
# kafka-topics-configmap.yaml, the marked literals in all three overlay
# kustomizations, and deploy/compose/.env.example. task-276 FR-2/FR-3.
#
#   gen-topics.sh           rewrite the marker blocks in place
#   gen-topics.sh --check   exit 1 with a diff on drift; writes nothing
#
# The generator is its own module and is deliberately outside go.work, so
# it is invoked with GOWORK=off (same posture as tools/atlasguards).
set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "$REPO_ROOT/libs/atlas-kafka/gen"
GOWORK=off exec go run . "$@"
