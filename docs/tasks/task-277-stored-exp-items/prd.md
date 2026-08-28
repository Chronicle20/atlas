# Stored-EXP Items (Solomon's Blessing / Gachapon EXP Tickets) — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-28
---

## 1. Overview

Two serverbound operations that let a character convert an item or a stored counter into
character EXP are absent from Atlas end-to-end. `USE_SOLOMON_ITEM`
(`CWvsContext::SendExpUpItemUseRequest`) is the "EXP-up item" path: the client sends a
dedicated request when the player double-clicks a stored-EXP item, and the server grants
EXP and consumes the item. `USE_GACHA_EXP` (`CWvsContext::SendTempExpUseRequest`) is the
redemption of an accumulated *gachapon experience* counter held on the character: the
player banks EXP into the counter over time and later converts the whole balance into
real EXP.

The persistence half of the gachapon counter already exists but is inert. `characters`
carries a `gachapon_experience` column
(`services/atlas-character/atlas.com/character/character/entity.go:40`), the value flows
through the character model, builder, provider, administrator and REST surface, a
`stat.TypeGachaponExperience` constant exists
(`libs/atlas-constants/stat/constants.go:27`), atlas-channel already maps that stat to a
value in its `STAT_CHANGED` fan-out
(`services/atlas-channel/atlas.com/channel/kafka/consumer/character/consumer.go:237` and
`services/atlas-channel/atlas.com/channel/character/snapshot/registry.go:264,307`), and
`libs/atlas-packet/stat/clientbound/changed.go:86,137` encodes it as a 4-byte stat. What
does not exist is anything that ever *credits* the counter, anything that *debits* it,
and any serverbound codec or handler for either operation. A grep for
`solomon|GachaExp|gacha_exp` over `services/` + `libs/` returns only the encode-side stat
plumbing.

The packet coverage matrix (`docs/packets/audits/STATUS.md`) currently records
`USE_SOLOMON_ITEM` as ❌ on gms_v83 (`0x09D`), gms_v84 (`0x0A1`), gms_v87 (`0x0A5`),
gms_v92 (`0x0B2`), gms_v95 (`0x0B5`) and jms_v185 (`0x071`), and ⬜ (n-a) on gms_v48,
gms_v61, gms_v72 and gms_v79; `USE_GACHA_EXP` is ❌ on the same six columns
(`0x09E`/`0x0A2`/`0x0A6`/`0x0B3`/`0x0B6`/`0x072`) with the same four ⬜. The ⬜ on gms_v72
and gms_v79 is **disputed**: the feature is expected to exist from GMS v72.1 onward, so
those two cells are treated as unresolved registry gaps to be settled by evidence in this
task, not as settled n-a.

## 2. Goals

Primary goals:

- Implement serverbound codecs for `USE_SOLOMON_ITEM` and `USE_GACHA_EXP` in
  `libs/atlas-packet` across every version where the client sends them, version-gated with
  the `MajorAtLeast` idiom where the layout diverges.
- Resolve the gms_v72 / gms_v79 matrix cells from client evidence: either add the registry
  rows and opcodes (promoting the cells out of ⬜) or record a `feature-na-evidence.yaml`
  entry proving the client genuinely lacks the op.
- Route both ops through `atlas-channel` handlers registered in every applicable seed
  template.
- Grant real character EXP on Solomon's Blessing use, with the item consumed exactly once
  and eligibility rules (level bounds, item validity) enforced server-side.
- Redeem the stored `gachapon_experience` balance to real EXP on `USE_GACHA_EXP`, zeroing
  the counter atomically and pushing both the EXP change and the counter change to the
  client.
- **Credit** the `gachapon_experience` counter from its real in-game accrual source, so
  the redemption path is live rather than inert.
- Promote the affected packet-matrix cells to ✅ with byte fixtures and pinned evidence.

Non-goals:

- The potential / cube / enhancement family (`ITEM_RELEASE_REQUEST`,
  `ITEM_OPTION_UPGRADE_USE`, `HYPER_UPGRADE_ITEM_USE`) — a separate L-sized subsystem.
- Parsing and applying the unhandled `Item.wz` `spec` effect families (`expinc`, `exp`,
  `expBuff`, `mesoupbyitem`, …) documented in
  `docs/research/missing-features/items-and-consumables.md` §5. If a Solomon item's EXP
  value happens to live in a `spec` field, this task parses **only** that field for
  **only** these items; the broader spec-coverage gap stays out.
- Equip item level/EXP (equip leveling).
- `USE_ITEMEFFECT` / `SHOW_ITEM_EFFECT`.
- The `gms_12` template. The matrix carries no column for gms_v12 on either op, and the
  cash-item-use family is unregistered there generally.
- Any change to the already-verified wire format of an unrelated op or version.

## 3. User Stories

- As a player, I want to double-click a Solomon's Blessing item and receive its EXP so the
  item is not dead weight in my inventory.
- As a player, I want a Solomon's Blessing item that I am ineligible to use (out of the
  item's level range) to be rejected without being consumed, so I do not lose it.
- As a player, I want EXP that I bank into the gachapon-EXP counter to survive logout and
  be visible on my character.
- As a player, I want to redeem my banked gachapon EXP and see both my EXP bar and my
  banked balance update immediately.
- As a player, I want redeeming an empty gachapon-EXP balance to be a no-op rather than a
  disconnect or a phantom EXP grant.
- As an operator, I want both operations to work identically on every tenant version the
  client supports them on, so tenants do not silently diverge.

## 4. Functional Requirements

### 4.1 Evidence derivation (prerequisite to all of §4.2–§4.5)

FR-1. The serverbound request layout for both ops MUST be derived from the client, not
assumed. Derivation follows `docs/packets/IMPLEMENTING_A_PACKET.md` §0–4 using the
checked-in IDA exports (`docs/packets/ida-exports/`) and, where those are insufficient,
live IDA. No field name, width, order, or item-ID value in the implementation may
originate from remembered MapleStory knowledge or from a Cosmic source file alone.

FR-2. The following values are explicitly **unknown at spec time** and MUST be established
from evidence during design:
  - the field layout of each request (whether it carries an inventory position, an item
    ID, both, or a trailing timestamp);
  - which item IDs / WZ classification the client gates `SendExpUpItemUseRequest` on, and
    where the EXP amount and any level bounds live in WZ;
  - the client's own gating for `SendTempExpUseRequest` (whether it self-gates on a
    non-zero displayed balance);
  - whether either op has a distinct clientbound response beyond `STAT_CHANGED`, and if so
    its failure arms;
  - the real accrual source that credits `gachapon_experience` (see §4.5);
  - whether gms_v72 and gms_v79 send these ops at all (see FR-3).

FR-3. gms_v72 and gms_v79 MUST be resolved, not skipped. If the export shows the opcode,
the registry row is added and the cell implemented and verified like any other; if the
export shows the client cannot send it, an entry is added to
`docs/packets/feature-na-evidence.yaml` justifying ⬜ and the disputed status is closed in
writing.

### 4.2 `USE_SOLOMON_ITEM` — Solomon's Blessing

FR-4. `atlas-channel` MUST register a handler for the op on every in-scope version.

FR-5. The handler MUST validate, server-side and before any grant:
  - the character owns the claimed item in the claimed slot;
  - the item is a Solomon-family item per the WZ classification established in FR-2;
  - the character satisfies the item's eligibility bounds (level range) as established in
    FR-2.

FR-6. On a validation failure the item MUST NOT be consumed and no EXP MUST be granted.
The failure MUST be logged; any client-visible failure response is only sent if FR-2
establishes one exists.

FR-7. On success the handler MUST consume exactly one of the item and award the item's EXP
amount via the existing seam
`character.Processor.AwardExperience(field, characterId, distributions, showEffect)`
(`services/atlas-channel/atlas.com/channel/character/processor.go:331`) using distribution
type `ITEM` (`ExperienceDistributionTypeItem`,
`services/atlas-channel/atlas.com/channel/kafka/message/character/kafka.go:107`).

FR-8. Consumption and the EXP grant MUST NOT be able to double-apply on a duplicate or
replayed request.

### 4.3 `USE_GACHA_EXP` — gachapon EXP redemption

FR-9. `atlas-channel` MUST register a handler for the op on every in-scope version.

FR-10. On request, atlas-character MUST, in a single transaction: read the character's
`gachapon_experience`, add that amount to the character's EXP through the existing EXP
award path (distribution type `ITEM`), and set `gachapon_experience` to 0.

FR-11. A redemption with a zero balance MUST be a no-op: no EXP granted, no error to the
client, no counter write. It MUST NOT disconnect the session.

FR-12. The resulting `STAT_CHANGED` MUST carry both the EXP change and a
`GACHAPON_EXPERIENCE` update, so the client's displayed balance returns to zero without a
relog. The existing channel fan-out
(`consumer.go:237`, `snapshot/registry.go:264,307`) already resolves the value; this
requirement is that the stat is included in the emitted update set.

FR-13. Concurrent redemptions for the same character MUST NOT be able to grant the balance
twice.

### 4.4 Level-cap interaction

FR-14. EXP granted by either path MUST respect the same level-cap and overflow behavior as
every other EXP grant in atlas-character; no bespoke cap logic is introduced.

### 4.5 Gachapon EXP accrual

FR-15. `gachapon_experience` MUST be credited by its real in-game source. The source is
unknown at spec time and MUST be established in design from the same evidence bar as FR-1
(client behavior + Cosmic reference + WZ). Candidate sources include monster-kill EXP
diversion during a gachapon-EXP event and a gachapon/reward-pool win; design picks one on
evidence and states why.

FR-16. Regardless of the source chosen, atlas-character MUST expose a durable command seam
to credit the counter by a delta (mirroring the existing `AWARD_EXPERIENCE` command shape,
`services/atlas-character/atlas.com/character/kafka/message/character/kafka.go:21,82,114`),
with the credit persisted and a `STAT_CHANGED` status event emitted carrying
`GACHAPON_EXPERIENCE`.

FR-17. A credit MUST clamp at the counter's storage bound (`uint32`) rather than wrapping.

FR-18. If design's evidence pass cannot establish an accrual source at the FR-1 bar, that
is a genuine blocker and MUST be surfaced to the user before implementation, not silently
converted into a "documented gap." The redemption path (§4.3) is not blocked by it.

### 4.6 Packet coverage

FR-19. Each implemented op × version cell MUST be promoted in
`docs/packets/audits/STATUS.md` via the standard flow: byte-fixture test with a
`packet-audit:verify` marker, pinned evidence record, regenerated matrix. A cell that does
not promote is a failure, not a prose claim.

FR-20. `packet-audit` checks (`matrix`, `fname-doc`, `operations --check`) MUST exit 0.

## 5. API Surface

No new HTTP endpoints are required by the core feature. The existing character REST
resource already exposes `gachaponExperience`
(`services/atlas-character/atlas.com/character/character/rest.go:23,96`) and the existing
PATCH/administrator path already knows the `GachaponExperience` column
(`administrator.go:101`).

New Kafka surface (exact naming to be fixed in design against the existing command
conventions in `kafka/message/character/kafka.go`):

- **Command — credit gachapon experience.** Keyed by character id; body carries a delta
  and a reason/source discriminator. Consumed by atlas-character.
- **Command — redeem gachapon experience.** Keyed by character id; zero body beyond the
  character. Consumed by atlas-character; performs FR-10 atomically.
- **Status event.** Both commands emit the existing `STAT_CHANGED`
  (`kafka.go:234`) status event including `GACHAPON_EXPERIENCE`; the redemption
  additionally emits the existing `EXPERIENCE_CHANGED` event
  (`StatusEventTypeExperienceChanged`).

Error cases are server-side only: an unknown character, a zero balance (no-op per FR-11),
and a clamped credit (FR-17) are logged, not surfaced as new client packets unless FR-2
establishes a real response op.

New packet surface:

- `libs/atlas-packet/character/serverbound/` (or the family directory design selects): one
  immutable struct per op with both `Encode` and `Decode`, version-gated per FR-1.

## 6. Data Model

No schema migration is required for the counter itself — `characters.gachapon_experience`
already exists as `uint32 gorm:"not null;default=0"`
(`services/atlas-character/atlas.com/character/character/entity.go:40`) and is
tenant-scoped by the existing character entity's tenant column.

If design's FR-2 pass establishes that Solomon EXP amounts or level bounds live in a WZ
`spec` field that `atlas-data`'s consumable reader does not parse
(`services/atlas-data/atlas.com/data/consumable/reader.go`), that reader gains **only** the
fields those items need, plus the matching model accessor and REST field. This carries the
same ingest-order caveat that task-219's morph coupon documented: existing tenants whose
item WZ was ingested before the change will not have the new spec values until
re-ingested. If so, an operational re-ingest follow-up MUST be recorded in `docs/TODO.md`.

No new entity is introduced.

## 7. Service Impact

| Service / library | Change |
|---|---|
| `libs/atlas-packet` | Two new serverbound codecs (`Encode` + `Decode`), version-gated; byte-fixture tests per cell. |
| `services/atlas-channel` | Two new socket handlers; wiring in `main.go`; the Solomon handler calls the existing `character.Processor.AwardExperience` seam and an item-consume seam; the gacha handler produces the redeem command. |
| `services/atlas-character` | New consumers for the credit and redeem commands; atomic redeem (award EXP + zero counter); `STAT_CHANGED` emission including `GACHAPON_EXPERIENCE`; clamped credit. |
| `services/atlas-consumables` | Item validation/consumption for the Solomon family, if design routes consumption here rather than through atlas-inventory directly. |
| `services/atlas-data` | Conditional — only if FR-2 shows a needed WZ `spec` field is unparsed (§6). |
| `services/atlas-configurations` | Seed templates for every in-scope version gain both handler registrations. |
| Accrual-source service (TBD, FR-15) | Produces the credit command. Identified in design. |
| `docs/packets/` | Registry rows (incl. the gms_v72 / gms_v79 resolution), evidence records, regenerated `STATUS.md` / `status.json`. |

## 8. Non-Functional Requirements

- **Multi-tenancy.** Every command, event, and REST call carries tenant context; the
  counter is read and written only within the character's tenant. No cross-tenant read.
- **Version safety.** No wire change to an already-verified op × version cell. Divergent
  fields use the `MajorAtLeast` version-gate idiom, never a raw major comparison.
- **Atomicity.** Redemption (FR-10) and consumption+grant (FR-7) are each single-writer
  and non-double-appliable; concurrency is asserted by test, not by inspection.
- **Observability.** Every rejected use logs the character id, item id, and the specific
  rule that rejected it, at a level consistent with neighboring handlers.
- **Security.** All eligibility, ownership, and balance checks are server-side. A crafted
  packet claiming an item the character does not own, or an out-of-range level, grants
  nothing.
- **Performance.** Both paths are single-character, low-frequency operations; no
  additional per-tick or per-map work is introduced.

## 9. Open Questions

1. **Accrual source (FR-15).** Which in-game event credits `gachapon_experience`? Design
   must settle this from evidence; if it cannot, escalate per FR-18.
2. **gms_v72 / gms_v79 (FR-3).** The matrix says ⬜ (n-a); the expectation is that the
   feature exists from GMS v72.1. Which is right is an evidence question, and the answer
   changes the version spread from six columns to eight.
3. **Solomon item identity (FR-2).** Which WZ classification / item-ID range the client
   gates `SendExpUpItemUseRequest` on, and where the EXP amount and level bounds live.
4. **Response packets (FR-2).** Whether either op has a clientbound response beyond
   `STAT_CHANGED`, and if so whether it has failure arms that need writers.
5. **Consumption ownership.** Whether the Solomon item is consumed through
   `atlas-consumables` (consistent with other use-tab items) or directly through the
   inventory seam, given the client uses a dedicated opcode rather than the normal
   use-item path.
6. **jms_v185 divergence.** The JMS opcodes (`0x071`/`0x072`) sit in a very different
   numeric range; whether the payload also diverges is unverified.

## 10. Acceptance Criteria

- [ ] Request layouts for both ops derived from client evidence and recorded; no guessed
      field, width, or item ID anywhere in the change.
- [ ] gms_v72 and gms_v79 resolved: either registry rows added and cells implemented, or a
      `feature-na-evidence.yaml` entry justifying ⬜, with the reasoning written down.
- [ ] Serverbound codecs for `USE_SOLOMON_ITEM` and `USE_GACHA_EXP` exist in
      `libs/atlas-packet` with both `Encode` and `Decode`, version-gated via `MajorAtLeast`.
- [ ] Byte-fixture tests with `packet-audit:verify` markers exist for every in-scope
      op × version cell.
- [ ] `docs/packets/audits/STATUS.md` shows ✅ for `USE_SOLOMON_ITEM` and `USE_GACHA_EXP`
      on every in-scope column, and the regenerated matrix is committed alongside its
      evidence records.
- [ ] `packet-audit matrix`, `packet-audit fname-doc --check`, and
      `packet-audit operations --check` all exit 0.
- [ ] Both handlers are registered in every in-scope seed template
      (`services/atlas-configurations/seed-data/templates/`), verified by grep across all
      templates.
- [ ] Solomon use: eligible character receives the item's EXP, item count decrements by
      exactly one, `EXPERIENCE_CHANGED` observed.
- [ ] Solomon use: ineligible character (out of level bounds) receives no EXP and the item
      count is unchanged — asserted by test.
- [ ] Solomon use: a request naming an item the character does not own grants nothing —
      asserted by test.
- [ ] Gacha redeem with a non-zero balance grants exactly the balance as EXP and leaves
      `gachapon_experience` at 0 in the same transaction — asserted by test.
- [ ] Gacha redeem with a zero balance is a no-op, emits no EXP grant, and does not
      disconnect — asserted by test.
- [ ] The `STAT_CHANGED` emitted by redemption includes a `GACHAPON_EXPERIENCE` update —
      asserted by test.
- [ ] Concurrent/duplicate redemption cannot grant the balance twice — asserted by test.
- [ ] The credit seam persists a delta, clamps at `uint32` rather than wrapping, and emits
      `STAT_CHANGED` — asserted by test.
- [ ] The accrual source identified in FR-15 produces the credit command, so a fresh
      character can bank and then redeem gachapon EXP end-to-end without manual DB edits.
- [ ] `docs/research/missing-features/items-and-consumables.md` "Wholly missing" §3 updated
      to reflect the implemented state.
- [ ] If a WZ `spec` field was added to `atlas-data`, a tenant re-ingest follow-up is
      recorded in `docs/TODO.md`.
- [ ] Flagless `tools/verify.sh` exits 0.
- [ ] Code review passes before the PR is opened.
