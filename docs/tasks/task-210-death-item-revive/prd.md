# Death Items — Wheel of Destiny revive & tomb effect — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-10

---

## 1. Overview

When a character dies in MapleStory, the client presents a death dialog. A player holding a
Wheel of Destiny (item `5510000`) may spend a charge to revive **in the map they died in**
rather than being sent to the map's return map; bystanders see a tomb / upgrade effect play
over the revived character. Separately, a player holding a Safety Charm (`5130000`) or one of
the ETC protection items loses no experience on death, and the client is told how many uses of
the charm remain.

Atlas already implements a large part of the *mechanic* but none of the *protocol*.
`services/atlas-channel/atlas.com/channel/respawn/processor.go` runs off the
`MAP_CHANGE`-while-dead path (`socket/handler/map_change.go:54`): it checks Cash inventory for
`item.WheelOfFortuneId`, redirects the respawn target from `mapData.ReturnMapId()` back to the
current map, and builds a `CharacterRespawn` saga that destroys one wheel, sets HP to 50,
deducts experience, cancels buffs, and warps. Safety-charm / Easter Basket / ProtectOnDeath
detection suppresses the experience deduction (`findProtectiveItem`, `calculateExpLoss`).

What is missing is everything the client can see or say about it. The serverbound
`USE_DEATHITEM` (`CUserLocal::RequestUpgradeTombEffect`) has no codec, no handler, and no
template registration on any version; the clientbound `SHOW_UPGRADE_TOMB_EFFECT`
(`CUserRemote::OnShowUpgradeTombEffect`) has no codec and no writer. The `EffectProtectOnDie`
codec exists in `libs/atlas-packet/character/clientbound/effect.go:374` complete with its
`usesRemaining` field, but `grep ProtectOnDie services/` returns nothing — no service ever
emits it. The consequence is that today the wheel is consumed silently and unconditionally,
the player is never offered the choice, charges are not modelled, and neither the dying player
nor anyone else in the map sees any effect.

This task closes the protocol gap end to end: both packets implemented and verified on every
version that has them, charges modelled on the asset quantity, and the existing protect-on-die
effect finally wired to the consumption it describes.

## 2. Goals

Primary goals:

- Implement the `USE_DEATHITEM` serverbound codec, channel handler, and seed-template
  registration on every version that carries the opcode (v83, v84, v87, v92, v95, JMS185).
- Implement the `SHOW_UPGRADE_TOMB_EFFECT` clientbound codec, writer, and map broadcast on
  every version that carries the opcode (v72, v79, v83, v84, v87, v92, v95, JMS185).
- Support **both** trigger paths for the in-map revive: the existing implicit
  `MAP_CHANGE`-while-dead consume, and an explicit client-driven `USE_DEATHITEM` request. The
  two must not double-consume.
- Model death-item **charges** so that a wheel or charm with multiple uses decrements rather
  than being destroyed whole, and report the remaining count to the client.
- Emit the existing `EffectProtectOnDie` / `EffectProtectOnDieForeign` effect when a protective
  item is consumed on death.
- Promote all 14 affected packet × version coverage-matrix cells to ✅ within this task.

Non-goals:

- `USE_ITEMEFFECT` / `SHOW_ITEM_EFFECT` (the cosmetic active-effect-item family, item 12 of
  `docs/research/missing-features/packet-gap-inference.md`). Adjacent, separately scoped.
- Water of Life / pet revive (item 6 of `docs/research/missing-features/items-and-consumables.md`).
- Reworking the experience-loss formula in `calculateExpLoss` (its 1% / 5% / 10% tiers and the
  `currentExp`-relative base are pre-existing and explicitly untouched here).
- Cash-shop purchase flow for the wheel or charm.
- The `atlas-ui` surface — no frontend change.

## 3. User Stories

- As a player who died in a boss map while holding a Wheel of Destiny, I want to spend one
  charge to revive on the spot, so that I do not lose my position in the run.
- As a player who died without a wheel, I want the normal return-map respawn to be unchanged.
- As a player standing next to someone who revives with a wheel, I want to see the tomb /
  upgrade effect play over them, so that the revive reads as an event rather than a teleport.
- As a player holding a Safety Charm, I want to be told how many uses remain after a death
  consumes one, so that I know when to buy more.
- As a player whose wheel has charges left, I want the item to stay in my inventory with a
  decremented count rather than vanishing after one use.
- As a maintainer, I want both packets verified against the client read order on every version
  that has them, so that the coverage matrix reflects reality rather than a prose claim.

## 4. Functional Requirements

### 4.1 `USE_DEATHITEM` serverbound

- **FR-1.1** A `USE_DEATHITEM` codec is added under `libs/atlas-packet/` with both `Decode` and
  `Encode`, following the immutable-struct + accessor conventions of the surrounding package.
- **FR-1.2** The field order is derived from the client (`CUserLocal::RequestUpgradeTombEffect`)
  per [`docs/packets/IMPLEMENTING_A_PACKET.md`](../../packets/IMPLEMENTING_A_PACKET.md) — from
  the GMS v95.1 IDB, cross-checked against v83. **The layout is not assumed by this PRD.** No
  field list appears here because none has been verified; deriving it is the design phase's
  first job.
- **FR-1.3** Version divergence, if any, is expressed with the `MajorAtLeast` idiom, never a
  raw `> N` comparison.
- **FR-1.4** A channel handler is registered for the op and bound in all six applicable seed
  templates at the version's opcode:

  | Version | Opcode |
  |---|---|
  | gms_v83 | `0x035` |
  | gms_v84 | `0x035` |
  | gms_v87 | `0x038` |
  | gms_v92 | `0x03B` |
  | gms_v95 | `0x03A` |
  | jms_v185 | `0x02D` |

  (Source: `docs/packets/audits/STATUS.md:579`. v48/v61/v72/v79 carry no opcode for this op and
  remain `n-a`.)
- **FR-1.5** Template edits obey [`docs/packets/TEMPLATE_CONVENTIONS.md`](../../packets/TEMPLATE_CONVENTIONS.md):
  the new `handlers` entry goes at its sorted `opCode` position, and it carries a validator —
  a handler with a missing validator is silently dropped at load.

### 4.2 `USE_DEATHITEM` semantics

- **FR-2.1** The handler validates that the requesting character is dead (`Hp() == 0`) and
  holds the referenced death item with at least one charge remaining. A request failing either
  check is logged and ignored — no packet response, no state change.
- **FR-2.2** On a valid request the handler drives the same in-map revive outcome as the
  implicit path: consume one charge, restore HP, remain in the current map, broadcast the tomb
  effect.
- **FR-2.3** **Single-consume guarantee.** The implicit `MAP_CHANGE`-while-dead path and the
  explicit `USE_DEATHITEM` path must never both consume a charge for one death. The mechanism
  is a design decision (candidate: a short-lived per-character "revive already in flight" guard
  keyed on the death, or making the implicit path defer to a recently-received `USE_DEATHITEM`).
  Whichever is chosen must be covered by a test that fires both paths for one death and asserts
  exactly one charge consumed.
- **FR-2.4** The *observed* client behaviour — under what circumstances the v83+ client actually
  emits `USE_DEATHITEM` versus relying on `MAP_CHANGE` — is an open question (see §9, OQ-1) to
  be settled by live testing. Both paths are supported precisely because the trigger condition
  is unverified.
- **FR-2.5** No behaviour change on v48/v61/v72/v79, where the opcode does not exist: the
  implicit path remains the only trigger.

### 4.3 `SHOW_UPGRADE_TOMB_EFFECT` clientbound

- **FR-3.1** A `SHOW_UPGRADE_TOMB_EFFECT` codec is added under `libs/atlas-packet/` with both
  `Encode` and `Decode`, layout derived from `CUserRemote::OnShowUpgradeTombEffect`. As with
  FR-1.2, the field list is not asserted here.
- **FR-3.2** A writer is registered in all eight applicable seed templates at the version's
  opcode:

  | Version | Opcode |
  |---|---|
  | gms_v72 | `0x0B1` |
  | gms_v79 | `0x0B5` |
  | gms_v83 | `0x0C3` |
  | gms_v84 | `0x0C7` |
  | gms_v87 | `0x0D0` |
  | gms_v92 | `0x0DF` |
  | gms_v95 | `0x0DD` |
  | jms_v185 | `0x0C9` |

  (Source: `docs/packets/audits/STATUS.md:299`. v48/v61 are `n-a`.)
- **FR-3.3** Writer entries carry an `fname` and appear at their sorted `opCode` position; the
  seed corpus count expectation is updated in lockstep.
- **FR-3.4** The effect is broadcast to **other** sessions in the map (the op is a
  `CUserRemote` handler — it describes a remote character), using the existing
  `channelmap.ForOtherSessionsInMap` primitive. Whether the reviving player also receives a
  local variant is determined by what the client's own-character path does; if it does not,
  none is sent.
- **FR-3.5** The broadcast fires from the shared revive outcome (§4.2 FR-2.2), so it plays on
  both the implicit and explicit trigger paths, and on v72/v79 where only the implicit path
  exists.

### 4.4 Charges

- **FR-4.1** A death item's remaining uses are modelled as the asset's quantity
  (`asset.Model.Quantity()`, `services/atlas-channel/atlas.com/channel/asset/model.go:164`).
  Consuming a use decrements quantity by one; the asset is destroyed only when the last use is
  spent.
- **FR-4.2** The implementation uses the existing `saga.DestroyAsset` step with `Quantity: 1,
  RemoveAll: false` — the step already expresses a partial decrement, so no new saga action is
  required. The current wheel step in `createRespawnSaga` already passes `Quantity: 1`; what
  changes is that the remaining count is read, reported, and gates whether the item is usable.
- **FR-4.3** An item at quantity 0 (or absent) is not a usable death item: the wheel does not
  redirect the respawn target and `USE_DEATHITEM` is rejected per FR-2.1.
- **FR-4.4** Whether the Wheel of Destiny and the Safety Charm carry their use count in
  inventory quantity or in a WZ-declared per-item charge field must be verified against WZ data
  before implementation (see §9, OQ-2). No charge count is asserted in this document. If WZ
  declares a per-item count that differs from quantity, FR-4.1 is revised in design rather than
  guessed at here.

### 4.5 Protect-on-die effect

- **FR-5.1** When `findProtectiveItem` selects an item and the death consumes a use of it, the
  channel emits `CharacterProtectOnDieItemUseEffectBody` to the dying character and
  `CharacterProtectOnDieItemUseEffectForeignBody` to other sessions in the map
  (`libs/atlas-packet/character/effect_body.go:135,141`).
- **FR-5.2** `safetyCharm` is `true` for `item.SafetyCharmId` and `false` for
  `EasterBasketId` / `ProtectOnDeathId`; the codec omits `itemId` from the wire when
  `safetyCharm` is true (`effect.go:404`), so the id is only meaningful in the non-charm case.
- **FR-5.3** `usesRemaining` is the post-decrement charge count from §4.4.
- **FR-5.4** `days` — the field's meaning (expiry days remaining, presumably) is unverified;
  its source value is an open question (§9, OQ-3). It is not to be filled with a plausible
  constant.
- **FR-5.5** Protective items are *also* subject to charge accounting: consuming a use of a
  charm decrements rather than destroying, matching FR-4.1. This is a behaviour change — today
  `createRespawnSaga` destroys one unit unconditionally.

### 4.6 Coverage matrix

- **FR-6.1** All 14 cells (6 × `USE_DEATHITEM`, 8 × `SHOW_UPGRADE_TOMB_EFFECT`) are promoted to
  ✅ via the single-cell verify procedure —
  [`docs/packets/audits/VERIFYING_A_PACKET.md`](../../packets/audits/VERIFYING_A_PACKET.md),
  driven by the `packet-verifier` agent — with byte fixtures carrying a `packet-audit:verify`
  marker and pinned evidence records.
- **FR-6.2** `n-a` cells (v48/v61 for both ops; v72/v79 for `USE_DEATHITEM`) are left `n-a` and
  pass the `n-a` consistency gate.
- **FR-6.3** `STATUS.md` / `status.json` are regenerated, not hand-edited.

## 5. API Surface

No REST surface changes are anticipated. The task is protocol- and channel-internal:

- **Socket, serverbound (new):** `USE_DEATHITEM` — layout TBD per FR-1.2.
- **Socket, clientbound (new):** `SHOW_UPGRADE_TOMB_EFFECT` — layout TBD per FR-3.1.
- **Socket, clientbound (existing codec, newly emitted):** `CHARACTER_EFFECT` /
  `EffectProtectOnDie` + `EffectProtectOnDieForeign`.
- **Saga (existing actions, no new action types):** `DestroyAsset`, `SetHP`,
  `DeductExperience`, `CancelAllBuffs`, `WarpToPortal` under `saga.CharacterRespawn`.

If the design phase concludes that charge accounting cannot be read from the inventory the
channel already fetches (`channelInventory.GetByCharacterId`), any new read is expected to use
an existing atlas-assets/atlas-compartments endpoint rather than introduce one. Introducing a
new endpoint is a design-phase escalation, not an assumption.

## 6. Data Model

No new persisted entities. No migrations.

Charge state lives on the existing asset row as quantity, mutated through the existing
`DestroyAsset` saga step (FR-4.2). Item ids are already declared in
`libs/atlas-constants/item/death_protection.go` (`WheelOfFortuneId` 5510000, `SafetyCharmId`
5130000, `EasterBasketId` 4031283, `ProtectOnDeathId` 4140903) together with the
`IsDeathProtectionItem` / `IsWheelOfFortune` / `IsSafetyCharm` predicates. Per DOM-21, reuse
these rather than redeclaring; any new predicate this task needs belongs in that same file.

## 7. Service Impact

| Component | Change |
|---|---|
| `libs/atlas-packet` | Two new codecs (`USE_DEATHITEM` serverbound, `SHOW_UPGRADE_TOMB_EFFECT` clientbound) with `Encode`+`Decode` and byte-fixture tests per version. |
| `libs/atlas-constants` | Possible additional death-item predicate/id in `item/death_protection.go`. No new package. |
| `services/atlas-channel` | New `USE_DEATHITEM` handler; new tomb-effect writer registration in `main.go`; `respawn/processor.go` reworked for charges, the shared revive outcome, the single-consume guard, and the protect-on-die effect emission. |
| `services/atlas-configurations` | Seed templates: 6 handler entries + 8 writer entries across `template_gms_{72,79,83,84,87,92,95}_1.json` and the JMS185 template, at sorted opcode positions with validators / `fname`s. |
| `docs/packets` | Registry entries, evidence records, regenerated `STATUS.md` / `status.json`. |
| `atlas-ui` | None. |

## 8. Non-Functional Requirements

- **Multi-tenancy.** Opcodes and template bindings are per-tenant configuration; the handler and
  writer resolve their wire values from tenant config (DOM-25) and never hard-code an opcode.
  Version-conditional field logic uses `MajorAtLeast`.
- **No regression on verified versions.** No wire change may alter an already-✅ cell of any
  other op. In particular the `CHARACTER_EFFECT` writer's existing verified cells must be
  unaffected by newly emitting the protect-on-die body.
- **Idempotency.** The revive path runs under a saga; a redelivered Kafka message or a
  duplicated client request must not consume a second charge (FR-2.3 and the saga's own
  transaction id).
- **Observability.** Each consume logs character id, item id, pre- and post-charge count, and
  which trigger path fired — enough to settle OQ-1 from logs during live testing.
- **Failure handling.** A failed charge decrement aborts the revive rather than granting a free
  in-map respawn; a failed tomb-effect broadcast is logged and does not abort the revive.

## 9. Open Questions

- **OQ-1 — When does the client send `USE_DEATHITEM`?** Unverified. The revive today is driven
  entirely by `MAP_CHANGE` with `Hp() == 0`, and it works. Whether `USE_DEATHITEM` is the death
  dialog's wheel button, a separate cosmetic "play the tomb effect" request, or version-dependent
  is to be settled by live packet capture on v83. Both paths are supported (FR-2.4) precisely
  because this is unknown; the answer may let one path be simplified later.
- **OQ-2 — Where do charges live?** Inventory quantity versus a WZ-declared per-item charge
  count, for both `5510000` and `5130000`. Must be checked against WZ data (`Item.wz/Cash/`)
  before FR-4.1 is implemented. No WZ tree is available in this checkout; the check runs against
  `atlas-data` in a live environment.
- **OQ-3 — What feeds `EffectProtectOnDie.days`?** The field exists in the codec with no
  emitter and no documented meaning. Likely expiry days remaining on the cash item; unverified.
- **OQ-4 — Does the reviving player get a local tomb effect?** FR-3.4 assumes remote-only
  because the fname is `CUserRemote::OnShowUpgradeTombEffect`; confirm from the client whether a
  local counterpart is expected.
- **OQ-5 — Does the wheel's in-map revive respect map field limits?** The current implementation
  redirects to the current map whenever a wheel is held, with no field-limit check, while
  `mapData` already exposes limits used for `NoExpLossOnDeath`. Whether a "no wheel here" field
  limit exists and should gate the redirect is unverified.

None of these block starting the design phase; OQ-1 and OQ-2 must be answered before the
respective functional requirements are implemented.

## 10. Acceptance Criteria

- [ ] `USE_DEATHITEM` codec exists in `libs/atlas-packet` with `Encode` and `Decode`, derived
      from the client read order, with per-version byte-fixture tests.
- [ ] `SHOW_UPGRADE_TOMB_EFFECT` codec exists in `libs/atlas-packet` with `Encode` and `Decode`,
      derived from the client read order, with per-version byte-fixture tests.
- [ ] `USE_DEATHITEM` handler registered in the six v83+ templates at the opcodes in FR-1.4,
      each with a validator.
- [ ] `SHOW_UPGRADE_TOMB_EFFECT` writer registered in the eight v72+ templates at the opcodes in
      FR-3.2, each with an `fname`.
- [ ] `tools/template-opcode-order-guard.sh`, `tools/template-duplicate-binding-guard.sh`, and
      `tools/template-movement-types-guard.sh` clean.
- [ ] Dying with a wheel revives in the current map on both trigger paths, and a test proves a
      single death consumes exactly one charge even when both paths fire.
- [ ] Dying without a usable wheel (absent, or zero charges) still respawns at
      `mapData.ReturnMapId()` — regression test.
- [ ] A death item with more than one charge is decremented, not destroyed; the last charge
      destroys the asset.
- [ ] Consuming a protective item emits `EffectProtectOnDie` to the owner and
      `EffectProtectOnDieForeign` to other sessions in the map, with the post-decrement
      `usesRemaining`.
- [ ] Other players in the map receive `SHOW_UPGRADE_TOMB_EFFECT` on an in-map wheel revive,
      including on v72/v79.
- [ ] All 14 matrix cells promoted to ✅ with pinned evidence; `n-a` cells unchanged and passing
      the consistency gate; `STATUS.md`/`status.json` regenerated.
- [ ] No previously-✅ cell of any other op regressed.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in every changed module;
      `docker buildx bake` clean for every service whose `go.mod` was touched.
- [ ] `tools/lint.sh --check`, `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, and
      `tools/skill-job-id-guard.sh` clean.
- [ ] OQ-1 through OQ-5 are each either answered with cited evidence in `design.md`, or
      explicitly recorded as still-open with the reason.
