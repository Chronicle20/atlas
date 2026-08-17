#!/usr/bin/env bash
# tools/change-surfaces.sh — deterministic classification of a change set.
#
# Answers the mechanical half of "which review applies here": which services and
# libs a diff touches, which architectural surfaces it crosses (REST, Kafka, DB,
# deploy, packet), and therefore which backend audit families are *at minimum*
# in scope. Whether the code actually satisfies those rules is semantic and
# stays with the reviewer; this script never makes that call.
#
# Why it exists
# -------------
# Measured on task-227: a `backend-guidelines-reviewer` opened with a 13.6KB
# `git diff --stat` pair at its turn 1-2 (carried through all 83 of its turns),
# then spent ~12 more turns rediscovering whether a Dockerfile exists and
# whether topic env vars changed — questions that are pure path classification.
# The roster decision itself ("the relevant subset of these agents") was
# unstated model judgement, where a missed `frontend_review=true` costs a whole
# review pass.
#
# THE OUTPUT IS ADDITIVE, NEVER AUTHORITATIVE FOR EXCLUSION.
# ---------------------------------------------------------
# A reviewer may add a family this script did not list. A reviewer may NOT drop
# a family because it is absent here. Every uncertain input widens the answer:
# an unresolvable base, a changed Go file outside services/ and libs/, or a
# failed git command emits `classification=uncertain` together with the full
# family list. A classifier that silently narrows a review is worse than no
# classifier, so this one is built to over-run rather than under-run.
#
# Usage:
#   tools/change-surfaces.sh [--base <rev>] [--all]
#
#   --base <rev>   diff base (default: merge-base with origin/main, then main)
#   --all          classify as if everything changed — every surface true
#
# Output: `key=value`, one per line, stdout only. Exit 0 whenever it produced a
# block (including the uncertain one); exit 2 on usage error.
#
# Adding a key is safe; renaming one is not.

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

BASE=""
ALL=0

while [ $# -gt 0 ]; do
    case "$1" in
        --base) BASE="${2:-}"; [ -n "$BASE" ] || { echo "change-surfaces.sh: --base needs a rev" >&2; exit 2; }; shift 2 ;;
        --all) ALL=1; shift ;;
        -h|--help) sed -n '2,40p' "$0"; exit 0 ;;
        *) echo "change-surfaces.sh: unknown option $1" >&2; exit 2 ;;
    esac
done

# Every family in the checklist's family index. The order is the checklist's.
ALL_FAMILIES="DOM-STRUCTURE FILE SUB REST CONSTANTS TESTING CACHE MESSAGING MULTITENANCY MIGRATION DEPLOY RUNTIME CHANNEL-WIRE RESILIENCE EXT SCAFFOLD SEC"

UNCERTAIN=""
mark_uncertain() { UNCERTAIN="${UNCERTAIN:+$UNCERTAIN; }$1"; }

# ------------------------------------------------------------------ fail open

emit_open() {
    # Everything true, every family, and the reason. Used whenever the change
    # set cannot be established with confidence.
    echo "base=${1:-unknown}"
    echo "changed_files=unknown"
    echo "changed_services=unknown"
    echo "changed_libs=unknown"
    echo "changed_packages=unknown"
    echo "go_changed=true"
    echo "ts_changed=true"
    echo "rest_surface=true"
    echo "kafka_surface=true"
    echo "db_surface=true"
    echo "deploy_surface=true"
    echo "packet_surface=true"
    echo "tooling_surface=true"
    echo "new_service=true"
    echo "backend_audit_families=$(echo "$ALL_FAMILIES" | tr ' ' ',')"
    echo "frontend_review=true"
    echo "classification=uncertain"
    echo "uncertain_reason=$2"
    exit 0
}

# ----------------------------------------------------------------- change set

resolve_base() {
    if [ -n "$BASE" ]; then echo "$BASE"; return 0; fi
    git merge-base HEAD origin/main 2>/dev/null && return 0
    git merge-base HEAD main 2>/dev/null && return 0
    return 1
}

if [ "$ALL" -eq 1 ]; then
    emit_open "all" "--all requested"
fi

if ! base="$(resolve_base)"; then
    emit_open "unknown" "no merge base with origin/main or main"
fi
if ! base_sha="$(git rev-parse --short "$base" 2>/dev/null)"; then
    emit_open "unknown" "base rev '$base' does not resolve"
fi

# Same three sources as tools/verify.sh: the committed range, the worktree diff
# against HEAD (covers staged AND unstaged), and untracked files.
if ! CHANGED="$( { git diff --name-only "$base"...HEAD
                   git diff --name-only HEAD
                   git ls-files --others --exclude-standard; } 2>/dev/null )"; then
    emit_open "$base_sha" "git diff failed against $base_sha"
fi
CHANGED="$(printf '%s\n' "$CHANGED" | sort -u | sed '/^$/d')"

# Added-line content of the same range. Content triggers (a new const block, an
# AndEmit call, a token/secret reference) are not path-derivable, and reading
# them from the diff is what keeps those families from being blanket-included on
# every Go change. A failure here is not fatal — it just widens the answer.
ADDED=""
if ! ADDED="$( { git diff -U0 "$base"...HEAD; git diff -U0 HEAD; } 2>/dev/null | grep '^+' || true )"; then
    ADDED=""
    mark_uncertain "could not read diff content"
fi

# NOT `grep -q`. Under `set -o pipefail` grep -q exits on the first match, the
# still-writing printf takes SIGPIPE (141), and pipefail reports that as the
# pipeline's status — turning a match into "no match" on any input larger than
# the 64KB pipe buffer. $ADDED is the whole added-line diff, routinely ~1MB, so
# this is not a theoretical race here. Let grep drain its input.
has() { printf '%s\n' "$CHANGED" | grep -E "$1" >/dev/null; }
added_has() { [ -n "$ADDED" ] && printf '%s\n' "$ADDED" | grep -E "$1" >/dev/null; }

changed_go="$(printf '%s\n' "$CHANGED" | grep -E '\.go$' || true)"
changed_ts="$(printf '%s\n' "$CHANGED" | grep -E '\.tsx?$' || true)"

# Known Go layouts: services/ and libs/ carry the DDD packages the audit
# families are written against; tools/ carries analyzers and CLIs that carry
# none of them. A changed Go file in a FOURTH place is a layout this classifier
# was not written against — widen rather than guess.
tooling_surface=false
if [ -n "$changed_go" ]; then
    printf '%s\n' "$changed_go" | grep -E '^tools/' >/dev/null && tooling_surface=true
    unknown_go="$(printf '%s\n' "$changed_go" | grep -vE '^(services|libs|tools)/' | head -1 || true)"
    if [ -n "$unknown_go" ]; then
        mark_uncertain "Go file changed outside services/, libs/ and tools/: $unknown_go"
    fi
fi

# The service/lib Go set is what the audit families apply to. A tools/-only Go
# change triggers no service family — but it does change the guards themselves,
# which is why tooling_surface is reported rather than swallowed.
domain_go="$(printf '%s\n' "$changed_go" | grep -E '^(services|libs)/' || true)"

changed_services="$(printf '%s\n' "$CHANGED" | sed -n 's|^services/\([^/]*\)/.*|\1|p' | sort -u)"
changed_libs="$(printf '%s\n' "$CHANGED" | sed -n 's|^libs/\([^/]*\)/.*|\1|p' | sort -u)"
changed_pkgs="$(printf '%s\n' "$domain_go" | sed 's|/[^/]*$||' | sort -u | sed '/^$/d')"

# Sibling files of every changed Go package. Most family triggers are stated as
# "changed package has <file>.go", which is a property of the package on disk,
# not of the diff.
#
# `pkg_has` is a union across every changed package, which is right for a
# minimum-family answer: if ANY package has rest.go, the REST family is in
# scope. SUB is the one trigger that is not a union — "has resource.go but no
# model.go" is a property of a single package, and a union would answer false
# whenever some other changed package happens to have a model.go. It is
# therefore evaluated per package, here.
pkg_files=""
has_sub_pkg=false
while IFS= read -r p; do
    [ -n "$p" ] && [ -d "$p" ] || continue
    pkg_files="$pkg_files$(ls "$p" 2>/dev/null | tr '\n' ' ')"$'\n'
    if [ -f "$p/resource.go" ] && [ ! -f "$p/model.go" ]; then
        has_sub_pkg=true
    fi
done <<EOF
$changed_pkgs
EOF
pkg_has() { printf '%s\n' "$pkg_files" | grep -E "(^| )$1( |$)" >/dev/null; }

# ------------------------------------------------------------------- surfaces

go_changed=false;     [ -n "$changed_go" ] && go_changed=true
ts_changed=false;     [ -n "$changed_ts" ] && ts_changed=true

rest_surface=false
{ pkg_has 'resource\.go' || pkg_has 'rest\.go' || pkg_has 'requests\.go'; } && rest_surface=true

kafka_surface=false
{ has 'kafka/(message|producer|consumer)/' || pkg_has 'producer\.go' || pkg_has 'kafka\.go' \
  || added_has 'AndEmit|message\.Emit|producer\.ProviderImpl'; } && kafka_surface=true

db_surface=false
{ pkg_has 'entity\.go' || pkg_has 'administrator\.go' || added_has 'Migration|gorm\.|database\.Query'; } && db_surface=true

deploy_surface=false
{ has '^(deploy/|Dockerfile|docker-bake\.hcl|\.github/config/services\.json)' \
  || added_has '(COMMAND|EVENT)_TOPIC_'; } && deploy_surface=true

packet_surface=false
{ has '^(libs/atlas-packet/|services/atlas-channel/)'; } && packet_surface=true

# A new service is an ADDED go.mod under services/ — the one signal that is not
# ambiguous. A rename shows as add+delete and is correctly treated as new.
new_service=false
if added_mods="$(git diff --name-only --diff-filter=A "$base"...HEAD 2>/dev/null)"; then
    printf '%s\n' "$added_mods" | grep -E '^services/[^/]+/.*go\.mod$' >/dev/null && new_service=true
else
    mark_uncertain "could not list added files"
    new_service=true
fi

frontend_review=false
{ [ "$ts_changed" = true ] || has '^services/atlas-ui/'; } && frontend_review=true

# --------------------------------------------------------- audit families
#
# Triggers copied from the family index in
# .claude/skills/backend-dev-guidelines/resources/audit-checklist.md. When that
# table changes, this list is the thing that goes stale — it is a MINIMUM set,
# which is why staleness widens the review rather than narrowing it.

families=""
add() { case " $families " in *" $1 "*) ;; *) families="$families $1" ;; esac; }

if [ -n "$domain_go" ]; then
    add FILE                                             # any changed Go package, no exemptions
    printf '%s\n' "$domain_go" | grep -vE '_test\.go$' >/dev/null && add RUNTIME   # any non-test Go file

    { pkg_has 'model\.go' || pkg_has 'entity\.go' || pkg_has 'rest\.go' || pkg_has 'provider\.go'; } && add DOM-STRUCTURE
    [ "$has_sub_pkg" = true ] && add SUB
    { pkg_has 'resource\.go' || pkg_has 'rest\.go' || pkg_has 'processor\.go'; } && add REST
    { pkg_has 'cache\.go' || added_has 'cache|Cache'; } && add CACHE
    [ "$kafka_surface" = true ] && add MESSAGING
    { pkg_has 'rest\.go' || added_has 'tenant\.MustFromContext|WithContext\(ctx\)|tenantId|TenantId'; } && add MULTITENANCY
    { added_has '^\+[[:space:]]*(type|const)[[:space:]]' ; } && add CONSTANTS
    { printf '%s\n' "$domain_go" | grep -E '_test\.go$' >/dev/null \
      || added_has 'interface \{|Processor|Provider|Administrator'; } && add TESTING
    { [ -n "$changed_libs" ] && [ -n "$changed_services" ]; } && add MIGRATION
    { pkg_has 'requests\.go' || added_has 'requests\.RootUrl|requests\.GetRequest|requests\.PostRequest'; } && add EXT
    { added_has 'model\.Decorator|degrade\.Observe|StatusInternalServerError'; } && add RESILIENCE
    { added_has 'token|Token|secret|Secret|password|Password|credential|Credential|Redirect'; } && add SEC
fi

[ "$packet_surface" = true ] && add CHANNEL-WIRE
{ [ "$deploy_surface" = true ] || [ -n "$changed_libs" ]; } && add DEPLOY
{ [ "$new_service" = true ] || has '^deploy/shared/routes\.conf$' \
  || added_has 'RegisterWriter|RegisterHandler'; } && add SCAFFOLD

# An uncertain input widens to the full list — see the header contract.
if [ -n "$UNCERTAIN" ]; then
    families=" $ALL_FAMILIES"
fi

# ---------------------------------------------------------------------- print

csv() { local v; v="$(printf '%s\n' "$1" | sed '/^$/d' | paste -sd, -)"; printf '%s' "${v:-none}"; }

echo "base=$base_sha"
echo "changed_files=$(printf '%s\n' "$CHANGED" | sed '/^$/d' | wc -l | tr -d ' ')"
echo "changed_services=$(csv "$changed_services")"
echo "changed_libs=$(csv "$changed_libs")"
echo "changed_packages=$(printf '%s\n' "$changed_pkgs" | sed '/^$/d' | wc -l | tr -d ' ')"
echo "go_changed=$go_changed"
echo "ts_changed=$ts_changed"
echo "rest_surface=$rest_surface"
echo "kafka_surface=$kafka_surface"
echo "db_surface=$db_surface"
echo "deploy_surface=$deploy_surface"
echo "packet_surface=$packet_surface"
echo "tooling_surface=$tooling_surface"
echo "new_service=$new_service"
# Unquoted on purpose: $families is a space-separated list that must split.
# shellcheck disable=SC2086
echo "backend_audit_families=$(csv "$(printf '%s\n' $families)")"
echo "frontend_review=$frontend_review"
if [ -n "$UNCERTAIN" ]; then
    echo "classification=uncertain"
    echo "uncertain_reason=$UNCERTAIN"
else
    echo "classification=confident"
fi
