#!/usr/bin/env bash
# script-ops-guard.sh — enforces that the sixteen shared script operations
# have exactly one implementation, in libs/atlas-script-core/ops.
#
# task-300 pulled the payload-construction logic shared by the script
# operation tables of atlas-map-actions, atlas-portal-actions,
# atlas-reactor-actions, and atlas-npc-conversations into
# libs/atlas-script-core/ops. A script operation table that constructs one
# of these saga payloads directly — instead of calling into ops — is a
# second implementation: exactly the drift this task removed. This guard
# keeps it from growing back one feature at a time.
#
# Scope is deliberately the four script-operation-table services, not the
# whole repo: every other service also builds these same saga payload
# types, but for its own domain logic (e.g. atlas-quest starting a reward
# quest, atlas-consumables granting a skill) — not by re-interpreting a
# script's operation parameters the way ops.* does. Banning that reuse
# repo-wide would not catch operation-table drift; it would just misfire on
# unrelated producers. atlas-saga-orchestrator is additionally out of
# scope: it legitimately *consumes* these payloads (it is the saga step
# handler that decodes and executes them).
#
# Within the four services, a handful of call sites build one of these
# payloads for a feature that is NOT the script operation table and cannot
# use ops.* (its function signature takes script params `p map[string]string`
# plus a Resolver/Target — inputs these call sites don't have). Those lines
# carry an inline `// script-ops-guard:allow — <reason>` marker, checked at
# the call site so the exemption's rationale travels with the code instead
# of living as a path in this script that a reviewer can't see next to the
# construction it excuses.
#
# The local import alias for the saga message package (commonly `saga`) may
# differ per file, so the guard matches the payload type name preceded by
# any package qualifier: `[A-Za-z_][A-Za-z0-9_]*\.SpawnMonsterPayload{` etc.
# `//`-comment lines, and lines carrying the `script-ops-guard:allow`
# marker, are excluded from matching.
#
# Run from the repo root; a hit → non-zero exit.
set -euo pipefail

ROOT="${SCRIPT_OPS_GUARD_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}"

SERVICE_DIRS=(
    "$ROOT/services/atlas-map-actions"
    "$ROOT/services/atlas-portal-actions"
    "$ROOT/services/atlas-reactor-actions"
    "$ROOT/services/atlas-npc-conversations"
)

PAYLOADS=(
    SendMessagePayload
    SpawnMonsterPayload
    MoveEnvironmentPayload
    ResetEnvironmentPayload
    ShowIntroPayload
    ShowHintPayload
    PlayPortalSoundPayload
    ApplyConsumableEffectPayload
    CreateSkillPayload
    UpdateSkillPayload
    WarpToPortalPayload
    WarpToSavedLocationPayload
    SaveLocationPayload
    StartInstanceTransportPayload
    StartQuestPayload
    StageClearAttemptPqPayload
)

rc=0
for payload in "${PAYLOADS[@]}"; do
    pattern="[A-Za-z_][A-Za-z0-9_]*\\.${payload}{"
    for dir in "${SERVICE_DIRS[@]}"; do
        [ -d "$dir" ] || continue
        while IFS= read -r hit; do
            [ -n "$hit" ] || continue
            file="${hit%%:*}"
            rest="${hit#*:}"
            line="${rest%%:*}"
            rel="${file#"$ROOT"/}"
            echo "script-ops-guard: FAIL — ${rel}:${line} constructs ${payload} directly; build it via libs/atlas-script-core/ops"
            rc=1
        done < <(grep -rn -E "$pattern" "$dir" \
            --include='*.go' \
            2>/dev/null \
            | grep -v -E '^[^:]+:[0-9]+:[[:space:]]*//' \
            | grep -v -F 'script-ops-guard:allow')
    done
done

if [ "$rc" -ne 0 ]; then
    exit "$rc"
fi

echo "script-ops-guard: OK — no shared script-operation payload constructed under the script-operation-table services."
