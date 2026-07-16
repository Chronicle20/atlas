#!/usr/bin/env bash
# check-version-coverage.sh — CI guard: the set of client versions that have a
# socket config template MUST equal the set declared in versions.json, keyed on
# (region, majorVersion). Fails (exit 1) in either direction:
#   - a template with no versions.json entry (the version would ship with no LB
#     ports — the blind spot that let PR #971 land v48/61/72/79 unported), or
#   - a versions.json entry with no template (LB ports for a phantom version).
# Pure shell + jq; modifies nothing. See docs/tasks/task-170-version-port-guard.
set -euo pipefail
export LC_ALL=C   # deterministic sort/comm collation

REPO_ROOT="$(git rev-parse --show-toplevel)"
VERSIONS="$REPO_ROOT/deploy/k8s/base/versions.json"
TEMPLATE_DIR="$REPO_ROOT/services/atlas-configurations/seed-data/templates"

[ -f "$VERSIONS" ]     || { echo "check-version-coverage: missing $VERSIONS" >&2; exit 1; }
[ -d "$TEMPLATE_DIR" ] || { echo "check-version-coverage: missing $TEMPLATE_DIR" >&2; exit 1; }

# (region major) pairs declared in versions.json, sorted & unique.
versions_set() {
    jq -r '.versions[] | "\(.region) \(.majorVersion)"' "$VERSIONS" | sort -u
}

# (region major) pairs parsed from template_<region>_<major>_<minor>.json names.
templates_set() {
    local f base region rest major
    for f in "$TEMPLATE_DIR"/template_*.json; do
        [ -e "$f" ] || continue          # nullglob-safe
        base="$(basename "$f" .json)"    # template_gms_83_1
        base="${base#template_}"         # gms_83_1
        region="${base%%_*}"             # gms
        rest="${base#*_}"                # 83_1
        major="${rest%%_*}"              # 83
        printf '%s %s\n' "$region" "$major"
    done | sort -u
}

vset="$(versions_set)"
tset="$(templates_set)"

missing_versions="$(comm -23 <(printf '%s\n' "$tset") <(printf '%s\n' "$vset"))"
missing_templates="$(comm -13 <(printf '%s\n' "$tset") <(printf '%s\n' "$vset"))"

rc=0
if [ -n "$missing_versions" ]; then
    rc=1
    while read -r region major; do
        [ -z "$region" ] && continue
        echo "check-version-coverage: $region $major has a socket config template but no deploy/k8s/base/versions.json entry — add it and run tools/gen-lb-ports.sh, or remove the template." >&2
    done <<< "$missing_versions"
fi
if [ -n "$missing_templates" ]; then
    rc=1
    while read -r region major; do
        [ -z "$region" ] && continue
        echo "check-version-coverage: $region $major has a deploy/k8s/base/versions.json entry (LB ports) but no socket config template in services/atlas-configurations/seed-data/templates." >&2
    done <<< "$missing_templates"
fi

[ "$rc" -eq 0 ] && echo "check-version-coverage: OK (template set == versions.json set)"
exit "$rc"
