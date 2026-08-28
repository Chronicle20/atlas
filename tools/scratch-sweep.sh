#!/usr/bin/env bash
# tools/scratch-sweep.sh — age out stale scratch files from the disk-backed
# scratch root.
#
# Why the scratch root lives on disk, not on tmpfs: WSL2 sizes `/tmp` at 50%
# of the VM's RAM, so every stale agent scratch file left in `/tmp` is RAM
# taken away from the compilers running the same build. Moving scratch to a
# disk-backed root (default `/var/tmp/atlas/scratch`) frees that RAM; this
# script keeps that root from growing without bound.
#
# usage: tools/scratch-sweep.sh [--root <dir>] [--age-days <n>] [--now] [--dry-run]
#
#   --root <dir>     scratch root (default: $ATLAS_SCRATCH_ROOT, else
#                    /var/tmp/atlas/scratch)
#   --age-days <n>   remove entries older than n days (default: 7)
#   --now            equivalent to --age-days 0 — remove everything
#   --dry-run        print what would be removed; remove nothing
#   -h, --help       this message
#
# Exit codes:
#   0  swept (or nothing to sweep)
#   2  usage error, or the resolved root is a dangerous path

set -euo pipefail

die() { echo "scratch-sweep: $1" >&2; exit "${2:-2}"; }

root="${ATLAS_SCRATCH_ROOT:-/var/tmp/atlas/scratch}"
age_days=7
dry_run=0

while [ $# -gt 0 ]; do
    case "$1" in
        --root)      root="${2:?--root needs a directory}"; shift 2 ;;
        --age-days)  age_days="${2:?--age-days needs a number}"; shift 2 ;;
        --now)       age_days=0; shift ;;
        --dry-run)   dry_run=1; shift ;;
        -h|--help)   sed -n '2,22p' "$0"; exit 0 ;;
        -*)          die "unknown option $1" ;;
        *)           die "unexpected argument $1" ;;
    esac
done

[[ "$age_days" =~ ^[0-9]+$ ]] || die "--age-days must be a non-negative integer"

# Refuse a dangerous root before any deletion happens. `/`, `/tmp`, `/var/tmp`,
# and the home directory are all plausible values a broken $ATLAS_SCRATCH_ROOT
# could resolve to, and sweeping any of them would be catastrophic — this
# guard runs before the root is even created.
case "$root" in
    /|/tmp|/tmp/|/var/tmp|/var/tmp/) die "refusing to sweep dangerous root: $root" ;;
esac
if [ -n "${HOME:-}" ] && { [ "$root" = "$HOME" ] || [ "$root" = "${HOME}/" ]; }; then
    die "refusing to sweep dangerous root: $root (the home directory)"
fi
# Fewer than two path components (e.g. "/foo") is too shallow to trust.
components="$(echo "$root" | tr -s '/' '\n' | sed '/^$/d' | wc -l | tr -d ' ')"
if [ "$components" -lt 2 ]; then
    die "refusing to sweep dangerous root: $root (fewer than two path components)"
fi
# A relative root resolves against the caller's cwd, which is not something
# this script controls — require an absolute path so the resolved root is
# always exactly what was configured.
case "$root" in
    /*) ;;
    *) die "refusing to sweep dangerous root: $root (not an absolute path)" ;;
esac

if [ ! -d "$root" ]; then
    mkdir -p "$root"
    chmod 700 "$root"
    echo "scratch-sweep: removed 0 entries from $root"
    exit 0
fi
chmod 700 "$root"

find_args=("$root" -mindepth 1 -maxdepth 1)
if [ "$age_days" -ge 1 ]; then
    find_args+=(-mtime "+$((age_days - 1))")
fi

mapfile -t candidates < <(find "${find_args[@]}")

count=${#candidates[@]}
if [ "$dry_run" -eq 1 ]; then
    for c in "${candidates[@]}"; do
        printf '%s\n' "$c"
    done
    if [ "$count" -eq 1 ]; then
        echo "scratch-sweep: would remove 1 entry from $root"
    else
        echo "scratch-sweep: would remove $count entries from $root"
    fi
    exit 0
fi

for c in "${candidates[@]}"; do
    rm -rf -- "$c"
done

if [ "$count" -eq 1 ]; then
    echo "scratch-sweep: removed 1 entry from $root"
else
    echo "scratch-sweep: removed $count entries from $root"
fi
