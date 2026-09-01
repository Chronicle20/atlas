#!/usr/bin/env bash
# buildx-bootstrap.sh — idempotently create/select the governed `atlas`
# buildx builder, pinned to deploy/buildkit/buildkitd.toml (docker-container
# driver, bounded parallelism, GC policy).
#
# Two consequences of the docker-container driver, both load-bearing for
# callers of this script:
#
#   1. Unlike the default `docker` driver, `docker-container` does NOT write
#      built images to the local image store by default — a build simply
#      solves and discards the result unless the caller passes `--load`
#      (see tools/build-services.sh) or `--push`.
#   2. Switching drivers means a brand-new BuildKit instance with an empty
#      cache: the first build after this runs is a cold cache, including the
#      `/go/pkg/mod` and `/root/.cache/go-build` cache mounts the Dockerfile
#      relies on. That is expected, not a regression.
#
# usage: tools/buildx-bootstrap.sh [--check] [--force]
#
#   --check   exit 0 if the builder exists; otherwise exit 1 with the command
#             that creates it. Makes no changes.
#   --force   remove and recreate the builder. Required to pick up a change to
#             deploy/buildkit/buildkitd.toml — buildx cannot update the config
#             of an existing builder in place.
#   -h        this message
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

usage() {
    cat <<'EOF'
usage: tools/buildx-bootstrap.sh [--check] [--force]

  --check   exit 0 if the builder exists; otherwise exit 1 with the command
            that creates it. Makes no changes.
  --force   remove and recreate the builder. Required to pick up a change to
            deploy/buildkit/buildkitd.toml — buildx cannot update the config
            of an existing builder in place.
  -h        this message
EOF
}

CHECK=0
FORCE=0
while [ $# -gt 0 ]; do
    case "$1" in
        --check) CHECK=1 ;;
        --force) FORCE=1 ;;
        -h|--help) usage; exit 0 ;;
        *) echo "buildx-bootstrap: unknown option: $1" >&2; exit 2 ;;
    esac
    shift
done

NAME="${ATLAS_BUILDER:-atlas}"
CONFIG="deploy/buildkit/buildkitd.toml"

exists() {
    docker buildx inspect "$NAME" >/dev/null 2>&1
}

if [ "$CHECK" -eq 1 ]; then
    if exists; then
        exit 0
    fi
    echo "buildx-bootstrap: builder '$NAME' does not exist — run tools/buildx-bootstrap.sh" >&2
    exit 1
fi

if [ "$FORCE" -eq 1 ]; then
    docker buildx rm "$NAME" >/dev/null 2>&1 || true
fi

if exists; then
    if [ "$FORCE" -eq 0 ]; then
        docker buildx use "$NAME"
        echo "buildx-bootstrap: builder '$NAME' already exists (use --force to recreate from $CONFIG)"
        exit 0
    fi
fi

docker buildx create --name "$NAME" --driver docker-container --config "$CONFIG" --bootstrap --use
