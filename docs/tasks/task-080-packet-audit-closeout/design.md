# Packet-Audit Closeout — Four-Version Baseline — Design

Status: Draft for review
Created: 2026-06-04
PRD: `prd.md` (approved)
Worktree: `.worktrees/task-080-packet-audit-closeout`

---

## 1. Purpose & Framing

The PRD is an itemized work-list (buckets B1–B6 + analyzer §4.7 + docs §4.8), each
item already carrying its IDA function@address and required wire change. This design
does **not** re-derive those per-packet shapes — the PRD is the authority for *what each
byte should be*. Instead this design fixes the things the PRD left open:

1. **Resolves the four open questions** (§9 of the PRD) against actual source, so execution
   doesn't re-discover them.
2. **Defines the cross-cutting mechanics** every wire fix shares — the version-gate idiom,
   the region-dispatched body pattern, the byte-test harness, IDA-evidence capture — so each
   bucket item is a fill-in-the-shape exercise, not a fresh architecture decision.
3. **Picks an analyzer-enhancement strategy** with an explicit fix-in-tool-vs-register boundary.
4. **Sequences the work** so shared infrastructure and high-risk hot-path fixes land before the
   long verification tail and the documentation roll-up.

The output is a *trusted baseline*: `_pending.md` holds only blessed permanent exclusions, a
fresh `packet-audit` run is clean of the named false-positive classes, every real wire bug is
fixed with byte-level tests and IDA-verified gates, and a "start a new version pass" guide exists.

---

## 2. Open Questions — Resolved

### Q1 — AffectedArea / mist plumbing (B1.1): extend the existing event, don't invent

**Finding (verified in source):**

- The in-memory model `services/atlas-maps/.../mist/model.go` **already carries**
  `sourceSkillId`, `sourceSkillLevel`, the origin (`originX/originY`), the relative RECT
  offsets (`ltX/ltY/rbX/rbY`), `ownerId`, `disease`, and `duration`.
- The Kafka event payload `mist.CreatedBody`
  (`services/atlas-maps/.../kafka/message/mist/kafka.go`) **drops `sourceSkillId` and
  `sourceSkillLevel`** and has **no mist-type (`nType`) field**. It carries owner, origin,
  RECT offsets, duration only.
- The producer `mist/producer.go` `createdEventProvider()` is where the drop happens — it
  simply doesn't copy the skill fields off the model.
- The channel consumer `services/atlas-channel/.../kafka/consumer/mist/consumer.go`
  constructs the packet and **hardcodes `0` for `skillLevel`** (the 10th constructor arg).

**Decision:** Extend the *existing* `CreatedBody` event — do not add a second event. Required
plumbing, in dependency order:

1. **atlas-maps event contract** (`mist.CreatedBody`): add `SourceSkillId uint32`,
   `SourceSkillLevel uint32`, and `Type` (mist/affected-area type → `nType`). The first two are
   already on the model; `Type` is the only genuinely new field.
2. **atlas-maps producer** (`createdEventProvider`): copy `m.SourceSkillId()`,
   `m.SourceSkillLevel()`, and the type off the model into the event.
3. **mist `Type` source:** the model has `disease` (a debuff string) + `sourceSkillId` but no
   explicit affected-area type. During execution, the B1.1 spike reads (a) the atlas-maps mist
   *creation* path to see whether a type is derivable from the skill/disease, and (b) the four
   IDBs' `CAffectedAreaPool::OnAffectedAreaCreated` to confirm what `nType` selects on the
   client. If `nType` is a fixed value for the skill-driven mist case, the model/event carries a
   constant; if it varies, it is derived at creation time. **This is a confined spike, not a new
   open question** — the contract change (add `Type`) is the same either way.
4. **packet rewrite** (`field/clientbound/affected_area_created.go`): replace the bespoke
   `originX/originY + 4 offsets` shape with the client layout
   `dwId, nType, dwOwnerId, nSkillID(int32), nSLV(byte), phase(int16), rcArea(16-byte RECT),
   tEnd(int32)`, with leading `tStart(int32)` gated `GMS && MajorVersion>=95`. The 16-byte
   `rcArea` is the **absolute** RECT, computed packet-side as `origin + offset` for LT and RB
   (the model stores relative offsets; the client wants absolute). Drop the invented
   `originX/originY` from the wire.
5. **channel consumer:** pass the real `SourceSkillId`, `SourceSkillLevel`, and `Type` through
   instead of the hardcoded `0`.

This is in-memory event/model plumbing only — no schema migration (consistent with PRD §6).

### Q2 — `LoginAuth` orphan: export-then-decide (bucket 6 spike)

Not resolvable without IDA; the design fixes the **decision procedure** so execution is
mechanical. Export `LoginAuth` (and the addressed login FNames) via `packet-audit export`
against all four IDBs, then:

- **Absent in all four IDBs** → dead/legacy → **remove** the writer + its template entry,
  recorded in the accepted-exclusions registry as "removed, not present in any baseline."
- **Present only in JMS185** → **gate** `Region()=="JMS"`, audit its shape, assign a verdict.
- **Present in GMS** → it was mis-filed as orphan; audit and verdict normally.

### Q3 — Merchant mode 1: confirm-then-dispose (B2.3 spike)

Decision procedure: read `OnEntrustedShopCheckResult` (and the entrusted-shop check-result
dispatch) across all four IDBs.

- **Mode 1 absent in all four** → document as client/KMS-only in the registry; **do not
  implement**. (The PRD already notes "mode 1 absent in v95.")
- **Mode 1 present in any baseline** → implement it, gated to the versions that carry it, with a
  byte test.

Mode 8 (`Decode4 shopId + Decode1 channelId`, ErrorUnknown-channel notice) is implemented
regardless (PRD says present). Mode 11 (StringPool 3508) gets a constant + emitter when a
server-side path exercises it; if nothing emits it, it is registered as a defined-but-unused
constant, not left as a silent gap.

### Q4 — Analyzer depth: fix the tractable classes in-tool, register the opaque residue

**Finding (verified in `tools/packet-audit/`):** three of the four false-positive classes have
their machinery already present and need completion, not new subsystems; only sub-struct descent
is deep:

| Class | Locus | Existing machinery | Effort |
|---|---|---|---|
| Early-return / exclusive-branch | `internal/atlaspacket/analyzer.go` (guard stack ~237–405; `blockTerminatesWithReturn` ~335–360; suffix-taint ~395–405) | Guard stack + return-detection helper already exist | Low–med |
| Opaque-buffer / width equality | `internal/diff/diff.go` (`primWidth`/`idaWidth` ~80–114) | Width maps exist; need equivalence rules (`WriteByteArray(N)≡DecodeBuf(N)`, `WriteLong≡8-byte buf`, `WriteInt16+WriteShort(0)≡Decode4`, `point≡EncodeBuffer(8)`) | Low–med |
| Qualified struct-name tracking | `internal/.../run.go` `locateAtlasFile` (~1709–1744) | `pkg` disambiguation param **already implemented**; gap is the completeness of the `candidatesFromFName` map | Low |
| Sub-struct / loop descent | `analyzer.go` (~433–465, `KindRepeat`), `registry.go` (~109–180), `diff.go` (~128–164) | Loops emit `KindRepeat`; registry pre-analyzes types **that have an `Encode`/`Write` method**; **no fallback for types without one** (`model.Asset`, `GW_ItemSlotBase`) | Med–high |

**Decision — bounded "fix in tool first" with an explicit register boundary:**

- Fix all three low/low-med classes in the analyzer outright (early-return modeling, width
  equivalence, complete the `candidatesFromFName` map for the known collisions).
- For sub-struct descent: generalize descent for any field whose type **has an `Encode`/`Write`
  method or decomposes into known primitives** (covers the bulk of the PRD §4.7 list — party
  `WritePartyData`, npc `Action`/`ShopList`, character `CharacterInfo`, inventory bodies, etc.).
- The **boundary:** a type with no `Encode` method and no statically-decomposable layout
  (genuine opaque residue) is **not** chased with a decompiler — it goes to the
  accepted-exception registry (§4.8) with a one-line justification. The bar is "a clean re-run
  of the named classes," not "a perfect decompiler." This is the PRD's own framing in open Q4,
  made concrete: tool-generality where the type self-describes, registry where it doesn't.

A class is "done" when re-running `packet-audit` over the four versions emits **no** ❌/🔍 from
that class except entries that appear verbatim in the registry.

---

## 3. Cross-Cutting Mechanics (every wire fix follows these)

These are extracted once so each bucket item is a fill-in exercise. They match existing repo
idioms (verified in source) — this task introduces **no new** convention.

### 3.1 Version gate

```go
t := tenant.MustFromContext(ctx)
v95Plus := t.Region() == "GMS" && t.MajorVersion() >= 95   // computed local, reused
```

Accessors: `t.Region() string`, `t.MajorVersion() uint16`, `t.MinorVersion() uint16`. Gates are
plain Go `if` on these locals. Canonical exemplars already in tree:
`stat/clientbound/changed.go` (v95 HP/MP width), `model/avatar.go`, `model/movement.go`
(`t.Region() != "GMS" || t.MajorVersion() > 83`). Gate naming convention for new gates:
`Region()=="GMS" && MajorVersion()>=N`.

### 3.2 Region-dispatched body (the >2-version answer; respects the 2-nested-guard cap)

The "2-nested-guard hard cap" is **review-and-`awk` policy**: no encoder/decoder may exceed two
levels of nested `if` guards. A divergence that spans >2 versions (B1.5 EffectWeather GMS-vs-JMS,
B5.1 JMS cash bodies) is expressed by **dispatching to a per-region body function** at the top of
the encode/decode closure, *not* by stacking a third guard:

```go
func (m *X) Decode(l, ctx) func(r, opts) {
    t := tenant.MustFromContext(ctx)
    return func(r *request.Reader, opts map[string]interface{}) {
        if t.Region() == "JMS" {
            m.decodeJMS(r); return
        }
        m.decodeGMS(t, r)   // GMS body may itself carry ≤2 guards
    }
}
```

Each region body is independently testable and keeps every function under the nesting cap. The
repo nesting `awk` must stay clean after every change.

### 3.3 Byte-level wire-shape test (the oracle — no change ships on analyzer verdict alone)

Tests live beside the packet (`*_test.go`), use the model's own Builder (no `*_testhelpers.go`),
and assert exact bytes per targeted version. Tenant construction:

```go
ctx := pt.CreateContext("GMS", 83, 1)               // test/context.go helper
// variants: GMS v28/83/87/95, JMS v185  (test.Variants)
got := pkt.Encode(nil, ctx)(nil)
if !bytes.Equal(got[:6], []byte{0x50,0x00,0x7E,0x00,0x02,0x00}) { ... }
```

Every wire change gets a test asserting the byte slice for **each** version it targets (e.g.
AffectedArea: one assertion for v83/v87/JMS185 8-field shape, one for v95 with `tStart`). Where a
field is time-derived (durations), assert only the load-bearing prefix, as existing tests do.

### 3.4 IDA-evidence capture

Each resolved verification item and each fix records, in its audit entry / the registry, the IDA
`FName@address` and the read-order it was verified against (e.g.
`CCashShop::OnBuy@0x47eaa7 → Encode1 isMaplePoint, Encode4 dwOption, Encode4 nCommSN`). This is
the durable proof the byte test encodes.

---

## 4. Workstream Designs

### 4.1 Real wire bugs — B1 (field, chat, quest)

Each is a localized encoder/decoder rewrite + caller updates + byte tests, using §3 mechanics:

- **B1.1 AffectedArea** — per §2/Q1: event-contract extension (atlas-maps) → producer →
  packet rewrite (absolute RECT, `tStart` gated `GMS>=95`) → channel consumer. Spans two services
  + one lib; the only multi-service item. Tests: v83/v87/JMS185 (8 fields) and v95 (+`tStart`).
- **B1.2 chat `Multi`** — prepend `updateTime uint32` gated `GMS>83`; update all group-chat-send
  callers. Hot path — caller sweep is part of the change, not a follow-up.
- **B1.3 quest `ActionStart/Complete`** — insert `Encode4(nItemPos)` between `npcId` and the
  conditional `x,y`; packet + `quest_action.go` handler land together; verify the atlas
  `autoStart` gate against `!CQuestMan::IsAutoAlertQuest`.
- **B1.4 quest `ActionRestoreLostItem`** — redesign to count-prefixed id array
  (`Encode1(0)+Encode2(questId)+Encode4(count)+EncodeBuffer(4*count)`); replace the single
  `unk1+itemId` model with a slice.
- **B1.5 EffectWeather JMS** — region-dispatched body (§3.2): GMS keeps the leading byte; JMS
  reads `Decode4 itemId` first, optional `Decode4 extra` when
  `get_consume_cash_item_type(itemId)==51`, optional `DecodeStr message` when `itemId!=0`.

### 4.2 atlas-channel handler-logic — B2

- **B2.1 NPC continue-conversation discriminator** — the structs are already correct; the fix is
  in `npc_continue_conversation.go` handler routing: text reply is msgType **3**/**14**, not 2.
  Map 3/14→`ContinueConversationText`, 5/8/9→`ContinueConversationSelection`,
  0/1/2/13→no trailing body. Pure handler logic; covered by handler-level tests.
- **B2.2 hired-merchant serverbound** — implement/verify the decode + channel handler for the op
  family currently stubbed as a bare constant.
- **B2.3 merchant modes 1/8/11** — per §2/Q3.

### 4.3 Verification deferrals — B3 (resolve to a verdict; fix new bugs in-task)

B3.1–B3.6 are IDA spikes that each terminate in a verdict, not necessarily a code change. Pattern
per item: enumerate the client modes/op-bytes from the cited IDB function, diff against the Atlas
template/router config, and either (a) confirm ✅ with evidence, or (b) if a real divergence
surfaces, fix it in-task with a byte test and gate. B3.6 (social cross-version enum-drift) is the
broadest: confirm template-configured sub-op *value spaces* (mode/op numbers) match the client
across all four versions for buddy/chat/guild/party/note; the per-struct wire shapes are already
✅, so this is a config-value audit, not an encoder rewrite. Any fix is config-or-constant level.

### 4.4 v87 provisional gates — B4

B4.1: confirm `stat/Changed` HP/MP width gate and `ui/Lock` int32 gate against the **v87 IDB**
(currently confirmed v83 + JMS185 only). Outcome is "keep boundary" or "tighten boundary" + a v87
byte test added to the existing test matrix. No new pattern.

### 4.5 JMS cash-shop NX-payment — B5

**Finding (verified):** the channel→cashshop→wallet path is **fully region-agnostic and already
wired**: `CashShopOperationHandleFunc` → `RequestPurchase` → Kafka `REQUEST_PURCHASE` →
atlas-cashshop consumer → `PurchaseAndEmit` → wallet (Credit/Points/Prepaid). The cash serverbound
bodies already region-dispatch inline (`ShopOperationBuy` branches `GMS>=87` vs else). The **only**
gaps are (1) JMS-correct serverbound *bodies* for the 5 ops and (2) the JMS *template* has no
`CashShopOperationHandle` entry / op-byte map.

**Design:**

1. **Bodies** — add JMS-correct decode for buy/gift/couple/friendship/rebate using the
   **region-dispatched body** pattern (§3.2), *not* a 3rd nested guard. JMS shapes use the SPW
   string + serial-number forms per the cited IDA (`CCashShop::OnBuy@0x47eaa7`,
   `SendGiftsPacket@0x47bced`, `OnBuyCouple@0x48085a`, `OnBuyFriendship@0x481184`,
   `OnRebateLockerItem@0x47c059`). Byte test each against JMS185.
2. **Template** — add the `CashShopOperationHandle` handler entry to `template_jms_185_1.json`
   with the JMS op-byte→operation map, each op-byte justified by its cited `Encode1(...)`. Also
   remap the two template-only interaction ops (PersonalStore `BuyItem` 0x14/0x1F,
   `DeliverBlackList` 0x1B) whose bodies already match.
3. **Routing** — **no new code**: the existing handler already calls `RequestPurchase`, which the
   existing cashshop consumer turns into a wallet debit. Once the template routes JMS op-bytes to
   `CashShopOperationHandle` and the bodies decode correctly, JMS purchases settle through the
   existing wallet flow. The design explicitly *forbids* inventing a new command/topic.

**Scope guard:** if a JMS body needs a field the current `RequestPurchaseCommandBody`
(`Currency`, `SerialNumber`) can't carry, that is surfaced as a finding and the command body is
extended minimally — but the default expectation is that the existing contract suffices (buy is
currency+serial).

### 4.6 Login IDA-export backlog — B6

Export the addressed FNames (`OnViewAllCharResult`, `SendSelectCharPacketByVAC`,
`OnSelectCharacterByVACResult`, `OnDenyLicense`, `OnButtonClicked`, `LoginAuth`) and the
bare-handler set (AfterLogin, RegisterPin, the PIC family 0x15–0x1E, SetGender,
WorldCharacterListRequest, ServerStatus, PicResult) to the version json via `packet-audit export`,
then audit and assign verdicts. `LoginAuth` per §2/Q2. v87 login quirks:
`SendCheckPasswordPacket@0x62dfb4` v87 appends `Encode4(PartnerCode)` (zero functional impact →
read-and-discard or document, decided by the spike); `SendSelectCharPacket` 0x1D/0x1E v87 PIC
opcode layout differs → either v87-specific handler variants or opcode-keyed dispatch, whichever
the export shows is minimal. Bare handlers that map to a real client function get audited; those
with no client counterpart in any IDB are documented as intentional.

### 4.7 Analyzer enhancements

Per §2/Q4. Implemented in `tools/packet-audit/internal/{atlaspacket,diff}` and `…/run.go`.
Validation: a fresh four-version run diffed against the curated registry shows no spurious ❌/🔍
from the four named classes. The analyzer changes ship **before** the SUMMARY/TOTAL regeneration
(§4.8) so the regenerated artifacts reflect the enhanced tool.

### 4.8 Baseline documentation + ledger curation

End-state:

- **Four `SUMMARY.md`** regenerated post-fix + post-analyzer; no spurious ❌/🔍.
- **`_pending.md` (both copies)** reduced to zero actionable items; deferral content replaced by a
  curated **accepted permanent exclusions** registry — each residual entry carries IDA evidence +
  a one-line justification (genuinely-unanalyzable opaque buffers, removed-legacy FNames,
  client/KMS-only modes) — plus a pointer to this task as closeout of record.
- **`TOTAL.md`** — task statuses flipped to shipped, verdict roll-up recomputed, §3/§5 replaced
  with "baseline complete — zero open actionable deferrals."
- **New "start a new client-version pass" guide** — where IDBs go, how to run `packet-audit
  export`/audit, how SUMMARY/TOTAL/_pending relate, the gate-naming convention, and the
  region-dispatched body strategy for >2-version divergences.

---

## 5. Sequencing & Dependencies

Layered, not strictly bucket-by-bucket — shared infrastructure and the multi-service / hot-path
fixes go first; the verification tail and docs roll-up go last (docs depend on everything).

1. **Phase A — Analyzer + test harness foundation.** Land §4.7 analyzer enhancements first. This
   de-noises every subsequent re-run, so verification spikes (B3/B4/B6) read a clean signal and
   the eventual SUMMARY regeneration is meaningful. Independent of all wire fixes.
2. **Phase B — Real wire bugs (B1) + handler fixes (B2).** Highest correctness value (hot paths:
   chat, quest, NPC conversation). B1.1 first within this phase (it's the only multi-service
   change — atlas-maps event contract + atlas-channel + lib — so it sets the event plumbing
   early). Each item: fix + byte tests + gates, self-contained.
3. **Phase C — JMS cash-shop (B5).** Bodies + template + verify routing into the existing wallet.
   Independent of B1/B2; can overlap Phase B but kept distinct because it touches templates +
   atlas-cashshop verification.
4. **Phase D — Verification spikes (B3, B4, B6).** IDA-driven; each ends in a verdict, fixing
   in-task where a real bug surfaces. Batched here because they share the "spike → verdict →
   maybe-fix" shape and benefit from the Phase-A clean analyzer.
5. **Phase E — Docs + ledger curation (§4.8).** Last, because regenerated SUMMARY/TOTAL and the
   zeroed `_pending.md` must reflect the *final* state of code + analyzer.
6. **Phase F — Verify gates + code review.** Full CLAUDE.md gate run on every changed module
   (`go test -race`, `go vet`, `go build`, `docker buildx bake` per touched `go.mod`,
   `redis-key-guard.sh`), then `plan-adherence` + `backend-guidelines` reviewers before PR.

Critical-path note: Phase A → Phase E is the longest chain (analyzer must precede doc
regeneration). B1.1's event-contract change is the only cross-service coupling and is front-loaded
in Phase B.

---

## 6. Alternatives Considered

**A. Execution structure.**
- *Bucket-by-bucket sequential* (B1→B2→…→docs). Simple to track but interleaves
  analyzer/doc work badly: you'd regenerate SUMMARYs against a noisy analyzer, then again after
  §4.7. Rejected.
- *Layered phases (chosen).* Analyzer first → fixes → spikes → docs last. One SUMMARY
  regeneration, clean signal for spikes, hot-path fixes early. Slightly more upfront ordering
  discipline. **Chosen** — the doc roll-up genuinely depends on everything else being final.
- *Risk-first, docs continuous.* Land B1 bugs immediately, update docs incrementally. Rejected:
  incremental ledger edits churn `_pending.md` repeatedly and the analyzer noise persists through
  most of the task.

**B. Analyzer depth (open Q4).**
- *Minimal — register everything.* Move all four classes straight to the exception registry, no
  tool change. Cheap now, but every future version pass re-triages the same false positives — the
  exact pain the PRD calls out. Rejected.
- *Maximal — full decompiler-grade descent.* Chase every opaque buffer. Unbounded cost for a tool
  that's an aid, not the oracle. Rejected.
- *Bounded fix-in-tool + register the opaque residue (chosen).* Fix the three tractable classes
  outright (machinery already present), generalize descent for self-describing types, register
  only genuinely-opaque buffers. **Chosen** — matches the PRD's "fix in tool first" with a
  defensible stopping line.

**C. B1.1 event contract.**
- *New dedicated affected-area event.* Cleaner-looking but duplicates the mist creation/broadcast
  path and orphans the existing consumer. Rejected.
- *Extend existing `CreatedBody` (chosen).* The model already holds skill id/level; only `Type`
  is new. One consumer, one producer, minimal blast radius. **Chosen.**

**D. JMS cash bodies.** PRD mandates region-dispatched bodies over a 3rd nested guard (2-guard
cap). No alternative explored — this is a fixed constraint; design only specifies *how* (§3.2).

---

## 7. Testing Strategy

- **Byte-level wire tests are the oracle** (§3.3). Every B1/B4/B5 wire change and every in-task
  B3/B6 fix gets a per-version byte assertion. No change merges on analyzer verdict alone.
- **Handler-logic fixes (B2.1, B2.3)** get handler-level tests asserting the routed branch /
  emitted command for the corrected discriminator/mode.
- **Analyzer enhancements (§4.7)** get unit tests in `tools/packet-audit` for each false-positive
  class (a fixture that previously produced ❌/🔍 now produces ✅), plus the integration check: a
  full four-version run diffed against the curated registry.
- **Regression guard:** closed items (storage `Show`, `MonsterControl`, SETFIELD/WarpToMap gates)
  must stay green and untouched — a no-diff assertion on those files / their tests.
- **CLAUDE.md gates** (Phase F): `go test -race ./...`, `go vet ./...`, `go build ./...` clean per
  changed module; `docker buildx bake atlas-<svc>` for **every** service whose `go.mod` is touched
  (atlas-maps, atlas-channel, atlas-cashshop at minimum); `tools/redis-key-guard.sh` clean; nesting
  `awk` clean.

---

## 8. Risks & Mitigations

| Risk | Mitigation |
|---|---|
| **B1.1 mist `Type` source unknown** (model has no explicit type) | Confined Phase-B spike reads the atlas-maps creation path + IDA `OnAffectedAreaCreated`; contract change (add `Type`) is identical whether the value is constant or skill-derived. Not a blocker. |
| **JMS body needs a field the existing purchase command can't carry** | Surface as a finding; extend `RequestPurchaseCommandBody` minimally. Default expectation: currency+serial suffices. No new topic. |
| **Sub-struct descent scope creep** (could swallow the whole task) | Hard boundary (§2/Q4): descend only self-describing types; opaque residue → registry. "Done" = named classes clean vs registry, not a perfect decompiler. |
| **IDA spike (B3/B6) surfaces a large new bug** | Fix in-task per PRD, with byte test + gate; if a spike reveals work clearly beyond closeout scope, it is registered as a *new* tracked task, not silently deferred back into `_pending.md`. |
| **`docker buildx bake` catches a missing `COPY` not seen by `go build`** | Phase F bakes every touched service from worktree root, per CLAUDE.md — not optional. |
| **Hot-path caller sweep missed** (B1.2 chat, B1.3 quest) | Caller updates are part of the same change; `go build ./...` across modules + handler tests catch unmigrated callers. |
| **Regression to closed items** | Explicit no-touch + green assertion (§7). |

---

## 9. Definition of Done

Mirrors PRD §10 acceptance criteria; the design adds the *how-verified* column implicitly via §3
(byte tests) and §7 (gates). In short: B1.1–B1.5, B2.1–B2.3, B3.1–B3.6, B4.1, B5.1, B6 all
resolved with tests + IDA evidence + correct four-version gates; §4.7 analyzer clean against the
registry; §4.8 four SUMMARYs + zeroed `_pending.md` (both copies) + "baseline complete" TOTAL +
new-version-pass guide; closed items green; all CLAUDE.md gates pass; code review run before PR.
