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

# The override set ALWAYS includes the mandatory floor (FR-9.4, D6).
set -- services/atlas-monsters/atlas.com/monsters/monster/processor.go
overrides=$(printf '%s\n' "$@" | ./tools/mode-select.sh | tail -1)
for required in atlas-login atlas-channel atlas-monsters; do
    echo "$overrides" | tr ' ' '\n' | grep -qx "$required" \
        || fail "override set [$overrides] is missing $required"
done

echo "PASS"
