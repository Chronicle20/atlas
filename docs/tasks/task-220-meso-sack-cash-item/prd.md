# Meso Sack Cash Item — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-13
---

## 1. Overview

Meso sacks are cash-inventory items (WZ classification `520`, `Item.wz/Cash/0520.img`) that
convert into a fixed amount of mesos when used. The item family exists in every client version
Atlas serves — `5200000` (1,000,000), `5200001` (5,000,000) and `5200002` (10,000,000) are present
in all ten tenant versions — and the client already has a send path for them:
`GetCashSlotItemType` maps `item.ClassificationCurrencySack` to cash-slot type **19**
(`services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:894`).

Atlas does not implement that branch. `CharacterCashItemUseHandleFunc` has arms for ~14 slot types;
type 19 is not one of them, so a meso sack use falls through every branch and the request is
silently dropped. The player's client stays locked behind its exclusive-request gate until it times
out, and the item is neither consumed nor paid out.

Two pieces are missing beneath the handler as well. The payout amount lives in the WZ node
`info/meso`, which `atlas-data`'s cash reader never parses
(`services/atlas-data/atlas.com/data/cash/reader.go`) and the channel's cash view model never
carries (`services/atlas-channel/atlas.com/channel/data/cash/rest.go`), so the amount is not
reachable from the handler today. And `atlas-character`'s `RequestChangeMeso` rejects an award that
would overflow the `uint32` meso ceiling by returning `ErrMesoOverflow` **without emitting any
event** (`services/atlas-character/atlas.com/character/character/processor.go:838-841`), which would
leave a saga step hanging until timeout rather than failing fast.

This task implements the type-19 branch end to end: parse the amount, credit it atomically against
consumption of the sack, render the meso-gain chat line, and — when the character cannot hold the
mesos — restore the sack and tell the player why.

## 2. Goals

Primary goals:

- Using a meso sack consumes exactly one sack from the CASH inventory and credits the character the
  item's WZ `info/meso` amount, atomically (never one without the other).
- The meso gain renders as the standard meso chat line, and the client's exclusive-request gate is
  released on every outcome (success and every failure).
- A character at or near the meso ceiling keeps their sack and is told they cannot hold any more
  mesos.
- Works on all ten tenant versions: GMS v48, v61, v72, v79, v83, v84, v87, v92, v95, and JMS v185.
- The payout amount is data-driven from WZ — no item id → amount table in code.

Non-goals:

- **Maple Point sacks.** `5200009` / `5200010` (v84, v87, v92, v95) carry `info/maplepoint` and no
  `info/meso`. Paying NX is a separate cash-shop concern. These must be rejected, not silently
  consumed (FR-2.4).
- **Randomized payout distribution.** `5202000` / `5202001` / `5202002` (v92, v95; `5202000` only on
  JMS v185) additionally carry `mesomin` / `mesomax` / `mesostdev`. This task pays their flat
  `info/meso` value and ignores the distribution — matching the reference implementation
  (`UseCashItemHandler.java:420-423` calls `ii.getMeso(itemId)`, which reads `info/meso` only). A
  gaussian roll is a possible follow-up; it is not required for the items to function.
- **`gms_12`.** That template does not register `CharacterCashItemUseHandle` and there is no `gms_12`
  tenant, so it is out of scope. No socket-config template changes are required by this task at all.
- Any other `052x` cash-slot type (transformation coupons, jukebox, pet name tag, …).

## 3. User Stories

- As a player, I want to use a meso sack from my cash inventory so that I receive its mesos and see
  the gain confirmed in chat.
- As a player at the meso ceiling, I want to be told I cannot hold any more mesos and keep my sack,
  so that I do not lose a purchased item to a silent failure.
- As a player, I want my client to become responsive again after using a sack — whether it worked or
  not — so that I am never stuck behind a hung request.
- As a player on a legacy client (v48–v79), I want meso sacks to work identically, since the items
  exist on my version.
- As an operator, I want a failed payout to leave no partial state — no consumed-but-unpaid sack and
  no paid-but-unconsumed sack.

## 4. Functional Requirements

### FR-1 — Data plumbing (`atlas-data`)

- **FR-1.1** The cash reader parses `info/meso` into a first-class `Meso uint32` field on the cash
  `RestModel`. It is **not** folded into the existing `Spec map[SpecType]int32` — it is an award
  amount, not a spec effect.
- **FR-1.2** The field is absent/zero when the WZ node is absent. No default, no fallback constant.
- **FR-1.3** The value is serialized as `meso` on the `cash_items` JSON:API resource, `omitempty`
  consistent with the surrounding fields.
- **FR-1.4** Cash items are stored as JSONB documents (`document.Storage[..., RestModel]`, kind
  `CASH`), so previously ingested tenants will not gain the field until their WZ is re-ingested. The
  task must state the re-ingest requirement in its rollout notes and verify the field is populated
  for each of the ten tenants before the feature is considered live.

### FR-2 — Handler branch (`atlas-channel`)

- **FR-2.1** `CharacterCashItemUseHandleFunc` gains a branch for `CashSlotItemType(19)`, named via a
  new `CashSlotItemTypeCurrencySack` constant alongside the existing named constants.
- **FR-2.2** The branch decodes no sub-body beyond the common `ItemUse` header, subject to FR-6.1.
- **FR-2.3** The branch resolves the item's `meso` amount through `atlas-channel`'s
  `data/cash` processor (see FR-3.1). A failed lookup rejects the use per FR-2.5.
- **FR-2.4** **Fail closed on a missing amount.** If the resolved `meso` is `0` or absent, the use is
  rejected: no item consumed, no meso awarded, a warning logged naming the item id, and the client
  unlocked per FR-5.1. This is the rule that makes the Maple Point sacks safe by construction, and it
  also protects any version whose sack ships without a base `meso` value.
- **FR-2.5** Every rejection path in this branch (unknown item, lookup failure, zero amount) unlocks
  the client per FR-5.1 before returning.
- **FR-2.6** The existing pre-branch guard that verifies the item in `source` matches the claimed
  `itemId` (`character_cash_item_use.go:52-57`) continues to gate this branch; the amount is derived
  from the server-resolved template id, never from a client-supplied value.

### FR-3 — Cash view model (`atlas-channel`)

- **FR-3.1** `atlas-channel`'s `data/cash.RestModel` gains the `meso` field so `GetById` returns it.

### FR-4 — Atomic consume + award (saga)

- **FR-4.1** The use is executed as a saga, not as two independent commands. Both existing actions
  are reused: `DestroyAssetFromSlot` (or `DestroyAsset`, per design) to consume exactly one sack from
  the CASH compartment slot the request named, and `AwardMesos` to credit the amount.
- **FR-4.2** The `AwardMesosPayload` carries `ShowEffect: true` so the meso gain renders as the chat
  line, `Amount` = the WZ `meso` value, and an actor identifying the item use.
- **FR-4.3** A failure of the award step compensates the consume step — the sack is restored to the
  player's inventory. A saga that fails must leave the character's mesos and inventory exactly as
  they were before the request.
- **FR-4.4** A new saga type (e.g. `meso_sack_use`) discriminates these sagas so the channel's
  saga-failure consumer can render the correct client feedback (FR-5.2). It follows the existing
  `SagaTypePointReset` precedent in
  `services/atlas-channel/atlas.com/channel/kafka/consumer/saga/consumer.go:347-359`.

### FR-5 — Client feedback

- **FR-5.1** **Unlock on every outcome.** Every terminal path — success, rejection, saga failure —
  releases the client's exclusive-request gate with an empty `StatChanged` carrying the
  enable-actions flag, exactly as the Vegas-scroll and point-reset paths do. The success path's
  unlock must not race the meso stat update; the design phase pins the ordering.
- **FR-5.2** **Meso-ceiling message.** When the award is rejected because the character cannot hold
  the mesos, the client receives a pink `WorldMessage` telling them they cannot hold any more mesos,
  followed by the FR-5.1 unlock. The exact copy is fixed in the design phase; the requirement is
  that the message names the meso limit as the reason, not a generic failure.
- **FR-5.3** The meso-gain chat line on success comes from the existing `ShowEffect` path
  (FR-4.2) — this task introduces no new success-side writer.

### FR-6 — Wire verification

- **FR-6.1** The design phase must IDA-verify, per version, that cash-slot case 19 of
  `CWvsContext::SendConsumeCashItemUseRequest` sends **no sub-body** beyond the common header (with
  the usual trailing `update_time` on GMS ≤ v84, per `cashsb.UpdateTimeFirst`). The Cosmic reference
  reads no extra bytes, but that is a server-side inference, not wire evidence. If any version does
  send a sub-body, a codec is added under `libs/atlas-packet/cash/serverbound/` and decoded here.
- **FR-6.2** No change may be made to the encoding of any already-verified packet, and the
  `CharacterCashItemUseHandle` opcode registration in the ten templates is untouched.

### FR-7 — Overflow reporting (`atlas-character`)

- **FR-7.1** `RequestChangeMeso`'s overflow path emits a character `ERROR` status event with a new
  `MESO_OVERFLOW` error type, using the existing non-generic `StatusEventMesoErrorBody`
  (`{error, amount}`) shape, before returning `ErrMesoOverflow`. Today it returns silently, which
  would strand a saga step until timeout.
- **FR-7.2** No change is required in `atlas-saga-orchestrator`:
  `handleCharacterMesoErrorEvent`
  (`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/character/consumer.go:166`)
  accepts any `ERROR` event with that body shape and marks the step failed. The design phase must
  confirm the event's acceptance-table registration covers the new saga's award step, and that the
  existing `NOT_ENOUGH_MESO` behavior is unchanged.
- **FR-7.3** The overflow guard's semantics are unchanged: rejection, not clamping. A character who
  cannot hold the full amount receives nothing and keeps the sack.

### FR-8 — Version coverage

- **FR-8.1** The branch is exercised on all ten tenant versions. Version-divergent behavior, if FR-6.1
  finds any, is gated with the `MajorAtLeast` idiom — never a raw `> N` comparison.
- **FR-8.2** Items known present per version (live `atlas-data` `/api/data/cash/items/{id}` query,
  2026-08-13):

  | Item ids | Versions |
  |---|---|
  | `5200000`, `5200001`, `5200002` | all ten |
  | `5200009`, `5200010` (Maple Point — out of scope, FR-2.4) | v84, v87, v92, v95 |
  | `5202000` | v92, v95, JMS v185 |
  | `5202001`, `5202002` | v92, v95 |
  | `5205000` (Maple Point) | none |

## 5. API Surface

No new endpoints. Two response shapes gain a field:

**`GET /api/data/cash/items/{itemId}`** (`atlas-data`) — `cash_items` resource attributes gain:

```json
{ "meso": 1000000 }
```

Omitted when the WZ node is absent. Existing attributes (`slotMax`, `protectTime`,
`stateChangeItem`, `bgmPath`, `spec`, `timeWindows`, `petSkills`, `petSkillAdd`, `tradeBlock`) are
unchanged.

**`atlas-channel`'s `data/cash.RestModel`** — mirrors the same `meso` field so the handler can read it.

Error cases are unchanged: a missing item id remains a `404`, which the handler treats as FR-2.4
rejection.

## 6. Data Model

No relational schema change. `atlas-data` stores cash items as JSONB documents keyed by item id
under document kind `CASH` (`document.Storage[string, RestModel]`), so a new field is additive and
requires **re-ingest of the WZ per tenant version** to populate — there is no backfill migration and
no existing document gains the field on deploy (FR-1.4).

No new entities, no `tenant_id` scoping changes (document storage is already tenant-scoped), and no
change to character or inventory persistence — the meso credit reuses `atlas-character`'s existing
`meso` column and its `uint32` ceiling.

## 7. Service Impact

| Service | Change |
|---|---|
| `atlas-data` | Parse `info/meso` in `cash/reader.go`; add `Meso` to `cash/rest.go`. Reader unit tests. |
| `atlas-channel` | New `CashSlotItemTypeCurrencySack` branch in `character_cash_item_use.go`; `meso` on `data/cash/rest.go`; new saga type + failure arm in `kafka/consumer/saga/consumer.go` rendering the meso-ceiling message; enable-actions on every path. |
| `atlas-character` | Emit the `MESO_OVERFLOW` error status event on the `RequestChangeMeso` overflow path (FR-7.1). |
| `atlas-saga-orchestrator` | Expected no code change (FR-7.2); verify acceptance-table coverage for the new saga's award step and its compensation ordering. |
| `atlas-configurations` | No change — no template edits (FR-6.2). |
| `libs/atlas-packet` | No change expected; a serverbound sub-body codec is added only if FR-6.1 finds one. |
| `libs/atlas-saga` | No change expected — `AwardMesos`, `DestroyAsset`, `DestroyAssetFromSlot` all exist. |

## 8. Non-Functional Requirements

- **Multi-tenancy.** Every lookup and command is tenant-scoped via `tenant.MustFromContext(ctx)`. The
  payout amount is resolved per tenant version — the same item id may carry different amounts across
  versions and must never be cached across tenants.
- **Atomicity.** Consume and award are one saga; no path may leave the pair half-applied (FR-4.3).
- **Idempotency.** A duplicate request for the same slot must not double-pay. The client-side
  exclusive-request gate is the first line of defense (FR-5.1), and the slot/template guard
  (FR-2.6) is the second — the design phase confirms whether a third, server-side guard is warranted.
- **Observability.** Every rejection logs at warn with character id, item id and reason. A successful
  use is debug-logged with the awarded amount. Saga failures already log through the existing
  consumer.
- **Security.** The amount is derived from server-side WZ data keyed by the server-resolved template
  id; no client-supplied value influences the payout.
- **Performance.** One additional `atlas-data` lookup per use — a rare, player-initiated action. No
  hot path is touched.

## 9. Open Questions

1. **Meso-ceiling copy.** Exact player-facing wording for FR-5.2 (reference clients use a message to
   the effect of "You cannot hold any more mesos"). To be pinned in design, verified against version
   string data where available.
2. **`DestroyAsset` vs `DestroyAssetFromSlot`.** Which action correctly targets a CASH-compartment
   slot, and whether the compensator restores to the same slot. Design phase resolves against the
   existing storage/point-reset saga precedents.
3. **Per-version `meso` values.** Only the 83.1 tree is extracted locally, and `atlas-data` does not
   yet expose the field, so the actual amounts on v84–v95 and JMS v185 are unverified. FR-2.4 makes
   this non-blocking (a missing/zero value fails closed), but design should confirm the values after
   the FR-1.1 re-ingest.
4. **`5202000-2` on JMS v185.** Only `5202000` is present there. Whether JMS's copy carries a base
   `info/meso` is unverified — same FR-2.4 fallback applies.
5. **Unlock ordering on success.** Whether the enable-actions unlock should be sent by the handler
   immediately or deferred until the saga's meso stat update lands, so the client renders the new
   balance before it accepts further input.

## 10. Acceptance Criteria

- [ ] `atlas-data` parses `info/meso` into a first-class `Meso` field and serializes it as `meso`;
      reader unit tests cover present, absent, and zero values.
- [ ] `GET /api/data/cash/items/5200000` returns `"meso": 1000000` on the GMS v83 tenant after
      re-ingest, and a non-zero `meso` on all ten tenants for `5200000`/`5200001`/`5200002`.
- [ ] `atlas-channel`'s `data/cash` model exposes `meso`.
- [ ] Using `5200000` with a fresh character credits exactly 1,000,000 mesos, removes exactly one
      sack from the CASH compartment, and renders the meso-gain chat line.
- [ ] Using `5200001` / `5200002` credits 5,000,000 / 10,000,000 respectively.
- [ ] Using a sack whose award would exceed the meso ceiling: the character's mesos are unchanged,
      the sack is still in inventory, and the client shows the meso-ceiling message.
- [ ] Using `5200009` (Maple Point sack) on a v87 tenant consumes nothing, awards nothing, logs a
      warning, and unlocks the client.
- [ ] `5202000` on a v92 tenant pays its flat `info/meso` amount (distribution ignored, per non-goal).
- [ ] The client is responsive after every one of the above outcomes — no hung exclusive request.
- [ ] FR-6.1 IDA verification is recorded per version in the design doc, with addresses.
- [ ] `atlas-character` emits `MESO_OVERFLOW`, and the saga step fails fast (no timeout wait) —
      demonstrated by a test or a logged transaction.
- [ ] `go test -race ./...`, `go vet ./...` and `go build ./...` clean in `atlas-data`,
      `atlas-channel`, `atlas-character` (and `atlas-saga-orchestrator` if touched).
- [ ] `tools/lint.sh --check`, `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh` and
      `tools/skill-job-id-guard.sh` clean from the repo root.
- [ ] No `go.mod` changed, so no `docker buildx bake` target is mandatory; if any module's `go.mod`
      does change, bake every affected service.
- [ ] Code review run before PR.
