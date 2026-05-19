#!/usr/bin/env bash
# Atlas PR-env orphan sweep. Codifies the May-19 recovery: enumerate (and
# optionally delete) per-env state for one or more PR numbers.
#
# Usage:
#   sweep-orphans.sh [--apply] PR_NUMBER [PR_NUMBER ...]
#
# Without --apply (default): lists everything that would be deleted.
# With --apply: deletes it. Idempotent — safe to re-run after a partial sweep.
#
# Required env (same names cleanup.sh uses; defaults match cluster reality):
#   DB_HOST, DB_PORT, DB_USER, DB_PASSWORD
#   ATLAS_DB_NAMES                    — space-separated base DB names
#   BOOTSTRAP_SERVERS                 — Kafka bootstrap
#   REDIS_URL                         — host:port (NOT a URL)
#   GHCR_TOKEN                        — GitHub PAT (Contents+Packages write)
#   ATLAS_SERVICES                    — comma-separated service names
#   PIHOLE_API_BASE_1 / PIHOLE_TOKEN_1 / PIHOLE_API_BASE_2 / PIHOLE_TOKEN_2
#
# DRY_RUN_NO_INFRA=1 short-circuits external-command phases (testing only).

set -euo pipefail

# shellcheck source=lib.sh
. "$(dirname "$0")/lib.sh"

APPLY=0
PR_NUMBERS=()

usage() {
    cat <<'EOF'
Usage: sweep-orphans.sh [--apply] PR_NUMBER [PR_NUMBER ...]

  --apply        Actually delete state. Without this flag, sweep is list-only.
  PR_NUMBER      One or more positive integers.

Without --apply, all phases print what they would do, one resource per line,
prefixed with the phase name (drop-dbs, drop-topics, drop-groups,
drop-redis, drop-images, drop-dns, drop-app-finalizers, drop-branch).
Suitable for piping through `tee` or `diff` for visual review before re-running
with --apply.
EOF
}

while [ $# -gt 0 ]; do
    case "$1" in
        --apply) APPLY=1 ; shift ;;
        --list)  APPLY=0 ; shift ;;     # explicit form, same as default
        -h|--help) usage ; exit 0 ;;
        --) shift ; break ;;
        -*) echo "unknown flag: $1" >&2 ; usage >&2 ; exit 2 ;;
        *)  PR_NUMBERS+=("$1") ; shift ;;
    esac
done

if [ "${#PR_NUMBERS[@]}" -eq 0 ]; then
    usage >&2
    exit 2
fi

for n in "${PR_NUMBERS[@]}"; do
    if ! [[ "$n" =~ ^[0-9]+$ ]]; then
        echo "PR number '$n' is not a number" >&2
        usage >&2
        exit 2
    fi
done

sweep_pr() {
    local pr_number="$1"
    local env_hash
    env_hash="$(compute_atlas_env "$pr_number")"
    ATLAS_ENV="$env_hash" ATLAS_STEP=init log info \
        "sweeping PR ${pr_number} (ATLAS_ENV=${env_hash}) apply=${APPLY}"

    # Phase implementations are added in Task 7. Each phase MUST:
    #   - read APPLY (0 = list-only, 1 = delete)
    #   - prefix each enumerated resource with its phase name
    #   - tolerate missing resources (idempotent)
    #   - skip the phase entirely if its required env vars are unset
    #
    # In DRY_RUN_NO_INFRA mode (testing), skip every infra call but still
    # emit the env_hash line above so the harness can grep for it.
    if [ -n "${DRY_RUN_NO_INFRA:-}" ]; then
        return 0
    fi

    sweep_pg "$pr_number" "$env_hash"
    sweep_kafka "$pr_number" "$env_hash"
    sweep_redis "$pr_number" "$env_hash"
    sweep_ghcr "$pr_number" "$env_hash"
    sweep_pihole "$pr_number" "$env_hash"
    sweep_app_finalizer "$pr_number" "$env_hash"
    sweep_branch "$pr_number" "$env_hash"
}

# Phase implementations (Task 7). Stubs that no-op so the skeleton is runnable.
sweep_pg()             { :; }
sweep_kafka()          { :; }
sweep_redis()          { :; }
sweep_ghcr()           { :; }
sweep_pihole()         { :; }
sweep_app_finalizer()  { :; }
sweep_branch()         { :; }

for n in "${PR_NUMBERS[@]}"; do
    sweep_pr "$n"
done

ATLAS_STEP=done log info "sweep complete"
