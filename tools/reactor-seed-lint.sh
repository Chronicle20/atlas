#!/usr/bin/env bash
# tools/reactor-seed-lint.sh — validates the reactor-action seed corpus.
#
# Why this exists: reactor_script_schema.json describes the reactor script
# resource, but nothing ever validated a seed file against it. The corpus
# grew a meso-parameter regression (reactor-2001 seeded minMeso/maxMeso/
# mesoRange/item, none of which executor.go reads) that no gate could see.
#
# Runs tools/reactor-seed-lint over deploy/seed. That binary asserts, per
# file: schema conformance of data.attributes, envelope well-formedness
# (type/id/filename agreement), a non-empty description, and — across the
# eleven tenant directories — byte identity of every reactor's copies.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

SEED_ROOT="${1:-deploy/seed}"
SCHEMA="services/atlas-reactor-actions/docs/reactor_script_schema.json"

if [ ! -d "$SEED_ROOT" ]; then
    echo "reactor-seed-lint: ERROR — seed root not found: $SEED_ROOT" >&2
    exit 1
fi
if [ ! -f "$SCHEMA" ]; then
    echo "reactor-seed-lint: ERROR — schema not found: $SCHEMA" >&2
    exit 1
fi

go run ./tools/reactor-seed-lint "$SEED_ROOT" "$SCHEMA"
