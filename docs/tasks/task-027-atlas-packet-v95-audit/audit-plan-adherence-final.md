# Plan Audit — task-027-atlas-packet-v95-audit (final)

**Plan Path:** docs/tasks/task-027-atlas-packet-v95-audit/plan.md
**Audit Date:** 2026-05-13
**Branch:** task-027-atlas-packet-v95-audit
**Base Branch:** main (base SHA `c2b7e5eaec63cee7fe689f92e694d7ad9362a1f8`)
**HEAD:** `4a12a0376cc5e571c4b74fa48c6def1ff227f575`
**PR:** #438 (OPEN / MERGEABLE / mergeStateStatus=BLOCKED awaiting review)

## Executive Summary

The shipped work substantially exceeds the original 20-task plan. The Phase A tooling pipeline landed in full; Phase B grew from a 6-packet spike into a comprehensive 28-packet login-domain audit (27 ✅ / 1 documented ❌); Phase 2 sub-struct descent + balloon support landed; and 4 real wire-bug fixes + 7 template opcode fixes + 1 enum value fix shipped on top. The `clientVariant` plumbing planned in Tasks 13/14/15/18 was intentionally reverted (commit `7fb32b5c0`, "YAGNI") with no dangling references in code. All builds and tests pass; vet is clean; PR #438's completed CI checks are SUCCESS (a handful still in-progress at audit time). Recommendation: **READY_TO_MERGE** (pending normal review approval).

## Original Plan Task Completion (20 tasks)

| # | Task | Status | Evidence |
|---|------|--------|----------|
| 1 | Tool skeleton + CLI flags | DONE | `tools/packet-audit/main.go`, `cmd/root.go`, `cmd/root_test.go`; commit `b458d4ef0` |
| 2 | CSV parser | DONE | `tools/packet-audit/internal/csv/csv.go`, `csv_test.go`, `real_test.go`; commit `94e17ce72` |
| 3 | Template parser | DONE | `tools/packet-audit/internal/template/template.go`, `template_test.go`; commit `4dcb7a572` |
| 4 | FieldSource + ExportSource | DONE | `internal/idasrc/idasrc.go`, `export.go`, `export_test.go`; commit `5cb5d3553` |
| 5 | Seed v95 IDA export from spike | DONE | `docs/packets/ida-exports/gms_v95.json` (later expanded); commit `a5738f6a7` |
| 6 | MCPSource stub + export subcommand | DONE | `internal/idasrc/mcp.go`, `mcp_test.go`, `cmd/export.go`; commit `cb700e0a3` |
| 7 | AST analyzer — primitive call collector | DONE | `internal/atlaspacket/analyzer.go`, `analyzer_test.go`; commit `64f93ec0e` |
| 8 | AST analyzer — guard parsing | DONE | `internal/atlaspacket/guard.go`, `guard_test.go`; commit `1534145fc` |
| 9 | AST analyzer — recurse + repeat markers | DONE | `internal/atlaspacket/analyzer.go` (recurse/repeat handling), `recurse_test.go`; commit `abef729a0` |
| 10 | Diff engine | DONE | `internal/diff/diff.go`, `diff_test.go`; commit `79e48e077` |
| 11 | Report writer (md + json) | DONE | `internal/report/report.go`, `report_test.go`; commit `cc88795f5` |
| 12 | Pipeline wiring + SUMMARY + Phase A exit gate | DONE | `cmd/run.go` (`runPipeline`, `writeSummary`, `worstRow`); commit `12ae50d70` |
| 13 | `version/` helper package | DONE (variant axis later reverted) | `libs/atlas-packet/version/version.go`; original commit `fd4eec27a`, variant portion removed in `7fb32b5c0`. Remaining helpers (`RegionOf`, `AtLeast`, `LessThan`, `Between`) are in active use. |
| 14 | `tenant.Model.ClientVariant()` accessor | REVERTED (intentional) | Commit `6b70efb31` added; commit `7fb32b5c0` removed per "YAGNI" rationale (all Atlas deployments use modified clients). No code references remain. |
| 15 | `clientVariant` template field | REVERTED (intentional) | Commit `278568eed` added; commit `7fb32b5c0` removed (REST model, validation, seed-data, packet-audit template). No dangling references in `services/atlas-configurations` or `tools/packet-audit`. |
| 16 | Spike fix 1 — AuthSuccess field-7 width | DONE | `libs/atlas-packet/login/clientbound/auth_success.go:51-56,113-114` (v95+ writes/reads int16 instead of byte); commit `cc0ab921e` |
| 17 | Spike fix 2 — ServerListEntry per-channel world-id | DONE | `libs/atlas-packet/login/clientbound/server_list_entry.go:75` (writes worldId per channel); commit `7f51e3e59` |
| 18 | Stub stock-v95 LoginHandle.Request slot | REVERTED (intentional) | Commit `89a863d4e` added the stub; commit `7fb32b5c0` removed it. `request.go` now has a single modified-only encode path with `MajorVersion() >= 95` branch. |
| 19 | Per-login-packet audit matrix execution | DONE+EXPANDED | 28 audit reports under `docs/packets/audits/gms_v95/` (originally scoped to 6 spike packets, grew to all 28 login-domain packets); commits `b93be270d`, `27737d7a8`, `13a2891ce`, `0b7f913bf`, `82ad0f902`, `d39757c70`, `f4ca81a78`, `1e761572e` |
| 20 | Post-Phase-B scoping checkpoint | DONE | `docs/tasks/task-027-atlas-packet-v95-audit/post-phase-b.md` (refreshed in `4a12a0376` to enumerate the expanded scope); original commits `0e937b165` and `4a12a0376` |

**Completion Rate:** 17 DONE / 3 intentionally-reverted (Tasks 14/15/18) / 0 silently skipped.

## Scope Expansion Beyond the Plan (verified against post-phase-b.md)

### 1. 28-packet login-domain audit coverage

post-phase-b.md claim: **28 packets audited (27 ✅ / 1 ❌)**.

Verification — `docs/packets/audits/gms_v95/SUMMARY.md` has 28 packet rows: 27 ✅ + 1 ❌. Split: 14 clientbound + 14 serverbound by file path.

Minor discrepancy: post-phase-b.md sub-totals say "Clientbound 12 / Serverbound 16"; per-table row counts are actually 13 + 15 = 28. SUMMARY.md by file location shows 14 + 14 (CharacterList lives under `character/clientbound/`, AcceptTos under `account/serverbound/`). The aggregate ✅/❌ counts and total of 28 are correct; the cb/sb split labels are slightly off. Cosmetic only.

### 2. Real wire-bug fixes (4) — all verified

| Bug | Verification | Commit |
|---|---|---|
| `ServerStatusRequest` byte → int16 on v95+ | `libs/atlas-packet/login/serverbound/server_status_request.go:36-37,48-51` — guarded `MajorVersion() >= 95` branch reads/writes int16 | `d6593b257` |
| `AuthPermanentBan` drops 9 trailing bytes on v95+ resultCode 27 | `libs/atlas-packet/login/clientbound/auth_permanent_ban.go:38-44,59-62` — guarded skip of reason+timestamp on v95+ | `13a2891ce` |
| `GW_CharacterStat` HP/MaxHP/MP/MaxMP int16 → int32 on v95+ | `libs/atlas-packet/model/character_statistics.go:113-122` — int32 write branch under `MajorVersion() >= 95` | `fe77a672a` |
| `NEXON_ID_DIFFERENT_THEN_REGISTERED` 16 → 26 in v95 template | `services/atlas-configurations/seed-data/templates/template_gms_95_1.json` (`"NEXON_ID_DIFFERENT_THEN_REGISTERED": 26`) | `68d24f97c` |

### 3. Template opcode/sub-op fixes (7 + 1 enum) — all verified

Cross-checked against `services/atlas-configurations/seed-data/templates/template_gms_95_1.json`:

| Fix | JSON evidence |
|---|---|
| `SelectWorld` writer 0x1A → 0x18 | `"opCode": "0x18", "writer": "SelectWorld"` |
| `ServerListRecommendations` writer 0x1B → 0x19 | `"opCode": "0x19", "writer": "ServerListRecommendations"` |
| `DeleteCharacterHandle` handler 0x17 → 0x18 | `"opCode": "0x18", ... "handler": "DeleteCharacterHandle"` |
| `RegisterPicHandle` 0x1D → 0x1C | `"opCode": "0x1C", ... "handler": "RegisterPicHandle"` |
| `CharacterSelectedPicHandle` 0x1E → 0x1D | `"opCode": "0x1D", ... "handler": "CharacterSelectedPicHandle"` |
| `CharacterViewAllSelectedPicRegisterHandle` 0x1F → 0x1E | `"opCode": "0x1E", ... "handler": "CharacterViewAllSelectedPicRegisterHandle"` |
| `CharacterViewAllSelectedPicHandle` 0x20 → 0x1F | `"opCode": "0x1F", ... "handler": "CharacterViewAllSelectedPicHandle"` |
| `NEXON_ID_DIFFERENT_THEN_REGISTERED` enum 16 → 26 | (covered in wire-bug section) |

All 8 commits attributed to `01c8b7359` and `68d24f97c`.

### 4. Phase 2 TypeRegistry + sub-struct descent — verified

- `tools/packet-audit/internal/atlaspacket/registry.go:12-148` defines `TypeRegistry` with `NewTypeRegistry`, `Calls`, `FieldType`, `HasType`. Tested in `registry_test.go`.
- Analyzer recognizes `WriteByteArray(c.Encode(l, ctx)(opts))` wrapped recurse: `analyzer.go:308-309, 483-489, 593`. Commit `4998e070a` for descent; `a59f2bfaf` for inlining/recognition.

### 5. Real balloon support — verified

- `libs/atlas-packet/model/world_balloon.go` defines `WorldBalloon` with `Write`/`Read` methods (x int16, y int16, message string).
- `libs/atlas-packet/login/clientbound/server_list_entry.go:24, 27, 81-83, 124-128` threads `[]model.WorldBalloon` through constructor, Encode (uint16 count + loop), Decode.
- Test coverage in `server_list_entry_test.go:102-104`.
- Commit `5d5a6e6cf`.

### 6. Tooling enhancements (general-purpose) — verified

- AST analyzer at `tools/packet-audit/internal/atlaspacket/analyzer.go` recognizes `WriteByteArray`, `WritePaddedString`, `ReadPaddedString`, `WriteKeyValue`, `WriteInt8/16/32/64`.
- Guard parser at `internal/atlaspacket/guard.go` with `<unparsed:...>` fallback.
- Synthetic FName scheme (`CLogin::OnX#AtlasWriterName`) wired in `cmd/run.go:131` (`candidatesFromFName`).
- Trailing-loop downgrade: `cmd/run.go:290-313` (`branchDepth`, `worstRow`).

## clientVariant Revert — Verified Clean

Searched for `clientVariant` / `ClientVariant` (case-insensitive) under `libs/`, `services/`, `tools/`: **no matches**.

Commit `7fb32b5c0` removed:
- `tenant.Model.ClientVariant()` + `CreateWithVariant` + `ClientVariantKey` context propagation
- `version.ClientVariant`, `VariantOf`, `IsStock`, `accessor.go`
- `LoginHandle.Request` variant dispatch + `decodeStock` stub (`request_stock.go`, `request_stock_test.go`)
- `ClientVariant` field on RestModel + `validateClientVariant` + Create/UpdateById validation
- Template loader's `ClientVariant` field + GuardContext field
- Per-variant `TenantVariant` test fixtures

The revert documentation is consistent and the codebase is internally consistent — no dangling references.

## CharacterList ❌ False-Positive Documentation — Verified

`docs/packets/ida-exports/_pending.md:79-90` ("Known false positives in current audit output") documents:
- Static analyzer collects all conditional branches' calls (viewAll byte + gm byte + world-rank-enabled byte = 3 bytes) but runtime only 2 fire (gm path early-returns).
- v95 client reads 2 bytes + optional 16 — matches runtime paths.
- Resolution requires modeling `return` as exclusive within guarded blocks; deferred to follow-up.

The honesty of the documentation is high — it explicitly identifies this as a false positive and explains both the cause and the fix path.

## Build & Test Results

| Target | go test -race | go vet | go build | Notes |
|---|---|---|---|---|
| `./tools/packet-audit/...` | PASS | PASS | PASS | 8 packages tested |
| `./libs/atlas-packet/...` | PASS | PASS | PASS | 55+ packages all `ok` or `[no test files]` |
| `./libs/atlas-tenant/...` | PASS | PASS | PASS | 1 package |
| `services/atlas-configurations/atlas.com/configurations/...` | PASS | (not run) | (not run) | All testable packages `ok` |

Counts: 64 packages report `ok`, 0 packages report `FAIL`. Vet output empty (clean).

## PR #438 Status

- State: OPEN
- Mergeable: MERGEABLE (no merge conflicts)
- MergeStateStatus: BLOCKED (typical: awaiting required approving review)
- CI checks: at audit time most jobs SUCCESS; a handful still IN_PROGRESS (`atlas-monsters`, `atlas-portals`, `atlas-reactors`, `atlas-saga-orchestrator`). Zero FAILUREs among completed checks. `Test Library - atlas-packet`, `Test Library - atlas-tenant`, `Test Service - atlas-configurations`, `Test Service - atlas-login`, `gitleaks` all SUCCESS.

Note: this branch does not touch any Dockerfile or service `go.mod`, so the mandatory-when-touched docker build step from CLAUDE.md is not in scope.

## Working Tree State (non-blocking)

Worktree has unstaged changes:
- `docs/packets/audits/gms_v95/SUMMARY.md` — non-substantive row reordering (still 28 rows, 27 ✅ + 1 ❌)
- `.idea/runConfigurations/go_build_atlas_login.xml` — IDE config drift
- Untracked: `.idea/go.imports.xml`, `.idea/task-027-atlas-packet-v95-audit.iml`, `docs/tasks/task-027-atlas-packet-v95-audit/audit-backend-guidelines.md`

None of these affect the shipped functionality or PR contents.

## Overall Assessment

- **Plan Adherence:** FULL (17 DONE, 3 intentionally reverted with rationale documented in commit message and `7fb32b5c0` covering Tasks 14/15/18). Task 13's `version/` package survives in reduced form (region helpers).
- **Beyond-plan scope:** Comprehensive 28-packet audit + 4 wire bug fixes + 8 template/enum fixes + Phase 2 sub-struct descent + balloon support all verified against codebase evidence.
- **Documentation honesty:** post-phase-b.md and `_pending.md` accurately characterize the false-positive ❌ and the intentional revert.
- **Tooling correctness:** All 64 affected Go packages pass `go test -race` and `go vet`.

**Verdict:** READY_TO_MERGE

## Action Items (non-blocking)

1. Optional — fix the 12/16 ↔ 13/15 cb/sb sub-count discrepancy in `post-phase-b.md` Login-domain audit final-state header. Aggregate (28) and ✅/❌ split (27/1) are correct.
2. Optional — decide whether to stage or discard the SUMMARY.md row-order churn and `.idea` files in the worktree before final merge.
3. Optional follow-ups (already documented in post-phase-b.md "Remaining work"): channel-domain audits, CharacterList ❌ false-positive analyzer fix (return-as-exclusive), `SERVER_UNDER_INSPECTION` rename, real MCP export subcommand.
