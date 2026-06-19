# Plan Audit — task-099-keydown-skill-prepare-broadcast

**Plan Path:** docs/tasks/task-099-keydown-skill-prepare-broadcast/plan.md
**Audit Date:** 2026-06-15
**Branch:** task-099-keydown-skill-prepare-broadcast
**Base Branch:** main

## Executive Summary

All 7 plan tasks were faithfully implemented (7/7, 100%). The implementation correctly applies the in-flight design discoveries D10 (serverbound cancel folded into the existing `CharacterBuffCancel` handler — no standalone cancel handler/config row) and the 6-A struct rename (`SkillPrepareForeign`/`SkillCancelForeign` for matrix linkage), and intentionally removed `SkillCancelInfo` with no dangling references. Every Go gate passes (`go test -race`, `go vet`, `go build` clean in `libs/atlas-packet` and `services/atlas-channel`; `tools/packet-audit` tests pass; `matrix --check` exits 0 with no conflicts). All scope guards hold: ida-exports are insertions-only, the attack writer is untouched (D9), and no v92 files were added (D7 deferral). v92 is the only deferred work and is intentional, not a gap.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Pin per-version wire spec from IDBs (4 ops × 5 versions, read orders) | DONE | wire-spec.md:33-48 opcode summary (v84 clientbound 0xC2/0xC3 IDA-verified, line 43-46); per-op records with fname+address L55-118; OQ-5 MovingShoot ruled out-of-scope with per-version dispatch evidence L162-192; no UNRESOLVED cells L196-203 |
| 2 | Serverbound `SkillPrepareInfo` codec (version-conditional swallowMobId) + byte fixtures; `SkillCancelInfo` removed | DONE | skill_prepare_info.go:30-108 (struct + version-conditional `hasSwallowField` L21-23, gated on skillId==33101005 L88,104); tests skill_prepare_info_test.go:36-65 (round-trip + swallow round-trip + byte fixture covering v83 no-tail vs v95/jms tail). `grep SkillCancelInfo` → 0 hits (D10: cancel is CANCEL_BUFF) |
| 3 | Clientbound foreign codecs + body constructors + writer consts; `Operation()` returns foreign const | DONE | skill_prepare_foreign.go:12 const, :43 `Operation()` returns foreign writer; skill_cancel_foreign.go:12,:34 same; body ctors character/skill_prepare.go:15,31; byte-fixture markers in both `*_foreign_test.go` |
| 4 | Prepare handler (gate `shouldBroadcastKeydown`, no Destroy); keydown CANCEL folded into buff-cancel; effects helpers; main.go registers prepare handler + 2 writers | DONE | character_skill_prepare.go:26-33 gate, :38-57 handler (debug+return on miss, no Destroy — D3); character_buff_cancel.go:22 unconditional `buff.Cancel`, :25-32 keydown gate + `AnnounceForeignSkillCancel` (D10); effects.go:43,55 helpers; main.go:684-685 writers, :791 prepare handler (no cancel handler) |
| 5 | 5 seed templates wired (1 prepare handler + 2 writer rows each, per-version opcodes; v95 validator fix) | DONE | All 5 templates: prepare-handler op matches wire-spec (v83 0x5D, v84 0x5D, v87 0x60, v95 0x69, jms 0x58) w/ LoggedInValidator; writers match (v84 0xC2/0xC3, etc.); handler-count=1 each (no collision); v95 CharacterBuffCancel validator was MISSING-ON-MAIN, now LoggedInValidator (commit 50b7e2b3b) |
| 6 | Coverage-matrix promotion: fname cases, serverbound wrapper, exports harvested, reports+evidence+markers, STATUS.md 3 rows ✅ ×5 | DONE | run.go:321-336 candidatesFromFName cases for OnSkillPrepare/OnSkillCancel/DoActiveSkill_Prepare; serverbound wrapper character/serverbound/skill_prepare.go; 30 audit + 15 evidence + 5 export files in diff; STATUS.md:254,255,585 (clientbound prepare/cancel + serverbound prepare) all ✅ ×5; serverbound cancel rides CANCEL_BUFF row L584 ✅ ×5 |
| 7 | Verification sweep + followups.md operational/parked items | DONE | followups.md documents live config patch (incl. v95 validator note), in-map manual validation, D6 death/stun residual, v92 parked (D7), audit-verdict advisory note. All gates re-run below |

**Completion Rate:** 7/7 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None skipped. One intentional deferral:

- **v92 prepare/cancel — DEFERRED (design D7).** v92 has no client IDB; opcodes/read-orders cannot be IDB-verified and were deliberately not wired to avoid porting wire-format assumptions. Documented in followups.md §4. Confirmed no v92 template/file appears in `git diff main...HEAD --name-only`. This is per-design, not a gap.

## Build & Test Results

| Service / Module | Build | Tests | Notes |
|---|---|---|---|
| libs/atlas-packet | PASS | PASS | `go test -race ./...` ok; `go vet` clean; `go build` clean |
| services/atlas-channel/atlas.com/channel | PASS | PASS | `go test -race ./...` exit 0; `go vet` clean; `go build` clean |
| tools/packet-audit | PASS | PASS | `go test ./...` ok all packages |
| packet-audit `matrix --check` | n/a | PASS | exit 0, no conflicts |

Additional gate results:
- **ida-exports insertions-only:** `git diff main...HEAD --numstat -- docs/packets/ida-exports/` → all `N 0` (gms_v83 55/0, gms_v84 57/0, gms_v87 57/0, gms_v95 62/0, gms_jms_185 62/0). PASS.
- **Attack writer scope guard (D9):** `git diff` of `socket/writer/character_attack_common.go` is EMPTY. PASS.
- **v92 guard:** no v92 file in diff. PASS.
- **5 seed templates valid JSON:** all `jq -e .` OK. PASS.
- **redis-key-guard:** FAILs pre-existing on main (local-env `./... matched no packages`); our diff is redis-free — NOT a regression (matches task brief note).

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (subject to the operational follow-ups in followups.md — live tenant config patch + in-map manual validation, which are deploy-time, not code)

## Action Items

None blocking. For deploy/runtime (already tracked in followups.md, not code):

1. Patch live channel config for each existing supported-version tenant (handlers/writers don't hot-reload) and restart the channel — including the v95 `CharacterBuffCancel` validator that was missing on those live configs.
2. In-map manual validation (observer sees Hurricane / Monster Magnet / BigBang / Rapid Fire aura start on keydown, stop on key release; arrows still render).
3. Validate the death/stun-while-keydown path (D6 residual); add server-side cancel synthesis only if a stuck aura is observed.
4. v92: re-run wire-spec pin + Tasks 5/6 once a v92 IDB exists.

---

# Backend Guidelines Review (DOM-*/SUB-*/SEC-*)

- **Reviewer:** backend-guidelines-reviewer
- **Date:** 2026-06-15
- **Commit:** 62aadee9bfd742f11941f83f61b40b2b6ccd206c
- **Scope:** Go diff `main...HEAD` — `libs/atlas-packet/**` (skill prepare/cancel codecs), `services/atlas-channel/atlas.com/channel/socket/handler/**` (prepare handler, buff-cancel keydown demux, foreign announce helpers), `main.go` registration, `tools/packet-audit/cmd/run.go`.
- **Mindset:** FAIL until file:line evidence proves PASS.

## Objective Gate (Phase 1)

| Gate | Module | Result | Evidence |
|------|--------|--------|----------|
| `go build ./...` | `libs/atlas-packet` | PASS | clean, no output |
| `go vet ./...` | `libs/atlas-packet` | PASS | clean, no output |
| `go test ./... -count=1` | `libs/atlas-packet` | PASS | all packages `ok` |
| `go build ./...` | `atlas-channel` | PASS | clean, no output |
| `go vet ./...` | `atlas-channel` | PASS | clean, no output |
| `go test ./socket/handler/... -count=1` | `atlas-channel` | PASS | `ok atlas-channel/socket/handler 0.007s` — no producer hang |
| `go build`/`go vet ./...` | `tools/packet-audit` | PASS | clean |

## Classification

This change is a **packet-codec + socket-handler** change, NOT a REST domain package. None of the changed packages contain `model.go` (immutable-domain), `resource.go` (REST), `provider.go`, `administrator.go`, or `entity.go`. The DOM checklist items that target REST/DDD domains (DOM-01..05, DOM-08..19, DOM-22, DOM-23, EXT-*, SCAFFOLD-*) are **Not Applicable** by construction. Only the checks below have triggers in this diff.

## Applicable Checklist Results

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Logger param is `logrus.FieldLogger`, not `*logrus.Logger` | PASS | `character_skill_prepare.go:38`, `character_buff_cancel.go:17`, `effects.go:19/31/43/55` all take `logrus.FieldLogger`; codec `Encode/Decode` take `logrus.FieldLogger` (`model/skill_prepare_info.go:80/97`, `clientbound/skill_prepare_foreign.go:50/62`). Zero `*logrus.Logger`. |
| DOM-07 | Handlers don't use `logrus.StandardLogger()` | PASS | Handlers receive `l` from registration; no `StandardLogger()` in any changed handler. Registration in `main.go:791` passes the function ref (logger injected by the dispatcher, same as every sibling). |
| DOM-20 | Table-driven tests | PASS | `character_skill_prepare_test.go:42-104` (`cases := []struct{...}` + `t.Run`); `model/skill_prepare_info_test.go:94-167`; `clientbound/skill_prepare_foreign_test.go`, `skill_cancel_foreign_test.go`, `serverbound/skill_prepare_test.go` all use variant tables + `t.Run`. |
| DOM-21 | Reuse atlas-constants, no reinvented types/consts | PASS (with one justified local const) | Keydown classification reuses `skill.IsKeyDownSkill` / `skill.Id` (`character_skill_prepare.go:11,29`; `character_buff_cancel.go:11,25`), NOT a private list — verified `IsKeyDownSkill` exists at `libs/atlas-constants/skill/model.go:58`. The one new numeric literal `swallowMobPrepareSkillId = 33101005` (`model/skill_prepare_info.go:16`) has **no** atlas-constants equivalent (`grep 33101005 libs/atlas-constants/` → no match). See judgment note below — kept local, accepted. |
| DOM-24 | Kafka producer stubbed in emitting tests | N/A (no emit path) | The new handlers relay via `session.Announce` (socket writer, `session/processor.go:170`) through `_map.ForOtherSessionsInMap` — NOT `message.Emit`/`AndEmit`/`producer.Produce` (grep of all three changed handler files: zero matches). Handler test exercises only the pure `shouldBroadcastKeydown` gate, never the handler func, so no emit path runs. The `0.007s` handler-package runtime confirms no 42s producer-retry hang. No `producertest`/`TestMain` is required because no test crosses an emit boundary. |

## Convention Consistency (focus areas)

| Area | Status | Evidence |
|------|--------|----------|
| Immutable model + builder-style setters | PASS | `SkillPrepareInfo` has private fields + getters + chaining setters (`model/skill_prepare_info.go:30-75`). Clientbound `SkillPrepareForeign`/`SkillCancelForeign` are value types with private fields + getters + `New...` constructors (`clientbound/skill_prepare_foreign.go:20-42`, `skill_cancel_foreign.go:20-34`) — matches sibling clientbound codecs. |
| Version-conditional codec via `tenant.MustFromContext` | PASS | `model/skill_prepare_info.go:81,98` reads tenant from ctx; `hasSwallowField` gates on `Region()=="GMS" && MajorVersion()>=95 \|\| Region()=="JMS"` (`:21-23`). Encode/Decode are mirror-symmetric (`:88-90` vs `:104-106`) — round-trip + byte-fixture tests pin this across all 5 variants. |
| Curried handler / announce signatures | PASS | `AnnounceForeignSkillPrepare/Cancel` follow the project `f(l)(ctx)(wp)(args) Operator[session.Model]` curry (`effects.go:43-63`), identical shape to the pre-existing `AnnounceForeignSkillUse` (`:31-39`). Handler signature `func(l, ctx, wp) func(s, r, opts)` matches `CharacterUseSkillHandleFunc` (`character_skill_use.go:37`). |
| Foreign-codec `Operation()` returns the foreign writer const | PASS | `clientbound/skill_prepare_foreign.go:43` returns `CharacterSkillPrepareForeignWriter`; `skill_cancel_foreign.go:34` returns `CharacterSkillCancelForeignWriter`. Regression-pinned by `TestSkillPrepareForeignOperation`/`TestSkillCancelForeignOperation` (guards the known "foreign struct returns non-foreign const" bug). |
| Writer/handler registration | PASS | `main.go:684-685` registers both foreign writers in `produceWriters()`; `main.go:791` registers `CharacterSkillPrepareHandle`. Keydown-cancel reuses the already-registered `CharacterBuffCancelHandle` (no new handler needed — extension of existing demux). |
| Wrapper-delegates-to-shared-model pattern | PASS | `serverbound.SkillPrepare` embeds `model.SkillPrepareInfo` and delegates Encode/Decode (`serverbound/skill_prepare.go:37-43`), explicitly modeled on the AttackInfo wrapper convention (documented `:20-25`). |

## Error Handling / Logging (D3)

| Item | Status | Evidence |
|------|--------|----------|
| Benign drops logged at Debug, no session Destroy on cosmetic packet | PASS (deliberate, documented divergence from sibling) | `character_skill_prepare.go:46,51` log at `Debugf` and `return` on character-not-found and not-owned/not-keydown; `character_buff_cancel.go:28` silently skips broadcast on miss. This **intentionally diverges** from the sibling `CharacterUseSkillHandleFunc`, which calls `session.Destroy(s)` on an unowned skill (`character_skill_use.go:68`). The divergence is justified: the prepare/cancel packets are a cosmetic foreign relay that applies no skill effect, so a mismatched/spoofed skillId costs nothing and the drop is correctly non-fatal (D3). Documented at `character_skill_prepare.go:36-37`. Accepted. |
| Buff cancel still applies the real cancel before the relay gate | PASS | `character_buff_cancel.go:22` calls `buff.Cancel(...)` unconditionally; the keydown broadcast at `:25-32` is additive and gated, so the new code cannot regress the existing buff-cancel behavior. |

## Security Review (SEC-*)

atlas-channel is the client-facing socket service, so the client-trust questions are in scope even though it is not an auth service.

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| SEC — skill ownership validated before broadcast | PASS | `shouldBroadcastKeydown` (`character_skill_prepare.go:26-33`) requires the skill be present in the **server-loaded** skill book at `Level() > 0` (`cp.GetById(cp.SkillModelDecorator)` at `:44`), not a client-supplied level. A spoofed/unowned skillId is dropped. The cancel path applies the same gate (`character_buff_cancel.go:27-28`). |
| SEC — relayed client-supplied `action`/`actionSpeed`/`level` | PASS (accepted for cosmetic relay) | `action`/`actionSpeed` and the client-supplied `level` from the prepare packet are relayed verbatim into the foreign packet (`character_skill_prepare.go:55` → `effects.go:47` → `CharacterSkillPrepareForeignBody`, `skill_prepare.go:15-25`). These drive only the remote-render animation of an aura the caster is provably entitled to cast (ownership gate above). They apply **no** damage, buff, stat, or authoritative state — the level is purely a render hint for the foreign client. No server-side trust decision keys off these fields. Acceptable for a cosmetic foreign packet; the load-bearing authority (does this character own a leveled keydown skill?) is server-resolved. |
| SEC — swallowMobId trust | PASS (out-of-scope skill, decode-only) | `swallowMobId` is decoded only for skillId `33101005` on GMS v95+/JMS (`model/skill_prepare_info.go:104-106`). That skill is out of the keydown set, so `shouldBroadcastKeydown` returns false and the field is never relayed by this change. It exists solely so the serverbound reader consumes the correct byte count (wire-format correctness), preventing reader desync on those versions. |
| SEC-04 — no hardcoded secrets | PASS | No secrets/keys/passwords in any changed file. |

## Judgment Call — `swallowMobPrepareSkillId` (33101005) locality (requested)

**Verdict: keeping it local to the packet codec is correct; do NOT move it to atlas-constants.**

Rationale:
- It is a **wire-format decode conditional**, not a domain classification. Its only role is "does the serverbound prepare body carry a trailing u32 on this version?" (`model/skill_prepare_info.go:88,104`). atlas-constants holds gameplay classifications (item ids, inventory/weapon types, job/skill/world ids) consumed by business logic; a byte-layout guard for a single out-of-scope skill is not that.
- The skill is **out of scope** for this feature (not a keydown skill; never reaches the broadcast path). Promoting it to a shared `skill.SwallowId` constant would imply a gameplay relationship the task explicitly says is **unverified** ("Job attribution is not verified" — `:13-15`). Exporting an under-verified id into the shared lib invites misuse.
- It is co-located with the exact logic that needs it and documented with the IDA guard (`nSkillID == 33101005` in `DoActiveSkill_Prepare`). If/when the swallow skill is implemented as a gameplay feature, that is the right moment to lift the id into atlas-constants with verified attribution.

This is the rare DOM-21 case where a numeric literal stays local; the comment at `:12-16` correctly scopes it to wire behavior, not identity.

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- None.

### Notes / accepted judgments
- D3 divergence from `character_skill_use.go` (Debug-drop vs `session.Destroy`) is intentional and correct for a cosmetic relay packet.
- Client-supplied `action`/`actionSpeed`/`level` relayed verbatim is acceptable because the ownership/keydown gate is server-authoritative and the fields drive only foreign render.
- `swallowMobPrepareSkillId` correctly kept local to the codec (wire conditional, unverified attribution, out-of-scope skill).

## Overall: **PASS**

Build, vet, and tests are clean in both affected modules. Every applicable checklist item passes with file:line evidence. No blocking or non-blocking findings.
