# Backend Audit — task-069 (misc-domain packet audit)

- **Scope:** Changed Go files only, diff `8ef9d9fb5..7edcff74d`
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-06-03
- **Module(s):** `libs/atlas-packet` (shared lib), `tools/packet-audit` (tooling)
- **Build:** PASS
- **Tests:** PASS (race), all packages green
- **Vet:** PASS
- **Overall:** PASS

## Backend guidelines review

### Objective gate (Phase 1)

| Check | Status | Evidence |
|-------|--------|----------|
| `go build ./libs/atlas-packet/... ./tools/packet-audit/...` | PASS | exit 0 |
| `go test -race -count=1` (changed pkgs) | PASS | stat/clientbound 1.012s, socket/serverbound 1.012s, packet-audit/cmd 1.996s |
| `go vet ./libs/atlas-packet/... ./tools/packet-audit/...` | PASS | exit 0 |

### Scope note

atlas-packet is a wire-encoding shared library, not a DDD domain service: no
`model.go`/`builder.go`/`processor.go`/`administrator.go`/`resource.go`, no GORM,
no REST, no Kafka, no DB. The DOM-01..DOM-20, DOM-22..DOM-24, SUB-*, EXT-*,
SCAFFOLD-*, and SEC-* checklists are N/A by construction. The applicable rules
are: tenant/version-gating idiom, DOM-21 (no reinventing shared types/constants),
test quality, and the atlas-packet house rules (no `reflect`, no new
`interface{}` params, no benchmarks, immutable models).

### Applicable checks

| Check | Status | Evidence |
|-------|--------|----------|
| Tenant extraction via `tenant.MustFromContext(ctx)` | PASS | `stat/clientbound/changed.go:48,105`; `socket/serverbound/channel_connect.go:54,73` |
| Version-gate idiom matches established codebase pattern | PASS | `changed.go:51,106` use `t.Region() == "GMS" && t.MajorVersion() >= 95`, identical to `chat/serverbound/whisper.go:60`, `cash/serverbound/item_use.go:38`, `character/clientbound/attack.go:107`, `login/serverbound/request.go:64`. JMS gate `channel_connect.go:61,78` uses `t.Region() == "JMS"`. |
| `ctx` now consumed (was `_`-discarded before) | PASS | Encode/Decode signatures bind `ctx` and use it; previously ignored. |
| Encode/Decode are symmetric across all gates | PASS | stat: int16↔int32 + trailing byte mirrored (`changed.go:74-79`/`125-130`, `95-98`/`147-150`); channel_connect: uint16↔byte mirrored (`channel_connect.go:61-65`/`78-82`). |
| DOM-21 — no reinvented shared types/constants | PASS | No new domain type/alias/numeric-classification introduced. Stat codes use `atlas-constants/stat.Type` (`changed.go:8`). `"GMS"`/`"JMS"` literals are the established convention (no shared region constant exists; matches all sibling files). `boolToUint8` (`channel_connect.go:88`) is a 4-line local helper with no shared equivalent — acceptable. |
| No `reflect` | PASS | grep clean across all 5 changed Go files. |
| No NEW `interface{}` params | PASS | Only `interface{}` occurrences are the pre-existing `options map[string]interface{}` Encode/Decode signature convention and the `statistics` option type-switch — not introduced by this change. |
| No benchmarks added | PASS | grep `func Benchmark` clean. |
| Immutable models / builder conventions | PASS (N/A) | No new exported types. `Changed`/`Update`/`ChannelConnect` keep private fields + getters; no setters introduced. |
| `tools/packet-audit/cmd/run.go` changes | PASS | Pure additive `case` arms in `candidatesFromFName` mapping IDA FNames → struct names (`run.go:195-265`). No analyzer/logic change. |

### Test quality

| Aspect | Status | Evidence |
|--------|--------|----------|
| Table-driven over all tenant variants | PASS | `pt.Variants` (GMS v28/v83/v95, JMS v185) driven via `t.Run` in `changed_test.go:25,51,98` and `channel_connect_test.go:12,63`. |
| Byte-level wire-shape assertions (not just symmetry) | PASS | `TestStatChangedV95WireWidths` (`changed_test.go:83-94`) asserts v95=11 bytes vs v83=8 bytes — catches the wrong-but-symmetric pre-fix bug the round-trip tests cannot. `TestChannelConnectWireShape` (`channel_connect_test.go:53-90`) asserts JMS=31 vs GMS=30 bytes and reads the gm field as LE uint16 at offset 20. |
| Round-trip completeness | PASS | `pt.RoundTrip` asserts zero unconsumed bytes (`test/round_trip.go:49`), so a trailing-byte mismatch would fail. |

## Findings

### Blocking
- None.

### Major
- None.

### Minor / advisory (non-blocking, NOT guideline violations)
- The v95 second trailing flag byte and HP/MP widening are gated `MajorVersion() >= 95` while the inline comments (`changed.go:91-94`) explicitly flag v83/v87/JMS as "pending verification." This is correctly conservative: pre-v95 GMS and JMS keep the legacy 1-byte/int16 shape, so behavior for unverified versions is unchanged. No action required; called out only so the open verification item is tracked.

## Overall assessment

PASS. The two wire fixes correctly consume the previously-discarded `ctx`, gate
behind the project-standard `Region()`/`MajorVersion()` idiom, keep Encode/Decode
symmetric, and are proven at the byte level (not merely by round-trip symmetry,
which the prior wrong-but-symmetric code would have passed). No DOM-21 type
duplication, no `reflect`, no new `interface{}` params, no benchmarks. The
`packet-audit` tool change is purely additive FName→struct mapping. Build, race
tests, and vet are all clean.

---

## Plan adherence review

**Reviewer:** plan-adherence-reviewer
**Date:** 2026-06-03
**Branch:** task-069-misc-domain-packet-audit
**Base commit:** 8ef9d9fb5 (plan commit) → HEAD 7edcff74d

### Documented plan corrections honored (NOT counted as gaps)

All four are documented in `context.md` "EXECUTION CORRECTION" blocks and/or `post-phase-b.md` §"Plan corrections":

1. `--output docs/packets/audits` (tool appends `<region>_v<major>` at run.go:42) — verified: audit dirs `gms_v83/`, `gms_v87/`, `gms_v95/`, `jms_v185/` exist, no nested `gms_v95/gms_v95/`.
2. Phase 1 (Tasks 2–4) skipped — `_body.go` files are dispatcher helpers, not registry-resolvable struct bodies. Documented in commit `195a1fc9c` + context.md. Correct decision; no registry fixtures applicable.
3. Phase 2/3 methodology is FName × `candidatesFromFName`-driven, not template-driven (template tables are dead code). Real work = `gms_*.json` + run.go `candidatesFromFName` edits. Documented.
4. Regression gate is semantic (sorted packet→verdict set), not byte-identical (SUMMARY map-iteration order is non-deterministic). Documented.

### Per-phase verdicts

| Phase / sub-phase | Status | Evidence |
|---|---|---|
| Phase 0 — regression baseline | PASS | Commit `2ddb5c72e` normalized 28 login reports. Login verdicts unchanged: `CharacterList` remains the sole login-era ❌; all other login rows ✅. |
| Phase 1 — TypeRegistry fixtures | PASS (correctly skipped) | Commit `195a1fc9c` documents the skip with verified rationale (`NewTypeRegistry` resolves none of the assumed body type names; encoders write primitives directly). Documented correction #2. |
| 2a tool/ (0 packets) | PASS | Commit `8ff28ddae`; `_pending.md:117` "Tool domain — utility-only (task-069)". Zero SUMMARY rows, correct. |
| 2b stat (1) | PASS + real fix | Commit `c451289b4` (bucket) + `79abbb0d8` (fix: HP/MP int16→int32 + 2nd trailing byte, gated `Region=="GMS" && MajorVersion()>=95`, changed.go:49-148). `Changed` row present (❌ = documented mask-driven analyzer artifact, real wire bugs fixed). |
| 2c channel (2) | PASS | Commit `d1a2b2d6f`. `ChannelChange.md` + `ChannelChangeRequest.md` present. ChannelChange SUMMARY row ❌ is the documented `locateAtlasFile` collision (resolves to buddy file); packet verified manually (`_pending.md:127`). |
| 2d ui (3) | PASS | Commit `37391a20b`. `Open`, `Disable`, `Lock` rows all ✅. |
| 2e fame (4) | PASS | Commit `5668e4960`. `ReceiveResponse`, `GiveResponse`, `ErrorResponse`, `Change` rows present. GiveResponse ❌ = documented `WriteInt16+WriteShort(0)==int32` artifact. |
| 2f merchant (7) | PASS | Commit `20f1f8a99`. All 7 employee-shop structs present: OpenShop, ErrorSimple, ShopSearch, ShopRename, RemoteShopWarp, ConfirmManage, FreeFormNotice (all ✅). Hire-merchant → task-067 cross-link; bare serverbound handler deferred (`_pending.md:143`). |
| 2g quest (4 audited + 3 deferred) | PASS | Commit `7a1b91c3b`. Audited: ScriptProgress, Action, ActionScriptStart, ActionScriptEnd. ActionStart/ActionComplete/ActionRestoreLostItem deferred to `_pending.md:157` (require atlas-channel `quest_action.go` handler change — out of libs scope). See note below re: brief's "quest(5)". |
| 2h account (2 new) | PASS | Commit `339256e8d`. `RegisterPin` + `SetGender` rows ✅ (AcceptTos pre-existing from task-027). |
| 2i socket (5) | PASS + real fix | Commit `2cdeda8fe`. Hello, Ping, ChannelConnect, Pong, StartError rows present. JMS ChannelConnect gm field fix (Encode1→Encode2 for JMS) in commit `8c276e0c6` (channel_connect.go:58-78). Hello/ChannelConnect ❌ = documented width-label artifacts. |
| 2j _pending sweep | PASS | Commit `0dc7adebb`. `_pending.md` has canonical per-domain headings (tool, locateAtlasFile collisions, bare handlers, merchant modes, quest, Phase 3 TODOs). |
| Phase 3 — v83 | PASS | Commit `5dff16956`. `docs/packets/audits/gms_v83/` + `gms_v83.json` populated. No fixes (v95-era gates confirmed correct for v83). |
| Phase 3 — v87 | PASS | Commit `6bd324e14`. `docs/packets/audits/gms_v87/` + new `gms_v87.json`. No fixes (clean mirror). |
| Phase 3 — JMS v185 | PASS + real fix | Commits `8c276e0c6` (ChannelConnect fix) + `afce6fa47` (cross-version pass). `docs/packets/audits/jms_v185/` + new `gms_jms_185.json`. |
| Phase 4 — TOTAL.md + closeout | PASS | Commit `7edcff74d`. `TOTAL.md` accurate (coverage complete, 6 ❌ all explained in §4, quest=4-with-deferrals matches reality). `post-phase-b.md` accurate. |

### Build / test results (worktree root)

| Command | Result |
|---|---|
| `go build ./libs/atlas-packet/... ./tools/packet-audit/...` | PASS (exit 0) |
| `go test -race ./libs/atlas-packet/...` | PASS (no FAIL/panic) |
| `go test -race ./tools/packet-audit/...` | PASS (all packages ok) |

### Verdict-count reconciliation

- v95 SUMMARY: 56 rows = 28 login + 28 misc. Verdicts: 50 ✅ / 6 ❌ / 0 ⚠️ — matches `post-phase-b.md`.
- 28 misc rows by domain: merchant 7, socket 5, fame 4, quest 4, ui 3, account 3 (incl. pre-existing AcceptTos), channel 2, stat 1 = 28. ✓
- The 6 ❌ = CharacterList (login baseline, untouched) + 5 misc static-analyzer artifacts (Changed/Hello/ChannelConnect width-label, GiveResponse int16-pair, ChannelChange locateAtlasFile collision) — all documented in TOTAL.md §4 as non-bugs.

### Discrepancy vs review brief (informational, not a gap)

The review brief enumerated misc rows as `quest(5)` and `channel(2)`. The actual SUMMARY has **quest = 4 audited rows** (Action, ScriptProgress, ActionScriptStart, ActionScriptEnd) with 3 quest packets correctly deferred to `_pending.md` (ActionStart/ActionComplete/ActionRestoreLostItem — each needs a coupled atlas-channel handler change, out of libs-only audit scope). channel = 2 packets (ChannelChange + ChannelChangeRequest), but the ChannelChange SUMMARY row displays the collided `buddy/clientbound/channel_change.go` path due to the documented `locateAtlasFile` tool limitation. Both are consistent with TOTAL.md and `post-phase-b.md` and reflect documented deferrals/tool-limits, not silently dropped work.

### No silently dropped tasks

Every plan phase has commit evidence or a documented skip/deferral. No task outside the four documented corrections was dropped.

### Overall assessment

**Plan adherence: FULL** (accounting for the four documented corrections). **Recommendation: READY_TO_MERGE** — all phases implemented or correctly deferred with documentation, both real wire bugs (stat HP/MP widening, JMS ChannelConnect gm field) fixed with IDA-cited gates + 4-variant tests, builds and race tests clean. The pre-PR main-integration union step noted in `post-phase-b.md` §Integration remains the only outstanding action before the PR (forked before sibling tasks merged).
