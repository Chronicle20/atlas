# Plan Audit (Final) — task-123-megaphones-maple-tv

**Plan Path:** docs/tasks/task-123-megaphones-maple-tv/plan.md (21 tasks + Amendment A1)
**Audit Date:** 2026-07-18
**Branch:** task-123-megaphones-maple-tv (HEAD 4485129a8)
**Base Branch:** main (c9490b724)
**Prior review checkpoint:** e6346df6d (21/21 DONE, audited MOSTLY_COMPLETE) — this audit re-confirms that checkpoint and covers the 14 subsequent commits (legacy expansion + Cheap/Heart/Skull fix).

## Executive Summary

All 21 original plan tasks plus Amendment A1 (DOM-25 config-resolution) remain correctly implemented at HEAD; no wire-literal regressions were introduced by the 14 post-checkpoint commits. The scope-expansion work (legacy v48–v79 megaphone support, Cheap/Heart/Skull routing, and two real bug fixes — Skull-mis-routed-to-TV and legacy update-time omission) is coherent, matches the atlas-data-derived availability spec, and is conservative on item-loss risk: every legacy tier the gate opens has an IDA-verified serverbound byte fixture, and Maple TV stays blocked on legacy specifically because no legacy template carries the TV clientbound writers. All seven affected modules build, vet, and test clean; all four packet-audit gates exit 0; redis-key-guard and goroutine-guard are clean. No Critical or Important findings.

## Part 1 — Original 21 Tasks + Amendment A1 (re-confirmation)

Re-verified directly against source (not merely re-trusting the prior audit):

| Check | Status | Evidence |
|---|---|---|
| No client wire code as Go literal outside `libs/atlas-packet` | PASS | `grep` for `NewAvatarMegaphoneResult(8[0-9]` / `NewTvSendMessageResultError([0-9]` outside test files: zero hits. Only non-codec numeric literals found in `character_cash_item_use.go` are `MajorVersion() >= 83` / `< 83` version-branch guards, not wire bytes. |
| Reject sites use `*Body(reason)` funcs | PASS | `character_cash_item_use_megaphone.go:277` — `session.Announce(...)(tvpkt.TvSendMessageResultWriter)(tvpkg.TvSendMessageResultErrorBody(tvpkg.TvResultQueueTooLong))`; `character_cash_item_use.go:371` (via the megaphone handler file) — `chatpkg.AvatarMegaphoneResultBody(chatpkg.AvatarMegaphoneWaitingLine)`. Both resolve `errorCodes` per tenant (`libs/atlas-packet/chat/avatar_megaphone_body.go:31`, `libs/atlas-packet/tv/tv_body.go:72-76`). |
| `TvMessageType` is a semantic string end-to-end | PASS | `string` (not byte) in `libs/atlas-saga/payloads.go:1001`, `services/atlas-world/.../broadcast/model.go:26`, `.../kafka/message/broadcast/kafka.go:31,55,77`, `services/atlas-saga-orchestrator/.../kafka/message/broadcast/kafka.go:34`, `services/atlas-channel/.../kafka/message/worldbroadcast/kafka.go:40`. Consumed back into the semantic `tv.TvMessageType` type only at the packet boundary (`kafka/consumer/worldbroadcast/consumer.go:114`). |
| A1.3 (branch on reason, never resolved byte) | PASS | `AvatarMegaphoneResultBody`/`TvSetMessageBody`/`TvSendMessageResultErrorBody` all resolve via `atlas_packet.WithResolvedCode`/`ResolveCode` and never compare the resolved byte to a literal. |

**Build/vet/test on all 7 changed modules:** clean (see Part 3). This matches and reconfirms the e6346df6d checkpoint's "21/21 DONE" finding; nothing regressed in the 14 subsequent commits.

## Part 2 — Scope Expansion (legacy v48–79, Cheap/Heart/Skull, bug fixes)

### 2.1 Legacy per-(version, tier) gate — `character_cash_item_use.go:261-294`

```go
if t.MajorVersion() < 83 {
    if category == item.ClassificationAvatarMegaphone { return /* blocked, no consume */ }
    tier := (uint32(itemId) / 1000) % 10
    allowed := tier <= 4                              // basic/super/cheap/heart/skull, all versions
    if t.Region() == "GMS" {
        switch t.MajorVersion() {
        case 72, 79: allowed = allowed || tier == 6 || tier == 7   // item + triple
        case 61:     allowed = allowed || tier == 6                // item only
        }
    }
    if !allowed { return /* blocked, no consume */ }
}
```

Cross-checked against `.superpowers/sdd/megaphone-item-availability.md` (atlas-data-sourced): item megaphone (5076000) = v61+, triple (5077000) = v72+, everything else v48+. The gate matches exactly — v61 gets tier 6 only (triple item doesn't exist on v61), v72/v79 get both. Tier 5 (Maple TV) is **never** added to any legacy `allowed` expression — confirmed blocked on every legacy version regardless of tier availability, matching the stated reason (no legacy template has TV clientbound writer entries — verified: `grep '"writer": "Tv'` templates for gms_48/61/72/79 return no rows below).

**Serverbound wire verification backing the opened tiers** (byte fixtures, not just gate logic):
- `libs/atlas-packet/cash/serverbound/v61_test.go`: `CashItemUseMegaphone`/`CashItemUseSuperMegaphone`/`CashItemUseItemMegaphone`/`CashItemUseMapleTV` all carry `packet-audit:verify ... version=gms_v61 ida=0x...` markers with real IDA addresses and full decompile citations.
- `v72_test.go` / `v79_test.go`: same pattern, additionally covering `CashItemUseTripleMegaphone`.
- Coverage matrix (`docs/packets/audits/status.json`) confirms: `CashItemUseItemMegaphone` state=`verified` for gms_v61/v72/v79; `CashItemUseTripleMegaphone` state=`verified` for gms_v72/v79. These are exactly the (version, tier) pairs the gate opens — **no legacy tier is opened without a matrix-verified wire**.
- Maple TV: serverbound wire IS IDA-verified for v61/72/79 (byte fixtures exist in the same files), but the gate correctly does **not** open it — the code comment (`character_cash_item_use.go:226-243`) documents this is because the *clientbound* ack writers are absent from the legacy templates, which would be a real item-loss-equivalent bug (consume succeeds, every response packet fails to resolve an opcode). This is the correct, conservative call.

**Verdict: no item-loss risk found.** Every opened legacy tier has a verified serverbound wire; the one verified-but-blocked tier (TV) is blocked for a documented, correct reason.

### 2.2 Skull routes to super/world on every version (case 4)

`character_cash_item_use_megaphone.go:143-182`: case 4 (Skull, 5074000) unconditionally builds `NewItemUseSuperMegaphone` and emits `Tier: tierSuper, Scope: scopeWorld` — no version branch, no TV routing. The extensive inline comment documents the v83/v95/jms IDA findings that justify this (Skull's client wire is byte-identical to Super Megaphone on every version that sends it; the previous `>=95 → handleMapleTVUse` routing was a real decode-mismatch bug, now removed). Confirmed: `handleMapleTVUse` is only reachable from case 5 (`5075xxx`), never case 4.

### 2.3 Legacy megaphone updateTime is trailing, not absent

`libs/atlas-packet/cash/serverbound/item_use_megaphone.go`: no `megaphoneHasUpdateTime` gate remains (`grep -rn "megaphoneHasUpdateTime"` matches only a corrective code comment and a test-file historical reference, zero live logic). `Encode`/`Decode` use the plain `if !m.updateTimeFirst { ... }` — identical treatment for v48/61/72/79 and v83/84 (all `updateTimeFirst=false`). The file's doc comment explains the correction: the earlier gate mistook a shared-tail fallthrough for an absent field; IDA re-trace on all four legacy builds' shared rate-check tail confirmed the trailing `uint32` is always written. `item_use_megaphone_test.go` and the per-version `v61/72/79_test.go` fixtures assert the trailing bytes are present.

### 2.4 Cheap/Heart/Skull handler cases (tiers 0/3/4)

Present in `character_cash_item_use_megaphone.go` cases 0/3/4, each reusing the basic (channel-scope) or super (world-scope) codec/broadcast path with an extensive inline citation of the IDA recon that justifies "no distinct wire, dead-but-harmless on GMS v83–95 clients, real send on jms" — matches the adjudicated "known, not a gap" item in the task brief. No fabricated wire shape: the comments are explicit that GMS<95 clients never emit these ids at all (no encode call exists client-side), so decoding is a no-op safety net, not a guess presented as verified.

## Part 3 — Build / Test / Gate Verification

All commands run from the worktree root, each module in isolation:

| Module | `go build ./...` | `go vet ./...` | `go test ./... -count=1` |
|---|---|---|---|
| libs/atlas-packet | PASS | PASS | PASS (all packages `ok`, incl. `cash/serverbound`, `chat`, `chat/clientbound`, `tv`, `tv/clientbound`) |
| libs/atlas-saga | PASS | PASS | PASS |
| libs/atlas-redis | PASS | PASS | PASS |
| services/atlas-channel/atlas.com/channel | PASS | PASS | PASS (80 `ok` packages, 0 FAIL) |
| services/atlas-world/atlas.com/world | PASS | PASS | PASS (7 `ok` packages, 0 FAIL) |
| services/atlas-saga-orchestrator/atlas.com/saga-orchestrator | PASS | PASS | PASS (30 `ok` packages, 0 FAIL) |
| services/atlas-configurations/atlas.com/configurations | PASS | PASS | PASS (9 `ok` packages, 0 FAIL) |

**Packet-audit gates** (each run as its own command per instructions):

| Gate | Result |
|---|---|
| `go run ./tools/packet-audit matrix --check` | exit 0 |
| `go run ./tools/packet-audit dispatcher-lint` | `dispatcher-lint: clean`, exit 0 |
| `go run ./tools/packet-audit fname-doc --check` | `fname-doc check OK (234 structs without an audit report carry no fname)`, exit 0 |
| `go run ./tools/packet-audit operations --check` | `operations check OK (0 absent-writer note(s))`, exit 0 |

**Supplementary (CLAUDE.md-mandated, not explicitly requested but run for completeness):** `tools/redis-key-guard.sh` exit 0, `tools/goroutine-guard.sh` exit 0 (both informational-only module scans, no violations). `go run ./tools/packet-audit matrix` (no `--check`, full regenerate) produced **zero diff** against the committed `status.json`/`STATUS.md` — the matrix is not stale.

## Minor Observation (non-blocking, pre-existing, not introduced by this branch)

While cross-checking the matrix I found `CashItemUseItemMegaphone` / `CashItemUseTripleMegaphone` / `CashItemUseMapleTV` show `state: incomplete, note: "no audit report"` for gms_v87/gms_v95/jms_v185 in `status.json`, **despite** real byte-fixture tests, evidence-ledger YAMLs, and audit-report JSON/MD files existing on disk for those exact cells (e.g. `docs/packets/evidence/gms_v87/cash.serverbound.CashItemUseItemMegaphone.yaml`, `docs/packets/audits/gms_v87/CashItemUseItemMegaphone.md`). Root cause traced to `tools/packet-audit/internal/matrix/build.go`: the USE_CASH_ITEM dispatcher's registry `fname:` for gms_v87/v95/jms_v185 (`docs/packets/registry/gms_v87.yaml:2331` etc.) is literally `CItemSpeakerDlg::_SendConsumeCashItemUseRequest` — the same IDAName the sub-body reports use — so the op-row "claims" these writers and excludes them from sub-struct promotion. `git diff c9490b724..HEAD -- docs/packets/registry/gms_v87.yaml` shows **no change** to this fname field — the collision is pre-existing on `main`, not introduced by task-123. It does not affect item-loss risk (the legacy-gate-relevant cells, v61/72/79 item/triple, are unaffected and correctly show `verified`), and `matrix --check` exits 0 because this is not one of the tool's checked invariants. Flagging for awareness only; not a task-123 defect and not blocking.

## Overall Assessment

- **Plan Adherence:** FULL (21/21 original tasks + A1, re-confirmed clean at HEAD)
- **Scope-Expansion Coherence:** FULL (legacy gate matches atlas-data availability exactly; both bug fixes — Skull routing, legacy updateTime — are IDA-verified corrections, not regressions)
- **Item-Loss Risk:** NONE FOUND — every opened legacy tier has a matrix-verified serverbound wire; the one verified-but-risky tier (legacy Maple TV) is correctly left gated off.
- **DOM-25 Compliance:** CLEAN — no surviving wire literal found outside `libs/atlas-packet` codec internals.
- **Recommendation:** READY_TO_MERGE

## Action Items

None required before merge. Optional follow-up (not blocking): investigate/fix the pre-existing registry-fname collision noted above so the coverage matrix correctly reports v87/v95/jms verification state for the three affected sub-body packets — this is a documentation-fidelity issue in the tooling, not a code or safety defect.
