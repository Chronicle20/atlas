#!/usr/bin/env sh
# tools/mode-select_test.sh
set -eu
fail() { echo "FAIL: $*" >&2; exit 1; }
check() { # <expected-mode> <changed files...>
    want="$1"; shift
    got=$(printf '%s\n' "$@" | ./tools/mode-select.sh | head -1)
    [ "$got" = "$want" ] || fail "files [$*]: mode=$got, want=$want"
}

check sparse   services/atlas-monsters/atlas.com/monsters/monster/processor.go
check isolated deploy/k8s/base/atlas-monsters.yaml
check isolated libs/atlas-kafka/consumer/manager.go
check isolated libs/atlas-rest/requests/url.go
check isolated services/atlas-configurations/atlas.com/configurations/main.go
check isolated services/atlas-tenants/atlas.com/tenants/main.go
check isolated services/atlas-monsters/atlas.com/monsters/monster/entity.go
check isolated some/unknown/path.txt
check sparse   docs/tasks/task-232-sparse-ephemeral-environments/plan.md

# Root-level build-graph files define the whole fleet's build/deploy graph
# and must escalate — a PR touching them must not silently deploy only the
# mandatory floor while the actual change goes unexercised.
check isolated go.work
check isolated go.work.sum
check isolated docker-bake.hcl
# A deliberately unrecognised root-level file also escalates (conservative
# default: unmatched paths never default to sparse).
check isolated some-unknown-root-file.txt
check sparse   README.md

# The override set ALWAYS includes the mandatory floor (FR-9.4, D6).
set -- services/atlas-monsters/atlas.com/monsters/monster/processor.go
overrides=$(printf '%s\n' "$@" | ./tools/mode-select.sh | sed -n '2p')
for required in atlas-login atlas-channel atlas-monsters; do
    echo "$overrides" | tr ' ' '\n' | grep -qx "$required" \
        || fail "override set [$overrides] is missing $required"
done

# FR-9.5: per-PR mode override labels, in both directions. A forced mode
# must win regardless of what the escalation table or cideps would
# otherwise compute.
check_forced() { # <ATLAS_FORCE_MODE value> <expected-mode> <changed file>
    got=$(printf '%s\n' "$3" | ATLAS_FORCE_MODE="$1" ./tools/mode-select.sh | head -1)
    [ "$got" = "$2" ] || fail "forced=$1 file=$3: mode=$got, want=$2"
}

# down-force: a file that would otherwise escalate to isolated (libs/atlas-kafka
# is on the escalation table) is forced down to sparse.
check_forced sparse   sparse   libs/atlas-kafka/consumer/manager.go
# up-force: a file that would otherwise compute sparse is forced up to isolated.
check_forced isolated isolated services/atlas-monsters/atlas.com/monsters/monster/processor.go

# Both labels at once is an error, not a precedence rule — a PR asking for
# both is a mistake, and silently picking one would hide it.
if printf 'x\n' | ATLAS_FORCE_MODE="sparse isolated" ./tools/mode-select.sh >/dev/null 2>&1; then
    fail "conflicting force labels accepted"
fi

echo "PASS"
