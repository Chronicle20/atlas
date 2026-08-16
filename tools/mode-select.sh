#!/usr/bin/env sh
# tools/mode-select.sh — sparse-vs-isolated PR-environment mode decision
# (task-232, FR-9.2/9.3).
#
# Reads the changed-file list on stdin (one path per line, repo-relative,
# same list detect-changes already computed — this script does not re-walk
# the diff). Prints two lines to stdout:
#   1. the mode: "sparse" or "isolated"
#   2. the space-separated override-service set (sparse only; empty for
#      isolated — the isolated overlay builds every service, it has no
#      override set to compute)
#
# Escalation (FR-9.3) is evaluated first, conservatively: a changed path
# that matches none of the repo's known top-level roots escalates rather
# than being silently ignored. Only when nothing escalates does this call
# tools/cideps for the affected-service set and union in the mandatory
# floor (atlas-login, atlas-channel — FR-9.4/D6).
#
# Must be run from the repo root (same assumption tools/cideps and every
# other tools/*.sh script makes).
set -eu

CIDEPS_CONFIG="${CIDEPS_CONFIG:-.github/config/services.json}"

TMPFILE=$(mktemp)
trap 'rm -f "$TMPFILE"' EXIT
cat >"$TMPFILE"

# is_known_root: true when $1's top-level path segment is one this repo
# recognizes (a real directory under the repo root), or $1 is a bare
# root-level file explicitly allowlisted as safe to ignore. Anything else —
# an unrecognized top-level directory, OR an unrecognized bare root-level
# file (go.work, docker-bake.hcl, an unlisted new root file, ...) — is the
# "affected-service determination unreliable" case: escalate. A changed path
# that matches no rule must escalate, never default to sparse.
is_known_root() {
    case "$1" in
        services/*|libs/*|docs/*|deploy/*|tools/*|dev/*|.github/*) return 0 ;;
        # Root-level files with no build-graph or deploy impact — safe to
        # leave unrouted; document any addition here with why.
        README.md|LICENSE|.gitignore) return 0 ;;
        *) return 1 ;;
    esac
}

ESCALATE=0
CHANGED_LIBS=""
CHANGED_SVCS=""

add_csv() {
    # add_csv <existing-csv-varname-via-stdout-echo> <value> — emits the
    # updated CSV on stdout; caller captures it back into the variable.
    csv="$1"; val="$2"
    case ",$csv," in
        *",$val,"*) printf '%s' "$csv" ;;
        *) if [ -z "$csv" ]; then printf '%s' "$val"; else printf '%s,%s' "$csv" "$val"; fi ;;
    esac
}

while IFS= read -r f; do
    [ -z "$f" ] && continue

    base=$(basename "$f")

    case "$f" in
        deploy/k8s/base/*) ESCALATE=1 ;;
    esac
    case "$f" in
        deploy/shared/routes.conf) ESCALATE=1 ;;
    esac
    case "$f" in
        tools/gen-routes.sh) ESCALATE=1 ;;
    esac
    case "$f" in
        libs/atlas-kafka/*) ESCALATE=1 ;;
    esac
    case "$f" in
        */kafka/message/*) ESCALATE=1 ;;
    esac
    case "$f" in
        services/atlas-configurations/*) ESCALATE=1 ;;
    esac
    case "$f" in
        services/atlas-tenants/*) ESCALATE=1 ;;
    esac
    case "$f" in
        libs/atlas-kafka/*|libs/atlas-rest/*|libs/atlas-tenant/*|libs/atlas-redis/*|libs/atlas-env/*|libs/atlas-service/*)
            ESCALATE=1 ;;
    esac
    case "$base" in
        entity.go) ESCALATE=1 ;;
        migration*.go) ESCALATE=1 ;;
    esac

    if ! is_known_root "$f"; then
        ESCALATE=1
    fi

    case "$f" in
        services/*/*)
            svc=$(printf '%s\n' "$f" | sed -n 's#^services/\([^/][^/]*\)/.*#\1#p')
            [ -n "$svc" ] && CHANGED_SVCS=$(add_csv "$CHANGED_SVCS" "$svc")
            ;;
    esac
    case "$f" in
        libs/*/*)
            lib=$(printf '%s\n' "$f" | sed -n 's#^libs/\([^/][^/]*\)/.*#\1#p')
            [ -n "$lib" ] && CHANGED_LIBS=$(add_csv "$CHANGED_LIBS" "$lib")
            ;;
    esac
done <"$TMPFILE"

if [ "$ESCALATE" -eq 1 ]; then
    printf 'isolated\n\n'
    exit 0
fi

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
REPO_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)

if ! OUT=$(cd "$REPO_ROOT" && go run ./tools/cideps \
        --changed-libs="$CHANGED_LIBS" \
        --changed-services="$CHANGED_SVCS" \
        --config="$CIDEPS_CONFIG" 2>/dev/null); then
    # cideps failed — the affected-service determination is unreliable.
    printf 'isolated\n\n'
    exit 0
fi

SVC_NAMES=$(printf '%s' "$OUT" | jq -r '."go-services"[].name' 2>/dev/null || true)

OVERRIDES=$(printf '%s\natlas-login\natlas-channel\n' "$SVC_NAMES" | grep -v '^$' | sort -u | tr '\n' ' ')
OVERRIDES=${OVERRIDES% }

printf 'sparse\n%s\n' "$OVERRIDES"
