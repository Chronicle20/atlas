# Plan Audit — task-250-inner-portal-registration

**Plan Path:** docs/tasks/task-250-inner-portal-registration/plan.md
**Audit Date:** 2026-08-21
**Branch:** task-250-inner-portal-registration
**Base Branch:** main (merge base `5f299e4bb`)
**HEAD:** `d051c2420`

## Executive Summary

All 12 plan tasks plus the controller-created Task 2b are implemented and match
their prescribed design closely, with two deliberate, well-documented
amendments carried through consistently: the scope grew from six versions to
ten (`scope-amendment.md`), and the codec correctly gates the `fieldKey` byte
with `MajorAtLeast(61)` rather than the "no gate required" ruling Task 1
originally reached. Every one of the ten in-scope versions (gms_v48, v61, v72,
v79, v83, v84, v87, v92, v95, jms_v185) has a seed-template route, a registry
`packet:` declaration, an evidence YAML, and a `packet-audit:verify` marker;
`STATUS.md:615` shows ✅ across all ten cells and gms_v12 correctly carries no
route. Module-local builds and tests are green for `libs/atlas-packet`,
`services/atlas-channel/atlas.com/channel`, `services/atlas-character/atlas.com/character`,
and `tools/packet-audit`, and `go run ./tools/packet-audit matrix --check`
exits 0. The branch also carries a `packet-audit` tool defect fix
(`SpliceExport`'s lossy round trip) that is out of task-250's domain scope but
was required to land this task cleanly; it ships with its own regression test
(`export_test.go`) and is not counted against this plan's completion.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Derive `TryRegisterTeleport` for v83/v84/v92, record six structures | DONE (superseded scope) | `structures/gms_v83.md`, `gms_v84.md`, `gms_v87.md`, `gms_v92.md`, `gms_v95.md`, `jms_v185.md` all present. Task's "no `MajorAtLeast` gate required" ruling was later superseded by Task 2b once v48's 5-field shape entered scope — documented, not silently dropped (`scope-amendment.md` "Consequences per plan task"). |
| 2 | Re-derive ⬜ columns and derive entry-distance threshold | DONE | `version-coverage.md` records gms_v12 absent (CSV `0x000` sentinel) and gms_v48/61/72/79 present via direct IDB derivation. `structures/gms_v95.md:62-71` carries `## Threshold derivation`; `maxPortalEntryDistance = 81` is used verbatim in `portal/processor.go:26`. |
| 2b | (controller-added) Full re-derivation + structure docs for v48/61/72/79 | DONE | `structures/gms_v48.md`, `gms_v61.md`, `gms_v72.md`, `gms_v79.md` present; `scope-amendment.md` records the ruling and rationale. Treated in-scope per dispatcher instruction. |
| 3 | `InnerPortal` codec in `libs/atlas-packet` (amended to 10 versions) | DONE | `libs/atlas-packet/portal/serverbound/inner_portal.go` implements `encodesFieldKey` with `t.MajorAtLeast(61)` gate (region GMS) or JMS; `inner_portal_test.go` table-drives all 10 versions with 5-field (v48) vs 6-field (all others) fixtures plus `TestInnerPortalRoundTrip` over `pt.Variants`. `go test ./...` from `libs/atlas-packet` passes. |
| 4 | `data/portal` model accessors + per-map cache | DONE | `data/portal/model.go` adds `Name/Target/Type/X/Y/TargetMapId/ScriptName`; `requests.go` replaces `requestInMapByName` with whole-list `requestInMap` (`portalsInMap`); `processor.go` adds `sync.Map` cache keyed by `{tenantId, mapId}` with double-checked load, mirroring `data/map`. `processor_test.go` covers caching, tenant isolation, not-found, and accessors — all pass. |
| 5 | `position` last-known-position registry + session clear-on-destroy | DONE | `position/registry.go` implements `Key/Position/Registry/GetRegistry/Put/Lookup/Clear` exactly per spec, package imports only `sync` + `atlas-tenant`. `session/processor.go:420-476` adds `clearLastPositionOnDestroy` called from `Destroy` beside `clearAranComboOnDestroy`. Tests pass in both `position` and `session` packages. |
| 6 | `movement.TeleportCharacter` + last-position write | DONE | `movement/processor.go` adds `TeleportCharacter` (interface + impl) writing the registry synchronously before the `routine.Go` Kafka publish with `fh=0, stance=0`; `ForCharacter` also writes the registry before its own publish. `movement/mock/processor.go` gains `TeleportCharacterFunc`. `movement/teleport_test.go` present, tests pass. |
| 7 | `atlas-character` preserve stored foothold on zero-`fh` movement | DONE | `temporal_data.go` `UpdatePosition` gains a `stance` param and preserves `existing.fh`; `processor.go` `Move` routes `fh == 0` to `UpdatePosition`, else `Update`. `temporal_position_test.go` covers all three plan cases. `go test ./character/...` passes. |
| 8 | `portal.EnterInner` validation and position adoption | DONE | `portal/processor.go` implements the 7-step refusal/accept sequence in plan order: unresolvable source, sentinel/wrong-map target, empty target, unresolvable destination, proximity-with-registry-miss-fallthrough (`int32` widened before squaring), claimed-target mismatch, accept via `newMovementProcessor(...).TeleportCharacter`. `maxPortalEntryDistance = 81` matches Task 2's derivation. `processor_test.go` covers all ten table rows from the plan. `portal/mock/processor.go` gains `EnterInnerFunc`. |
| 9 | Handler + `main.go` registration | DONE | `socket/handler/portal_inner.go` is a direct analogue of `portal_script.go`, tenant/fields sourced from `ctx`/session only. `main.go:902` registers `handlerMap[portal2.InnerPortalHandle] = handler.InnerPortalHandleFunc` beside the `PortalScriptHandle` line. |
| 10 | Route opcode in seed templates (amended to 10 templates) | DONE | All 10 templates (`template_gms_{48,61,72,79,83,84,87,92,95}_1.json`, `template_jms_185_1.json`) carry the `InnerPortalHandle` entry with `LoggedInValidator`, correct per-version opcode, and `fname: CUserLocal::TryRegisterTeleport`. `template_gms_12_1.json` correctly has no route. |
| 11 | packet-audit `candidatesFromFName` case + registry `packet:` declarations (amended to 10 registries) | DONE | `tools/packet-audit/cmd/run.go:2847-2857` adds the `CUserLocal::TryRegisterTeleport` case yielding `{name: "InnerPortal", pkg: "portal", dir: DirServerbound}`. All 10 registry YAMLs (`gms_v48/61/72/79/83/84/87/92/95.yaml`, `jms_v185.yaml`) carry `packet: portal/serverbound/PortalInnerPortal` on their `USE_INNER_PORTAL` entries; the four new registries (v48/61/72/79) also gained the op itself with opcode + fname, as the amendment required. |
| 12 | Splice exports, pin evidence, add verify markers, regenerate matrix (amended to 10 cells) | DONE | 10 export JSONs spliced (`docs/packets/ida-exports/gms_{v48,v61,v72,v79,v83,v84,v87,v92,v95}.json`, `gms_jms_185.json`), each adding exactly the `CUserLocal::TryRegisterTeleport` entry (verified by the branch's own multi-round key-level diffs recorded in `progress.md`, and consistent with `git diff --stat` line counts of 31-48 per file). 10 evidence YAMLs present under `docs/packets/evidence/<version>/portal.serverbound.PortalInnerPortal.yaml`, each tool-written with `--verifies` pointing at `TestInnerPortalGoldenBytes`. 10 `packet-audit:verify` markers in `inner_portal_test.go` with addresses matching the derived table. `STATUS.md:615` shows ✅ on all ten version columns. `matrix --check`, `fname-doc --check`, `operations --check` all exit 0 as independently re-run for this audit. |

**Completion Rate:** 13/13 tracked units (12 plan tasks + Task 2b) — 100%
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. All plan tasks and the controller-added Task 2b are fully implemented
with evidence in the diff.

## Out-of-plan but branch-carried work (not a task-250 defect)

The branch also contains a `packet-audit` tool fix (commits `3df557498`,
`9306fa4cc`, plus the matrix-regeneration follow-up `a175e81d7`):
`SpliceExport`'s unmarshal→marshal round trip in
`tools/packet-audit/internal/idasrc/{export.go,extract.go}` was silently
dropping `note`/`notes`/`region`/`_note` and an explicit `discriminator` key
on every entry it rewrote. `progress.md` (session 4) documents that this was
discovered because Task 12's export splices stripped provenance from ~1000
unrelated entries across the ten export files, and that the fix was required
before the branch could go green (`matrix --check` would not otherwise
promote cleanly). It is tested: `export_test.go` adds
`TestSpliceExportPreservesCuratedProvenanceKeys` and
`TestSpliceExportPreservesOmittedDiscriminator`, and both are exercised by the
`tools/packet-audit` test run below. This is correctly out of task-250's
plan-task scope and is not counted as a completion gap; it also does not
touch `services/atlas-channel`, `services/atlas-character`, or
`libs/atlas-packet`.

## Frontend surface

Confirmed not applicable: `git log 5f299e4bb..d051c2420 -- services/atlas-ui/`
is empty, and `git diff --stat 5f299e4bb..d051c2420 -- services/atlas-ui/`
returns no output. `progress.md` records that unrelated in-progress
`services/atlas-ui` edits (an unfinished `lastUpdatedAt`/`useGridRefresh`
change, not typechecking on its own) were present in the working tree mid-task
and were discarded on the user's explicit ruling, with a backup patch kept in
the session scratchpad. No task-250 commit touches atlas-ui.

## Build & Test Results

| Module | Build | Tests | Notes |
|---|---|---|---|
| `libs/atlas-packet` | PASS | PASS | Includes `portal/serverbound` (10-version golden-bytes + round-trip tests). |
| `services/atlas-channel/atlas.com/channel` | PASS | PASS | Includes `data/portal`, `movement`, `portal`, `position`, `session`, `socket/handler`. |
| `services/atlas-character/atlas.com/character` | PASS | PASS | `character` package (temporal position tests) confirmed green. |
| `tools/packet-audit` | PASS | PASS | 13/13 packages ok, including `internal/idasrc` (SpliceExport round-trip tests). |

Additionally re-run for this audit: `go run ./tools/packet-audit matrix --check`
exits 0 (2 informational `n-a evidence consumed` notes, unrelated to
USE_INNER_PORTAL).

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

None. The plan's scope-amendment history (six → ten versions, gate-required
ruling) is fully carried through code, registries, templates, evidence, and
the coverage manifest with no loose ends found in this audit.
