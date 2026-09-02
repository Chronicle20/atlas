# Plan Audit — task-251-player-npcs (Tasks 1-8)

**Plan Path:** docs/tasks/task-251-player-npcs/plan.md
**Audit Date:** 2026-08-22
**Branch:** task-251-player-npcs
**Base Branch:** main
**Scope:** Tasks 1-8 only (of a larger plan; tasks 9+ are owned by other shards)

## Executive Summary

All 8 tasks in this shard's range are fully implemented and match the plan's corrected
spec (§0 P-1..P-4) exactly, including the intentional Task-2 negative result. Every
module touched builds and its tests pass (`libs/atlas-constants`, `libs/atlas-object-id`,
`libs/atlas-packet` (npc/model packages), `services/atlas-data`, `services/atlas-configurations`,
and the `atlas-player-npcs` module build). The service-registration guard and both
kustomize overlay renders (`pr`, `main`) pass. No skipped or deferred work found in this range.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | `job.MaxLevelFor` in `libs/atlas-constants` | DONE | `libs/atlas-constants/job/max_level.go:8-19` — switch on `GetType`, every arm returns 200, matches plan's rationale comment verbatim. `libs/atlas-constants/job/max_level_test.go` (38 lines) is table-driven per plan spec. `go test ./...` from `libs/atlas-constants` passes. |
| 2 | Verify Cygnus level cap against WZ/job data | DONE | This was an evidence-search task, not a code task. `docs/tasks/task-251-player-npcs/progress.md:1-77` records both legs: (a) WZ grep sweep of `ms_1172`/`AtlasMS` `String.wz`/`Character.wz` for `blockedJobs\|maxLevel\|levelLimit` — zero matches, plus a secondary Cygnus/level/cap sweep turning up only flavor text; (b) IDB sweep (v95 session `ecc757f4`) of `is_cygnus_job`'s 15 call sites and a listing-wide regex sweep — no level-cap comparison found. Result recorded as "Not confirmed" and the table correctly left at 200 (commits `f202ee8ff`, `2db6d3a08`). No guessed value was landed. |
| 3 | `PlayerNpcObjectIdBase` in `libs/atlas-object-id` | DONE | `libs/atlas-object-id/reserved.go:1-33` — `PlayerNpcObjectIdBase = uint32(100000)`, `PlayerNpcObjectIdFor` with the `scriptId < 9900000` guard exactly as specified. `reserved_test.go` covers all 5 plan table rows. `go test ./...` passes. |
| 4 | `atlas-data` — `imitate` flag and batched ground endpoint | DONE | `services/atlas-data/atlas.com/data/npc/reader.go:69` parses `imitate`; `npc/rest.go:16` adds the field. `map/rest.go:330-373` adds the three rest models; `map/resource.go:45` registers `POST /{mapId}/ground` immediately after `/footholds/below`; handler at `resource.go:426-462` returns results in input order with `Found:false` for unmatched points, and 400s on an empty point list (`resource.go:429-432`). `reader_test.go` and `resource_ground_test.go` both present and passing (`TestHandleGetMapGroundRequest` PASS, including the empty-list-400 subtest). |
| 5 | `IMITATED_NPC_DATA` codec | DONE | `libs/atlas-packet/npc/clientbound/imitated_npc_data.go` — struct/Operation/Encode/Decode shape matches plan exactly (count-prefixed loop, no version gate, composes `packetmodel.Avatar`). `docs/packets/audits/STATUS.md:134` shows the row ✅ on all 9 applicable columns (v61/v72/v79/v83/v84/v87/v92/v95/jms_v185) and ⬜ on gms_v48 as required. Evidence files present under `docs/packets/evidence/<version>/npc.clientbound.ImitatedNpcData.yaml` for all 9 versions. `go test ./npc/... ./model/...` from `libs/atlas-packet` passes. |
| 6 | `REMOVE_NPC` codec | DONE | `libs/atlas-packet/npc/clientbound/remove.go` — single `WriteInt`/`ReadUint32`, `packet-audit:fname CNpcPool::OnNpcLeaveField` comment records per-version function-body-size confirmation for all 10 versions. `STATUS.md:304` shows ✅ on all 10 columns including gms_v48 (0x0B2). Evidence files present for all 10 versions (`gms_v48` through `jms_v185`). |
| 7 | Route both writers in the seed templates | DONE | Verified opcodes in 4 sampled templates (`gms_48_1` RemoveNPC-only 0xB2; `gms_61_1` both 0x4E/0xC3; `gms_83_1` both 0x51/0x102; `jms_185_1` both 0x55/0x117) match the plan's table exactly; `template_gms_12_1.json` has neither writer as specified. `seed_template_writers_test.go` new file present. `go test ./...` from `services/atlas-configurations/atlas.com/configurations` passes. |
| 8 | `atlas-player-npcs` service scaffold and registration | DONE | Full registration checklist present and verified: `.github/config/services.json` entry, `docker-bake.hcl:84`, `go.work:74`, `tools/db-bootstrap.sh:90`, `deploy/shared/routes.conf:561-562` + generated template, `deploy/k8s/base/{atlas-player-npcs.yaml,kustomization.yaml,env-configmap.yaml}`, main/pr overlay patches and kustomizations, `.bruno/` scaffold. `tools/service-registration-guard.sh` exits 0 (verified live). `kubectl kustomize deploy/k8s/overlays/{pr,main}` both render clean (verified live). `go build ./...` from the module root succeeds (verified live). The two operator hand-backs (create `atlas-player-npcs-main` DB; flip GHCR package public) are correctly recorded in `progress.md:86-100` rather than silently skipped — matches the plan's explicit instruction that these two steps cannot be completed by the task and must be handed to the operator. |

**Completion Rate:** 8/8 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None in this range. Task 8's two operator hand-backs (DB creation, GHCR package visibility)
are explicitly out-of-band operator actions per the plan text itself (§ "Two steps this task
cannot complete and must hand back to the operator") and are properly recorded in
`progress.md`, not silently dropped.

## Build & Test Results

| Service/Module | Build | Tests | Notes |
|---|---|---|---|
| `libs/atlas-constants` | PASS | PASS | all packages ok |
| `libs/atlas-object-id` | PASS | PASS | |
| `libs/atlas-packet` (npc/model scope) | PASS | PASS | `./npc/...` and `./model/...` |
| `services/atlas-data/atlas.com/data` | PASS | PASS | full package suite incl. `TestHandleGetMapGroundRequest` |
| `services/atlas-configurations/atlas.com/configurations` | PASS | PASS | includes new `seed_template_writers_test.go` |
| `services/atlas-player-npcs/atlas.com/player-npcs` | PASS (build only) | not run | module now contains tasks 9+ code as well; full test run is out of this shard's scope — build succeeded cleanly |
| Service-registration guard | PASS | n/a | `tools/service-registration-guard.sh` exits 0 |
| Kustomize renders | PASS | n/a | both `pr` and `main` overlays render clean |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (for this task range; overall branch readiness depends on the other shards)

## Action Items

None. All 8 tasks in this range are fully and faithfully implemented per the plan's
corrected spec (§0), with evidence in code, tests, generated packet-audit artifacts, and
`progress.md`.
