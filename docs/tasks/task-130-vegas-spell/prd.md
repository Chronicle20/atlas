# Vega's Spell — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-02
---

## 1. Overview

Vega's Spell is a classic Cash Shop consumable pair — **Vega's Spell 10** (item `5610000`) and **Vega's Spell 60** (item `5610001`), both defined in `Item.wz/Cash/0561.img.xml` (verified present in the v83-era WZ; the WZ entries carry only `info` nodes, so the rate-boost values are server-side, not data-driven). Using a Vega's Spell consumes the vega cash item together with an upgrade scroll and applies that scroll to a target equip at a boosted success rate:

- Vega's Spell 10: a scroll whose success rate is exactly **10% is applied at 30%**.
- Vega's Spell 60: a scroll whose success rate is exactly **60% is applied at 90%**.

Semantics confirmed against Cosmic (`UseCashItemHandler.java` itemType 561; `ItemInformationProvider.scrollEquipWithId`) and retail GMS documentation (owner-confirmed 2026-07-02). The boost applies only when the scroll's natural success rate matches the vega variant exactly; other scrolls are not affected.

Atlas already has the complete scroll pipeline in `atlas-consumables` (`RequestScroll` → reserve → `ConsumeScroll` → `PassScroll`/`FailScroll`), with an explicit `// TODO consume vega scroll` at `services/atlas-consumables/atlas.com/consumables/consumable/processor.go:641`. The channel-side cash-item-use handler (`services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go`) already classifies category 561 to a `CashSlotItemType` (68 for GMS < 95, 71 for GMS ≥ 95) but has no dispatch arm, so vega use currently falls through to the "unhandled cash item type" warning. This task closes that gap end-to-end for all supported tenants.

## 2. Goals

Primary goals:
- A player can use Vega's Spell 10/60 from the cash inventory to scroll an equip with the matching 10%/60% scroll at the boosted rate.
- Both the vega item and the scroll are consumed atomically (all-or-nothing with the reservation flow); the equip is mutated per the standard scroll pipeline.
- Client receives correct feedback: the `VEGA_SCROLL` clientbound packet (start + result), the inventory updates, and the map-broadcast scroll effect — with the result resolved **immediately** (no server-side delay).
- Works on every supported tenant version: GMS v83, v84, v87, v92, v95 and JMS v185, with IDA-verified packet layouts/opcodes and byte fixtures per project packet standards.

Non-goals:
- Blocking cash-shop entry while a vega scroll animation plays (`cash_shop_entry.go:29` TODO) — explicitly deferred; the TODO stays.
- Replicating Cosmic's 3-second delayed result timer — owner decided the server resolves immediately.
- Any other category-561-adjacent or new cash item types.
- Changes to scroll math for non-vega paths.

## 3. User Stories

- As a player, I want to double-click Vega's Spell 10 and select an equip plus a 10% scroll so that the scroll is applied with a 30% success chance.
- As a player, I want to use Vega's Spell 60 with a 60% scroll so that it succeeds 90% of the time.
- As a player, I want a failed validation (wrong scroll rate, missing item, equip with no upgrade slots) to consume nothing so that I never lose a cash item to a no-op.
- As a nearby player, I want to see the scroll pass/fail effect on the user so that scrolling remains socially visible, exactly as with normal scrolls.

## 4. Functional Requirements

### FR-1 Serverbound handling (atlas-channel)
1. `CharacterCashItemUseHandleFunc` gains a dispatch arm for the vega `CashSlotItemType` (68 pre-v95 GMS, 71 GMS ≥ 95; JMS value verified in design).
2. Sub-packet body per Cosmic reference: `int32(1)`, `int32 equipSlot`, `int32(2)`, `int32 useSlot`. The exact byte layout MUST be IDA-verified per supported version during design/implementation — Cosmic is the starting hypothesis, not evidence.
3. The handler resolves the vega item (cash inventory, slot from the outer `ItemUse` packet, template id `5610000`/`5610001`), then forwards a command to `atlas-consumables` carrying: character id, vega item id + slot, scroll (use-inventory) slot, and equip slot.

### FR-2 Validation (atlas-consumables)
Before consuming anything, the service validates:
1. Vega item exists at the given cash slot and its template id is `5610000` or `5610001`.
2. Scroll exists at the given use slot and passes the existing `ValidateScrollUse` rules against the target equip.
3. The scroll's natural success rate (from atlas-data consumable info) matches the vega variant exactly — 10 for Vega 10, 60 for Vega 60. A mismatch rejects the request **without consuming anything**.
4. Target equip exists at the given equip slot and has ≥ 1 upgrade slot (existing rule; clean-slate/spike/cold-protection scrolls are not valid vega targets because their rates are not 10/60 — no special-casing needed beyond FR-2.3).

### FR-3 Atomic consumption
1. The vega cash item (×1) and the scroll (×1) are reserved together via the existing compartment reservation flow and consumed only when the scroll is actually applied.
2. Any failure between reservation and application cancels the reservation (existing `ConsumeError` semantics) — the player loses nothing.
3. Note: the vega item lives in the **cash** compartment; the reservation/consume path must support `inventory.TypeValueCash` alongside `TypeValueUse` (verify in design — the current `RequestScroll` only reserves use-inventory assets).

### FR-4 Boosted scroll application
1. The scroll is applied through the existing `ConsumeScroll` machinery (stat changes, slot decrement, level increment, curse handling) with exactly one difference: `successProb` becomes 30 (Vega 10) or 90 (Vega 60) at `processor.go:641-642`.
2. `whiteScroll` is forced false and `legendarySpirit` is false on the vega path.

### FR-5 Client feedback (immediate)
1. A new `VEGA_SCROLL` clientbound writer is added to `libs/atlas-packet`. Body per Cosmic: a single mode byte — `0x40` (start animation), `0x41` (success), `0x43` (fail). (`0x42`/`0x44` render "this item cannot be used"; `0x39`/`0x45` crash the client — documented, not used unless design chooses `0x42` for FR-2.3 rejection.)
2. On resolution the server sends the start and result modes in immediate succession (no 3s timer), followed by the standard inventory modify packets, the map-broadcast `ItemUpgrade` (SHOW_SCROLL_EFFECT, existing writer at `libs/atlas-packet/character/clientbound/item_upgrade.go`, `legendarySpirit=false`, `whiteScroll=false`), and enable-actions. Design verifies the client tolerates back-to-back start+result; if the client requires the animation gap, the packets are still sent immediately and the client animates on its own clock.
3. The mode byte MUST be config-resolved via the tenant `operations` table, never hard-coded (project dispatcher rule; owner-established uniformity policy).

### FR-6 Multi-version support
1. The `VEGA_SCROLL` opcode is IDA-verified for each supported version (GMS v83/v84/v87/v92/v95, JMS v185) and registered in every tenant seed template's writer list, with `operations` entries for the mode values per version.
2. Byte fixtures with `packet-audit:verify` markers cover the new writer per version, per project packet-verification standards.
3. Live tenant configs must be patched (new opcodes do not reach existing tenants via seed templates — known gotcha; the rollout note goes in plan.md).

## 5. API Surface

- **REST:** none.
- **Kafka:** one new or extended command consumed by `atlas-consumables` — either an extended `CommandRequestScroll` body (optional vega item id + cash slot, backward-compatible) or a distinct vega command/saga. **Decision deferred to design** (owner). Existing `RequestScrollBody` producers must be unaffected.
- **Socket:** new dispatch arm in the existing cash-item-use serverbound handler (no new opcode serverbound); new `VEGA_SCROLL` clientbound writer (new opcode per version).

## 6. Data Model

- No database changes.
- `libs/atlas-constants/item`: add `VegasSpell10 = 5610000`, `VegasSpell60 = 5610001` (naming per lib conventions) and an `IsVegasSpell(id)` classifier — verified absent today; check the lib index again at implementation time per DOM-21.

## 7. Service Impact

| Service / lib | Change |
|---|---|
| `atlas-channel` | Vega dispatch arm in `character_cash_item_use.go`; emit command; register new writer |
| `atlas-consumables` | Vega-aware request/validate/consume path; replace `processor.go:641` TODO |
| `libs/atlas-packet` | New `VEGA_SCROLL` clientbound writer + byte fixtures per version |
| `libs/atlas-constants` | Vega item ids + classifier |
| Tenant seed templates | Writer opcode + operations entries for all six versions |
| Live tenant configs | Post-merge patch + channel restart (rollout step, not code) |

## 8. Non-Functional Requirements

- **Multi-tenancy:** all opcodes/modes resolved from tenant config; no version conditionals keyed on hard-coded bytes.
- **Atomicity:** reservation-based consume; no partial loss of vega/scroll on any failure path.
- **Observability:** debug logs on vega use, boost application, and rejection reasons, consistent with the existing scroll logging.
- **Verification bar:** `go test -race`, `go vet`, `go build` per changed module; `docker buildx bake` for `atlas-channel` and `atlas-consumables` (and all services if a shared lib changes go.mod); `tools/redis-key-guard.sh`; packet fixtures + template `--check` tooling clean.

## 9. Open Questions

1. Delivery mechanism: extend `RequestScrollBody` vs. new command vs. saga (design phase).
2. Exact serverbound sub-packet layout and `VEGA_SCROLL` opcode per version (IDA verification during design/implementation; Cosmic layout is the hypothesis).
3. Whether FR-2.3 rejection should surface `VEGA_SCROLL 0x42` ("this item cannot be used") or silently enable-actions (design; 0x42 is documented-safe per Cosmic comment but should be client-verified).
4. Whether the existing equip addressing (`slot.GetSlotByPosition` + `Equipment().Get`) covers equips sitting in the equip **inventory** (positive slots) as well as equipped items (negative slots) — Cosmic reads the equip inventory; the Atlas scroll path must address the same target set (design).

## 10. Acceptance Criteria

- [ ] Using Vega's Spell 10 with a 10% scroll on a valid equip consumes exactly 1 vega item + 1 scroll and applies the scroll with 30% success probability (log-verifiable roll threshold).
- [ ] Using Vega's Spell 60 with a 60% scroll applies it at 90%.
- [ ] A vega used with a scroll whose rate is not exactly 10/60 (respectively) is rejected and nothing is consumed.
- [ ] Missing vega/scroll/equip, or equip with 0 upgrade slots, rejects without consumption.
- [ ] Success and failure both produce the `VEGA_SCROLL` result packet, inventory updates, map-broadcast scroll effect, and enable-actions, with no server-side delay.
- [ ] White-scroll and legendary-spirit flags are false throughout the vega path.
- [ ] `VEGA_SCROLL` writer registered with IDA-verified opcodes in all six tenant seed templates (gms_83/84/87/92/95, jms_185) with per-version operations entries and passing byte fixtures.
- [ ] `// TODO consume vega scroll` no longer exists in `processor.go`; the `cash_shop_entry.go:29` TODO remains untouched.
- [ ] Full verification bar (§8) clean; code review run before PR.
