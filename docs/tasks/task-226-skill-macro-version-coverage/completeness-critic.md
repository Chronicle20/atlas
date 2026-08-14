# Completeness Critic — task-226-skill-macro-version-coverage

**Verdict: CLEAN.** 0 CHANGED-BUT-UNCLAIMED findings, 0 CLAIMED-BUT-UNVERIFIED findings. One informational note (not a finding) on a shared-tool change.

## Manifest status

No `docs/tasks/task-226-skill-macro-version-coverage/coverage-manifest.yaml` exists, and the plan never called for one. Claimed scope was instead derived from `prd.md` (FR-1..FR-5, Non-Goals, Definition of Done), `design.md` §1.5, and `plan.md` Tasks 1-10, cross-checked against `layout-derivation.md` and `na-recheck.md`. Derived claimed scope:

- Ops: `MACRO_SYS_DATA_INIT` (clientbound, `character/clientbound/CharacterSkillMacro`) and `SKILL_MACRO` (serverbound, `character/serverbound/CharacterSkillMacroHandle`).
- Versions: gms_v61, v72, v79, v83, v84, v87, v92, v95, jms_v185 verified; gms_v48 `n-a` for both ops with positive-absence evidence.
- gms_v61 `SKILL_MACRO` corrected off `n-a` (PRD FR-2, `na-recheck.md`), located at `CMacroSysMan::FlushToSvr` 0x59746c, opcode 101.
- Tooling linkage in `candidatesFromFName` (design §1.5, plan Task 9) and template opcode bindings (plan Task 8) are declared as required byproducts of promoting the two ops.

## Step 1/2 — CHANGED-BUT-UNCLAIMED

Diff base `723519dc4` (origin/main merge-base) → HEAD `6f644eddc`.

**Touched codecs** (`git diff --name-only -- 'libs/atlas-packet' | grep '\.go$' | grep -v _test`):
- `libs/atlas-packet/character/clientbound/skill_macro.go` — claimed (MACRO_SYS_DATA_INIT).
- `libs/atlas-packet/character/serverbound/skill_macro.go` — claimed (SKILL_MACRO).
- `libs/atlas-packet/character/skill_macro.go` — claimed (shared model/gate file per PRD "Boundaries" table: `character/skill_macro.go` gains per-version gates).
- `libs/atlas-packet/model/macros.go` — **deleted**, not a new touch outside scope; this is the pre-existing ungated model the PRD explicitly targets for replacement (PRD Non-Goals/Boundaries: the old `model.Macros`/`model.Macro` type is superseded by the new version-gated `character.SkillMacro`). No other packet references this file (verified: production callers in `services/atlas-channel` were repointed to the new codec in the same branch). CLAIMED.

No codec file outside the `skill_macro*` family was touched. No CHANGED-BUT-UNCLAIMED (codec) findings.

**Touched version gates** (`MajorVersion|MajorAtLeast|IsRegion|Region\(\)` diff hunks): all hunks are inside `libs/atlas-packet/character/skill_macro.go` / `clientbound/skill_macro.go` / `serverbound/skill_macro.go`, all attributable to the claimed macro codecs. No gate change was found in any file outside the claimed dirs. No CHANGED-BUT-UNCLAIMED (gate) findings.

**Matrix delta** (`git diff -- docs/packets/audits/status.json`): exactly two `op` rows changed state —
- `MACRO_SYS_DATA_INIT`: 9× `incomplete` → `verified` (the 9 claimed versions).
- `SKILL_MACRO`: 1× `n-a` (gms_v48, unchanged) → present as `n-a`; 1× `n-a` (gms_v61) → `verified`; 8× `incomplete` → `verified` (the other 8 claimed versions).

No other `op` row's `cells[...].state` changed anywhere in the diff (confirmed via full-file diff grep on `"op"`/`"state"` — only `MACRO_SYS_DATA_INIT`, `SKILL_MACRO` appear next to changed `state` lines; `USE_INNER_PORTAL`/`SPAWN_PET` appear only as unchanged context rows bracketing the edit). No CHANGED-BUT-UNCLAIMED (matrix) findings.

**Registry `fname` corrections** (`docs/packets/registry/gms_v61.yaml`, `gms_v72.yaml`, `gms_v79.yaml`): all three edits are scoped to the `SKILL_MACRO` serverbound row (opcode 109/101) — v61 adds the previously-missing entry (`CMacroSysMan::FlushToSvr` @0x59746c), v72/v79 correct a wrong fname (`sub_6022DB` → `CMacroSysMan::FlushToSvr`) with a documented misattribution note. All CLAIMED under FR-2.2.

**Seed-template bindings**: 7 templates changed (gms_61, 72, 79, 87, 92, 95, jms_185), matching the 7 non-production, non-v83/v84 in-scope versions (v83/v84 already carried both bindings pre-task — verified `template_gms_83_1.json`/`template_gms_84_1.json` have zero diff and already contain `CharacterSkillMacroHandle`/`CharacterSkillMacro`). v72/v79 changes are `fname`-only corrections (1 line each, no new binding); v61 adds one new handler binding (opcode 0x65, matching the corrected registry entry — the writer binding already existed pre-task, consistent with only `SKILL_MACRO` having been `n-a` there). `corpus_test.go`'s updated count (3151→3157, +6) and its inline comment enumerate exactly: `CharacterSkillMacroHandle` on gms_61/87/92/95/jms_185 (5) + `CharacterSkillMacro` writer on gms_92 (1) = 6 new bindings. All CLAIMED.

**IDA export splice** (`docs/packets/ida-exports/gms_v61.json` and 8 others): hand-splice of `CMacroSysMan::FlushToSvr`, `CWvsContext::OnMacroSysDataInit`, `CMacroSysMan::SetMacro`, `MACROSYSDATA::Encode/Decode` entries, documented in `harvest-log.md` per `VERIFYING_A_PACKET.md` §10's splice convention. Scoped to macro functions only. CLAIMED.

**Note (not a finding) — `tools/packet-audit/cmd/run.go` `selectCandidates` dedup-key change:** this is a shared-tool change with global reach (the dedup key changed from `pkg::name` to `pkg::name::dir` for every packet, not just macro). Verified its blast radius directly: grepped every `candidatesFromFName` case for `(name, pkg)` pairs reused across two directions — the only other pkg=`character` name reused twice is `SkillPrepare`, but both of its occurrences (`run.go:369`, `:840`) are the same direction (`DirServerbound`), so the old key already collapsed them to the identical candidate and the new key does too (no behavior change). `SkillMacro` is confirmed the only pkg+name pair that spans both directions, matching the code comment's claim. The change is additive/safe and was a necessary consequence of plan Task 9 ("link both macro codecs into packet-audit candidate resolution" — design §1.5 identifies the collision), even though the specific dedup-key mechanism isn't spelled out verbatim in plan.md's Task 9 text. Not flagging as CHANGED-BUT-UNCLAIMED because it is in-family tooling work directly serving the declared FR-3 linkage requirement and is verified non-destructive to every pre-existing candidate.

## Step 3 — CLAIMED-BUT-UNVERIFIED

All 18 claimed op×version cells (`MACRO_SYS_DATA_INIT` × 9 versions, `SKILL_MACRO` × 9 versions) are `verified` in the final `status.json`. All 18 have a matching evidence record under `docs/packets/evidence/<version>/character.{clientbound.CharacterSkillMacro,serverbound.CharacterSkillMacroHandle}.yaml`, each citing a `packet-audit:verify`-marked test function (18 markers confirmed present, matching 9 versions × 2 directions), and each has a matching report pair (`.json`+`.md`) under `docs/packets/audits/<version>/`.

The two `n-a` claims (gms_v48, both ops) have matching `feature-na-evidence.yaml` entries with positive-absence proof (dual-direction sibling cross-checks, per-op). No manifest/matrix mismatch: both cells are `n-a` in status.json and both are documented in `feature-na-evidence.yaml`.

No CLAIMED-BUT-UNVERIFIED findings.

## Summary

Every touched codec, gate, registry fname, template binding, and matrix-state transition on this branch maps to the task's declared scope (macro's two ops, the 9 in-scope versions, the v48 n-a pair, and the v61 SKILL_MACRO n-a correction). The one shared-tool change (`selectCandidates` dedup key) reaches beyond the macro packets in principle but was verified to be a no-op for every other existing candidate pair. The branch's actual footprint matches its declared scope.
