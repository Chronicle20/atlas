#!/usr/bin/env bash
# tools/lib/build-slot.sh — machine-wide build slot broker.
#
# A counting semaphore over K machine-wide slots, so N concurrent sessions
# (N worktrees, N terminals, a human plus CI) cannot all run a heavy build
# gate at once and thrash the host. Sourceable, not a standalone command,
# because tools/verify.sh (Task 7) holds a slot around the shell *function*
# `launch_go_layers` — a slot held by a subprocess would release the instant
# that subprocess exits, before the build it guards even starts.
#
# Why this blocks in `flock` rather than polling in a loop: CLAUDE.md forbids
# spending inference turns polling a process or waiting on a child agent. A
# blocking (or `flock -w`-bounded) `flock` call costs the kernel's own wait,
# not a spin loop burning turns to notice the lock freed.
#
# Slots are files under a machine-global directory (default
# /var/tmp/atlas/slots, overridable via ATLAS_SLOT_DIR) — global on purpose,
# so every worktree and every session on the host shares the same K slots.
# Each slot's lock lives on a file descriptor; the kernel drops it the moment
# the holding process dies, for any reason. There is deliberately no
# stale-lock cleanup path here — there is nothing that can go stale.
#
# Usage:
#   . tools/lib/build-slot.sh
#   acquire_build_slot "some label" || exit $?
#   ...heavy work...
#   release_build_slot
#
# Env:
#   ATLAS_SLOT_DIR          slot directory (default: /var/tmp/atlas/slots)
#   ATLAS_SLOT_THREADS      thread budget of one slot (default: 6); K and the
#                           verify.sh Go pool width both derive from it
#   ATLAS_BUILD_SLOTS       number of slots, positive integer (default:
#                           physical_cores / ATLAS_SLOT_THREADS, floored at 1
#                           — see _build_slot_default below)
#   ATLAS_BUILD_SLOT_TIMEOUT  seconds to wait before giving up; unset/empty
#                             means block forever

# Deliberately no `set -e`/`set -u` at file scope: this is sourced into
# callers (tools/verify.sh among them) that set their own shell options.

# Fixed fd for the held lock (fd 200). fd 0-2 are stdio; fd 9 is reserved by
# Task 8 for the module-cache lock. A fixed literal fd, rather than
# `exec {fd}>`'s dynamically-allocated one, keeps `shellcheck -S error`
# (tools/shell-guard.sh) clean.

# _build_slot_dir — resolves ATLAS_SLOT_DIR, defaulting to the machine-global
# scratch dir.
_build_slot_dir() {
    printf '%s\n' "${ATLAS_SLOT_DIR:-/var/tmp/atlas/slots}"
}

# _build_slot_physical_cores — physical cores on this host, not SMT threads.
#
# Go compilation scales with physical cores; SMT buys ~20-30%, not 2x. The
# original K=4 was sized from `nproc` (24) on a 12-core 5900X and was ~2x
# oversubscribed before any Claude session, docker, or k8s took a core.
# `lscpu -p` lists one row per logical CPU with its CORE,SOCKET pair; unique
# pairs are the physical cores. Falls back to nproc/2 when lscpu is absent
# (it is present on WSL2 and every Linux CI runner this repo uses).
_build_slot_physical_cores() {
    local cores=""
    if command -v lscpu >/dev/null 2>&1; then
        cores="$(lscpu -p=CORE,SOCKET 2>/dev/null | grep -v '^#' | sort -u | wc -l | tr -d ' ')"
    fi
    if [ -z "$cores" ] || [ "$cores" -lt 1 ] 2>/dev/null; then
        local logical
        logical="$(nproc 2>/dev/null || echo 2)"
        cores=$((logical / 2))
    fi
    [ "$cores" -lt 1 ] && cores=1
    printf '%s\n' "$cores"
}

# _build_slot_threads — the thread budget ONE slot is allowed to consume.
#
# This is the single number everything else derives from: K below, and
# tools/verify.sh's Go pool width (workers = slot threads / `go build -p`).
# Keeping it in one place is what stops K and the pool from being sized on
# different assumptions — the original K=4 assumed 6 threads per slot while
# the pool inside a slot ran 4 workers x 6 threads.
_build_slot_threads() {
    local t="${ATLAS_SLOT_THREADS:-6}"
    case "$t" in *[!0-9]* | '') t=6 ;; esac
    [ "$t" -lt 1 ] && t=1
    printf '%s\n' "$t"
}

# _build_slot_default — K when ATLAS_BUILD_SLOTS is unset.
#
# K = physical_cores / slot_threads, floored at 1, so K slots each consuming
# their full budget add up to the physical cores and no more. 12 cores at 6
# threads -> 2 slots; 24 cores -> 4; a 4-core laptop -> 1.
_build_slot_default() {
    local cores k
    cores="$(_build_slot_physical_cores)"
    k=$((cores / $(_build_slot_threads)))
    [ "$k" -lt 1 ] && k=1
    printf '%s\n' "$k"
}

# _build_slot_count — resolves and validates ATLAS_BUILD_SLOTS.
#
# Prints the count on stdout and returns 0, or prints nothing and returns 2
# when the value is not a positive integer.
_build_slot_count() {
    local n="${ATLAS_BUILD_SLOTS:-}"
    [ -z "$n" ] && n="$(_build_slot_default)"
    case "$n" in
        *[!0-9]* | '')
            echo "build-slot: ATLAS_BUILD_SLOTS must be a positive integer, got '$n'" >&2
            return 2
            ;;
    esac
    if [ "$n" -lt 1 ]; then
        echo "build-slot: ATLAS_BUILD_SLOTS must be a positive integer, got '$n'" >&2
        return 2
    fi
    printf '%s\n' "$n"
}

# acquire_build_slot <label>
#
# Blocks until a machine-wide build slot is free (or ATLAS_BUILD_SLOT_TIMEOUT
# elapses), then holds it until release_build_slot is called or this process
# exits. On success sets BUILD_SLOT to the acquired slot number (1..N) and
# returns 0. On an invalid ATLAS_BUILD_SLOTS returns 2. On timeout returns 75
# (EX_TEMPFAIL, sysexits.h — chosen so it cannot be confused with the
# eventual command's own exit status).
acquire_build_slot() {
    local label="$1"

    if ! command -v flock >/dev/null 2>&1; then
        echo "build-slot: 'flock' is not on PATH — cannot broker build slots" >&2
        return 2
    fi

    local n
    n="$(_build_slot_count)" || return 2

    local dir
    dir="$(_build_slot_dir)"
    mkdir -p "$dir"

    local i
    i=1
    while [ "$i" -le "$n" ]; do
        : > "$dir/slot.$i"
        i=$((i + 1))
    done

    SECONDS=0

    # First pass: try every slot without blocking.
    i=1
    while [ "$i" -le "$n" ]; do
        exec 200>"$dir/slot.$i"
        if flock -n 200; then
            BUILD_SLOT="$i"
            echo "build-slot: '$label' acquired slot $BUILD_SLOT after ${SECONDS}s" >&2
            return 0
        fi
        exec 200>&-
        i=$((i + 1))
    done

    # All slots busy: block on a deterministic slot for this process, rather
    # than every waiter thundering onto slot 1.
    local target
    target=$(( $$ % n + 1 ))
    exec 200>"$dir/slot.$target"

    if [ -n "${ATLAS_BUILD_SLOT_TIMEOUT:-}" ]; then
        if flock -w "$ATLAS_BUILD_SLOT_TIMEOUT" 200; then
            BUILD_SLOT="$target"
            echo "build-slot: '$label' acquired slot $BUILD_SLOT after ${SECONDS}s" >&2
            return 0
        fi
        exec 200>&-
        echo "build-slot: no build capacity for '$label' after ${SECONDS}s" >&2
        return 75
    fi

    flock 200
    BUILD_SLOT="$target"
    echo "build-slot: '$label' acquired slot $BUILD_SLOT after ${SECONDS}s" >&2
    return 0
}

# release_build_slot — releases the slot held by the current process, if any.
#
# Redirections are scoped to the { } group rather than a bare `exec`, so a
# missing fd (release called without a prior acquire) does not leak a
# permanent stderr redirect onto the rest of the calling shell.
release_build_slot() {
    { flock -u 200 && exec 200>&-; } 2>/dev/null || true
    unset BUILD_SLOT
}
