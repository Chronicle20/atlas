# Plan Audit — task-027-atlas-packet-v95-audit

**Plan Path:** docs/tasks/task-027-atlas-packet-v95-audit/plan.md
**Audit Date:** 2026-05-14
**Branch:** task-027-atlas-packet-v95-audit (HEAD = 8e805ded3)
**Base Branch:** main (428da0cc0)

## Executive Summary

The plan was honored and meaningfully exceeded. All 20 numbered tasks landed against
the worktree, with Tasks 13–15 (clientVariant plumbing) explicitly reverted under YAGNI
after the per-packet audit revealed no need for a tenant-side variant flag. The
expansion-scope items (28-packet login domain, Phase 2 sub-struct analyzer, four real
wire bugs, seven template opcode fixes plus the cross-version v83/v87/JMS v185 audit)
are all in the diff with file/line evidence. All claimed wire-bug fixes are present
in the code and the v83/v87/v95 template enum fix is in all three JSON files.
`go build`, `go vet`, and `go test -race` are clean across both `libs/atlas-packet/`
and `tools/packet-audit/`. The single ❌ remaining in the v95 audit run is the
analyzer-side `CharacterList` false positive, which is documented in
`docs/packets/ida-exports/_pending.md` and in `post-phase-b.md`.

The plan checkboxes were never ticked (0/136 boxes literally checked), but the
checked-state was clearly not the tracking mechanism — commits, audit reports, and
post-phase-b.md are the artifacts of record.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Tool skeleton + CLI flags | DONE | `tools/packet-audit/main.go`, `cmd/root.go`, `cmd/root_test.go`; commit `b458d4ef0` |
| 2 | CSV parser | DONE | `tools/packet-audit/internal/csv/csv.go` (180 lines); commit `94e17ce72` |
| 3 | Template parser | DONE | `internal/template/template.go` (104 lines) + real_test; commit `4dcb7a572` |
| 4 | FieldSource interface + ExportSource | DONE | `internal/idasrc/idasrc.go` + `export.go`; commit `5cb5d3553` |
| 5 | Seed v95 IDA export | DONE | `docs/packets/ida-exports/gms_v95.json`; commit `a5738f6a7` |
| 6 | MCPSource stub + export subcommand | DONE | `internal/idasrc/mcp.go`, `cmd/export.go` (stub returns 3); commit `cb700e0a3` |
| 7 | AST primitive call collector | DONE | `internal/atlaspacket/analyzer.go`; commit `64f93ec0e` |
| 8 | AST guard parsing | DONE | `internal/atlaspacket/guard.go` (172 lines) + tests; commit `1534145fc` |
| 9 | Sub-struct recursion + repeat markers | DONE | analyzer.go grew to 597 lines, `recurse_test.go`; commit `abef729a0` |
| 10 | Diff engine | DONE | `internal/diff/diff.go` (146 lines); commit `79e48e077` |
| 11 | Report writer (md + JSON) | DONE | `internal/report/report.go` (69 lines); commit `cc88795f5` |
| 12 | Wire pipeline + SUMMARY + exit codes | DONE | `cmd/run.go` (322 lines); SUMMARY.md exists at `docs/packets/audits/gms_v95/SUMMARY.md`; commit `12ae50d70` |
| 13 | `version/` helper package | REVERTED | Implemented in `fd4eec27a`, then dropped in `c64c8ad2e` (YAGNI per `7fb32b5c0`); documented in commit messages, not in `post-phase-b.md` |
| 14 | `tenant.Model.ClientVariant()` accessor | REVERTED | Implemented in `6b70efb31`, dropped in `7fb32b5c0`. No `ClientVariant` references remain in `libs/atlas-tenant/` |
| 15 | `clientVariant` template field | REVERTED | Implemented in `278568eed`, dropped in `7fb32b5c0`. No `clientVariant` strings remain in `services/atlas-configurations/` |
| 16 | Spike fix 1 — AuthSuccess field-7 width | DONE | `libs/atlas-packet/login/clientbound/auth_success.go:51-55`; commit `cc0ab921e` |
| 17 | Spike fix 2 — ServerListEntry per-channel worldId | DONE | `server_list_entry.go:75` `w.WriteByte(byte(m.worldId))`; commit `7f51e3e59` |
| 18 | Stub stock-v95 LoginHandle.Request slot | DONE (different shape) | `libs/atlas-packet/login/serverbound/request.go:64-66,80-82` adds v95-gated `unknown2` byte. The structurally rewritten stock-variant dispatch envisioned in the plan was scoped away with the clientVariant revert; the modified-v95 wire shape was confirmed against IDA and lives in `request.go` directly. Commit `89a863d4e`. |
| 19 | Per-login-packet audit matrix | EXCEEDED | 28 packets audited (vs plan's 29-row table; rows like `LoginAuth`/`PicResult`/`ServerLoad`/`ServerSelect` documented as not-exercised-by-v95-GMS in `post-phase-b.md`). Per-packet `.md`+`.json` artifacts live in `docs/packets/audits/gms_v95/` (57 files = 28 packets × 2 + SUMMARY). Commits `82ad0f902`, `27737d7a8`, `13a2891ce`, `0b7f913bf`, `bd1600ada`, `5d5a6e6cf`, `d39757c70`, `f4ca81a78`, `1e761572e`, `b93be270d`. |
| 20 | Post-Phase-B scoping checkpoint | DONE | `docs/tasks/task-027-atlas-packet-v95-audit/post-phase-b.md` (157 lines); commits `0e937b165` (initial), `4a12a0376` (expansion refresh), `8e805ded3` (cross-version section). |

**Completion Rate:** 17 DONE / 3 REVERTED-with-rationale / 0 SKIPPED-without-approval (17/20 = 85% landed-as-written; 100% accounted for)
**Skipped without approval:** 0
**Partial implementations:** 0 (Task 18 differs from the plan's blueprint but is intentionally simpler post-revert)

## Skipped / Deferred Tasks

- **Tasks 13–15 (clientVariant plumbing)** — reverted under YAGNI in commit `7fb32b5c0`. The audit-driven discovery of v95's real wire bugs all fit into `t.Region() == "GMS" && t.MajorVersion() >= 95` gates and didn't need a separate stock/modified variant axis. Impact: the plan's "structural drift" handling (Task 18 stock-variant dispatch) is not delivered, but the modified-v95 wire bug that motivated it (LoginHandle.Request `unknown2`) is fixed inline. This is mentioned in commit messages but is **not** called out in `post-phase-b.md`'s "Remaining work" section.
- **CharacterList ❌** — verdict left red in `docs/packets/audits/gms_v95/SUMMARY.md`. This is a known analyzer false positive (per-entry trailer counts loop-body conditional branches with early returns). Documented in `docs/packets/ida-exports/_pending.md` "Known false positives" section and in `post-phase-b.md` §"Remaining work" item 2. Runtime wire is verified correct.
- **MCP `export` subcommand** — landed as a stub returning exit code 3 with "not implemented"; called out as deferred in `post-phase-b.md` §"Remaining work" item 4.

## Wire-Bug Fix Verification

| Claim | Location | Verified |
|---|---|---|
| `ServerStatusRequest` int16 widening on GMS | `libs/atlas-packet/login/serverbound/server_status_request.go:36-40` (encode) + `:48-52` (decode) — `t.Region() == "GMS"` branch reads/writes uint16 | YES |
| `AuthPermanentBan` trailing-bytes skip on GMS | `libs/atlas-packet/login/clientbound/auth_permanent_ban.go:42-45` — `if t.Region() != "GMS"` guard around `WriteByte(0)` reason + `WriteLong(0)` timestamp; comment cites IDA v83-v95 verification | YES |
| `GW_CharacterStat` HP/MP int32 widening on v95+ | `libs/atlas-packet/model/character_statistics.go:113-123` (encode) + `:189-199` (decode) — `t.Region() == "GMS" && t.MajorVersion() >= 95` branch uses `WriteInt(uint32(m.hp))` etc.; comment "v95 widened HP/MaxHP/MP/MaxMP from int16 to int32 in GW_CharacterStat" | YES |
| `NEXON_ID_DIFFERENT_THEN_REGISTERED` 16→26 in template_gms_95_1.json | `services/atlas-configurations/seed-data/templates/template_gms_95_1.json:295` `"NEXON_ID_DIFFERENT_THEN_REGISTERED": 26` | YES |
| Same fix in template_gms_87_1.json | line 648 = 26 | YES |
| Same fix in template_gms_83_1.json | line 1209 = 26 | YES |

All four claimed wire fixes (plus the two cross-version template fixes) are present in the diff at the cited lines.

## Expansion-Scope Verification

| Expansion item | Evidence |
|---|---|
| Login-domain audit, 28 packets | 28 unique packet names × 2 (`.md`+`.json`) = 56 files + SUMMARY.md in `docs/packets/audits/gms_v95/` |
| Phase 2 sub-struct analyzer | `tools/packet-audit/internal/atlaspacket/registry.go` (`TypeRegistry`, `NewTypeRegistry`, `Calls`, `FieldType`) + `diff.FlattenWithRegistry` at `internal/diff/diff.go:127`. Commit `4998e070a` |
| Real `WorldBalloon` support | `libs/atlas-packet/model/world_balloon.go` (new file, 36 lines) wired into `server_list_entry.go:24-25,43,80-85,123-129`. Commit `5d5a6e6cf` |
| Cross-version v83/v87/JMS v185 audit | `post-phase-b.md` §"Cross-version login-domain audit"; v83 and v87 template fixes committed in `2771f3bd7`; commit `3c5c9e540` broadens v83/v95 wire fixes (e.g. `ServerStatusRequest`, `AuthPermanentBan`) to all GMS rather than gating on v95 |

## Build & Test Results

| Module | Build | Vet | Test (-race -count=1) | Notes |
|---|---|---|---|---|
| `libs/atlas-packet/` | PASS | PASS | PASS | All ~70 packages green; no FAIL output |
| `tools/packet-audit/` | PASS | PASS | PASS | 6 packages with tests (cmd, atlaspacket, csv, diff, idasrc, report, template); all green |

No docker-build cycle needed (no `services/atlas-*/go.mod` or `Dockerfile` touched — only `services/atlas-configurations/seed-data/templates/*.json`, which is data not consumed by the Go build).

## Overall Assessment

- **Plan Adherence:** MOSTLY_COMPLETE (17/20 landed as written, 3 reverted with explicit YAGNI rationale; expansion scope exceeded plan substantially)
- **Recommendation:** READY_TO_MERGE

The plan was honored; the deviations (Tasks 13/14/15 revert, Task 18 simplification) are justified by audit findings and committed under clear revert messages. The expansion items are all real, evidenced in the diff, and tested.

## Action Items

Optional / non-blocking:

1. **Document the clientVariant revert in `post-phase-b.md`.** The "Remaining work" section enumerates four follow-ups but does not explicitly note that the plan's Tasks 13–15 were intentionally dropped. A reader of the plan + post-phase-b.md without commit-log access would not know clientVariant was abandoned. Suggest adding a one-paragraph "Plan deviations" subsection.
2. **CharacterList ❌ in SUMMARY.md.** The single red verdict is a documented analyzer limitation, not a wire bug. Consider either (a) extending the analyzer per `post-phase-b.md` item 2, or (b) annotating the SUMMARY.md row with a "(false positive — see _pending.md)" suffix so casual readers don't see a red ❌ and assume the audit failed.
3. **MCP `export` subcommand stub.** Plan Task 6 documents this as a stub by design; `post-phase-b.md` item 4 carries it forward. No fix needed for this task; just track for future work.

None of these block merging the branch as-is.
