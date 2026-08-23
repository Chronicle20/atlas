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

# bug-sparse-env-ui-not-deployed: a non-Go service (atlas-ui — node-service,
# no go.mod, never appears in go-services) must still enter the override
# set when it has its own base Deployment, or a UI-only PR silently serves
# main's UI.
check sparse services/atlas-ui/src/App.tsx
set -- services/atlas-ui/src/App.tsx
overrides=$(printf '%s\n' "$@" | ./tools/mode-select.sh | sed -n '2p')
for required in atlas-login atlas-channel atlas-ui; do
    echo "$overrides" | tr ' ' '\n' | grep -qx "$required" \
        || fail "override set [$overrides] is missing $required"
done

# The existing Go-service case must yield exactly its prior override set —
# docker-services already carries go-affected services back in (see
# tools/cideps/main.go), so unioning it in must not add anything spurious.
set -- services/atlas-monsters/atlas.com/monsters/monster/processor.go
overrides=$(printf '%s\n' "$@" | ./tools/mode-select.sh | sed -n '2p')
[ "$overrides" = "atlas-channel atlas-login atlas-monsters" ] \
    || fail "go-service override set [$overrides] gained a spurious entry"

# A changed service with no base Deployment (atlas-pr-bootstrap: a
# support-image with docker_image set but no deploy/k8s/base manifest) must
# not enter the override set — nothing deploys it, and it would still
# survive PLACEHOLDER_DELETE_BLOCK and land in the environment-record
# overrides map if it did.
check sparse services/atlas-pr-bootstrap/main.go
set -- services/atlas-pr-bootstrap/main.go
overrides=$(printf '%s\n' "$@" | ./tools/mode-select.sh | sed -n '2p')
[ "$overrides" = "atlas-channel atlas-login" ] \
    || fail "override set [$overrides] wrongly includes atlas-pr-bootstrap (no base Deployment)"

# FR-9.5: per-PR mode override labels, in both directions. A forced mode
# must win regardless of what the escalation table or cideps would
# otherwise compute. This asserts BOTH the mode line AND the override-set
# line — a forced-sparse case that checks only the mode is not a test:
# it passed while the override-set computation was silently skipped and
# every base Deployment got removed with nothing to replace it.
check_forced() { # <ATLAS_FORCE_MODE value> <expected-mode> <changed file>
    got=$(printf '%s\n' "$3" | ATLAS_FORCE_MODE="$1" ./tools/mode-select.sh | head -1)
    [ "$got" = "$2" ] || fail "forced=$1 file=$3: mode=$got, want=$2"
}

check_forced_overrides() { # <ATLAS_FORCE_MODE value> <changed file> <required override...>
    force="$1"; file="$2"; shift 2
    overrides=$(printf '%s\n' "$file" | ATLAS_FORCE_MODE="$force" ./tools/mode-select.sh | sed -n '2p')
    [ -n "$overrides" ] || fail "forced=$force file=$file: override set is empty"
    for required in "$@"; do
        echo "$overrides" | tr ' ' '\n' | grep -qx "$required" \
            || fail "forced=$force file=$file: override set [$overrides] is missing $required"
    done
}

# down-force: a file that would otherwise escalate to isolated (libs/atlas-kafka
# is on the escalation table) is forced down to sparse. Forcing the mode must
# not bypass the override-set computation — the label means "validate this
# against the shared control plane," not "deploy nothing."
check_forced sparse   sparse   libs/atlas-kafka/consumer/manager.go
check_forced_overrides sparse libs/atlas-kafka/consumer/manager.go atlas-login atlas-channel

# The mandatory floor (atlas-login, atlas-channel) must be present in the
# forced-sparse override set even when the change carries no service impact
# of its own.
check_forced_overrides sparse docs/tasks/task-232-sparse-ephemeral-environments/plan.md atlas-login atlas-channel

# down-force where the affected-service set is non-empty: the override set
# must include the file's own service in addition to the floor.
check_forced_overrides sparse services/atlas-monsters/atlas.com/monsters/monster/processor.go atlas-login atlas-channel atlas-monsters

# up-force: a file that would otherwise compute sparse is forced up to isolated.
check_forced isolated isolated services/atlas-monsters/atlas.com/monsters/monster/processor.go

# Both labels at once is an error, not a precedence rule — a PR asking for
# both is a mistake, and silently picking one would hide it.
if printf 'x\n' | ATLAS_FORCE_MODE="sparse isolated" ./tools/mode-select.sh >/dev/null 2>&1; then
    fail "conflicting force labels accepted"
fi

echo "PASS"
