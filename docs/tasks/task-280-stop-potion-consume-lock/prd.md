# STOP_PORTION Consume Lock Enforcement — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-28
---

## 1. Overview

Mob skill 134 ("Crazy Skull", potion lock) applies the `STOP_PORTION` character
temporary stat. The application half of this feature is already complete and
shipped: `services/atlas-monsters/atlas.com/monsters/monster/disease.go:24-36`
routes `monster2.SkillTypeStopPotion` through `debuffWireValue` so the WZ `x`
attribute passes to the client unchanged;
`libs/atlas-constants/character/temporary_stat.go:81` defines
`TemporaryStatTypeStopPortion`; and `atlas-buffs` classifies it as a disease in
`services/atlas-buffs/atlas.com/buffs/character/immunity.go:10`, so it
participates in immunity, dispel, and heal-dispel like every other debuff.

What is missing is enforcement. `atlas-consumables` contains no reference to
`STOP_PORTION` anywhere (verified: repository-wide grep for
`STOP_PORTION|StopPortion|STOP_POTION|StopPotion` returns zero hits under
`services/atlas-consumables/`). `RequestItemConsume`
(`services/atlas-consumables/atlas.com/consumables/consumable/processor.go:300`)
reserves and consumes the item with no debuff check at all. A stock client
suppresses the use locally while the stat is active, so ordinary play is
unaffected — but a modified client that simply emits `REQUEST_ITEM_CONSUME`
anyway is served normally, and the entire mechanic is a no-op against exactly
the population it exists to constrain. This is a server-authority gap, not a
gameplay feature gap.

The fix has a direct in-repo precedent. Task-256 (zombify) established the
pattern of reading a character's live buffs from inside the consume flow:
`resolveZombified` (`consumable/processor.go:176-183`) calls
`buff.Processor.GetByCharacterId` and folds the result into the effect plan,
degrading to "the debuff has no effect" when `atlas-buffs` is unreachable. This
task reuses the same client and the same fail-open posture, but the decision is
made earlier (before the reservation, not during effect application) and the
outcome is rejection rather than modification.

## 2. Goals

Primary goals:

- Make `STOP_PORTION` server-authoritative: while the stat is active, the server
  refuses to consume USE-inventory consumables regardless of what the client sends.
- Cover the pet auto-pot path, which funnels into the same command
  (`services/atlas-channel/atlas.com/channel/socket/handler/pet_item_use.go:162`
  calls `consumable.Processor.RequestItemConsume`), so a single gate at the
  `atlas-consumables` entrypoint covers both request shapes.
- Leave the client unstuck on rejection: the reservation is cancelled and a
  distinct error type reaches `atlas-channel`, which re-enables actions.
- Preserve current behaviour exactly when the stat is not active, and when
  `atlas-buffs` is unreachable.

Non-goals:

- Changing how `STOP_PORTION` is applied, timed, cancelled, or dispelled
  (owned by `atlas-monsters`, `atlas-buffs`; see task-190 for duration/cancel).
- Enforcing the sibling `STOP_MOTION` stat, or any other unenforced debuff.
- Any change to `libs/atlas-packet` — this task introduces no new wire packet.
  The new error type travels on an existing Kafka event body field.
- Client-side changes of any kind.
- Gating the cash-shop item-use path (`consumables/cash/`).
- Gating equip-scroll, Vicious Hammer, Vega, skill book, or catch-monster
  requests — these are separate entrypoints and are not consumables.

## 3. User Stories

- As a player fighting Crazy Skull, I want the potion lock to actually matter,
  so that the boss's signature mechanic is not trivially defeated by other
  players running modified clients.
- As a server operator, I want potion-lock enforcement decided on the server,
  so that client integrity is not a prerequisite for the mechanic working.
- As a player using a legitimate client, I want no observable change: my potion
  hotkey behaves identically when I am not potion-locked, and the client is
  never left in a stuck state if the server does reject a use.
- As a player with an auto-pot pet, I want the pet's auto-consume to respect the
  potion lock, so the pet is not an enforcement bypass.
- As an operator during an `atlas-buffs` outage, I want potions to keep working,
  so that a dependency failure degrades to "the debuff has no effect" rather
  than "nobody can heal".

## 4. Functional Requirements

### FR-1 — Gate placement

The check MUST be performed in `atlas-consumables` inside
`ProcessorImpl.RequestItemConsume`
(`services/atlas-consumables/atlas.com/consumables/consumable/processor.go:300`),
before `p.cpp.RequestReserve` is called and before the one-time handler is
registered. Rejecting before reservation means no reservation exists to cancel
in the common rejection case.

The check MUST NOT be duplicated in `atlas-channel`. The channel's pet auto-pot
handler reaches `RequestItemConsume` through the same command
(`pet_item_use.go:162`), so a single gate at the `atlas-consumables` entrypoint
covers both the direct `REQUEST_ITEM_CONSUME` path and the pet auto-pot path.

### FR-2 — Item scope

The gate applies to items for which `usesStandardConsumer(itemId)` returns true
(`consumable/processor.go:116-124`) — item classifications 200, 201, 202, 205,
and `ClassificationConsumableTransformation`.

For every other branch of `RequestItemConsume`'s classification dispatch —
town-warp scrolls, pet food, cash pet food, summoning sacks, pet skill pouches,
monster cards, morph coupons, reward-table boxes, and the `ConsumeBare`
fallback — behaviour is unchanged: the request proceeds as it does today.

The gate MUST be evaluated only for in-scope items. An out-of-scope item MUST
NOT trigger a buffs read; the gate must not add a network call to paths it does
not govern.

### FR-3 — Lock predicate

A new exported helper in
`services/atlas-consumables/atlas.com/consumables/character/buff/model.go`
reports whether a `[]buff.Model` contains an unexpired buff carrying a stat
change of type `STOP_PORTION`. It MUST mirror the shape and expiry semantics of
the existing `IsZombified` helper (`character/buff/model.go:63-...`) — same
unexpired-only filter, same iteration over `Changes()`.

The stat type string MUST come from
`libs/atlas-constants/character/temporary_stat.go`
(`ts.TemporaryStatTypeStopPortion`), never a string literal. Note the existing
spelling is `STOP_PORTION` (with the R); the codebase's constant is
authoritative and MUST NOT be renamed as part of this task.

The magnitude of the stat (the WZ `x` wire value) is NOT consulted. Presence of
an unexpired `STOP_PORTION` change is the entire predicate.

### FR-4 — Fail-open on dependency failure

If the buffs read returns an error, the gate MUST log at Warn (including the
character id and the underlying error) and allow the consume to proceed. This
matches `resolveZombified`'s documented posture (`processor.go:176-183`): an
`atlas-buffs` outage degrades to "the debuff has no effect", never to a blanket
consume denial.

A read that succeeds and returns no buffs is not a failure — it resolves to
"not locked" and the consume proceeds.

### FR-5 — Rejection outcome

When the gate resolves to locked, `RequestItemConsume` MUST:

1. Not register the one-time consume handler and not call `RequestReserve`.
2. Emit an `ERROR` event on `EVENT_TOPIC_CONSUMABLE_STATUS` carrying the new
   error type from FR-6, for the requesting character.
3. Return a sentinel error (e.g. `ErrPotionLocked`, declared alongside
   `ErrPetCannotConsume` / `ErrPetCannotLearn`) so callers and tests can
   distinguish this rejection from a generic failure.
4. Log at Debug or Info — a locked consume is an expected in-game condition on a
   legitimate client racing the debuff, not an error. It MUST NOT log at Error.

Because no reservation was created, the rejection path MUST NOT call
`CancelItemReservation`. If the design places the gate such that a reservation
could already exist, the reservation MUST be cancelled — but FR-1's ordering is
chosen specifically to avoid that case.

### FR-6 — New error type

Add `ErrorTypePotionLocked = "POTION_LOCKED"` to
`services/atlas-consumables/atlas.com/consumables/kafka/message/consumable/kafka.go`
(alongside `ErrorTypeConsumeFailed` etc. at lines 118-122) and mirror the same
constant in
`services/atlas-channel/atlas.com/channel/kafka/message/consumable/kafka.go`
(which currently mirrors only `ErrorTypePetCannotConsume`, line 90).

`consumeErrorType` (`consumable/processor.go:442-458`) MUST map
`ErrPotionLocked` to `ErrorTypePotionLocked`, so the sentinel and the wire value
stay in sync.

### FR-7 — Channel-side handling

`services/atlas-channel/atlas.com/channel/kafka/consumer/consumable/consumer.go`
MUST recognise `POTION_LOCKED` alongside the existing
`ErrorTypePetCannotConsume` branch (line 105) and unstick the client — the same
enable-actions/stat-reset response the pet branch performs. The existing
fall-through comment at line 143 ("`ErrorTypeConsumeFailed` … has no
action") documents why an unrecognised type leaves the client stuck; this task
must not add a new type that lands in that fall-through.

No new client-visible message is required. Unsticking is sufficient; a legitimate
client never reaches this path.

### FR-8 — No behaviour change when unlocked

For a character with no active `STOP_PORTION`, `RequestItemConsume` MUST behave
byte-for-byte as it does today for every classification branch, including the
transaction id, reservation timeout (30s), and the reservation payload.

## 5. API Surface

No new or modified REST endpoints.

`atlas-consumables` consumes an existing `atlas-buffs` read endpoint:
`GET {BUFFS}/characters/{characterId}/buffs`, already wired via
`services/atlas-consumables/atlas.com/consumables/character/buff/requests.go`
and drained through `requests.DrainProvider`. No change to that client is
required beyond the new predicate helper.

Kafka contract change (additive, backward compatible):

- Topic: `EVENT_TOPIC_CONSUMABLE_STATUS`
- Event: `ERROR`
- Field: `body.error`
- New permitted value: `"POTION_LOCKED"`

Producers older than this change never emit the value; consumers older than this
change fall through to their existing unrecognised-type branch. No schema
version bump is required.

## 6. Data Model

No new entities, no persisted fields, no database migration. The gate reads
transient buff state over REST and makes an in-memory decision.

Multi-tenancy is inherited: the buffs client resolves its root URL from the
request context via `requests.RootUrlFor(ctx, "BUFFS")`, and the tenant is
carried on the context that already flows into `RequestItemConsume`.

## 7. Service Impact

| Service | Change |
|---|---|
| `atlas-consumables` | New lock predicate in `character/buff/model.go`; new resolver + gate in `consumable/processor.go` (`RequestItemConsume`); new `ErrPotionLocked` sentinel; `consumeErrorType` mapping; `ErrorTypePotionLocked` constant in `kafka/message/consumable/kafka.go`. |
| `atlas-channel` | Mirror `ErrorTypePotionLocked` in `kafka/message/consumable/kafka.go`; handle it in `kafka/consumer/consumable/consumer.go` to unstick the client. No change to `socket/handler/pet_item_use.go` — it already routes through the gated command. |
| `atlas-buffs` | No change. It already tracks and exposes `STOP_PORTION`. |
| `atlas-monsters` | No change. Application already correct (`monster/disease.go:24-36`). |
| `libs/atlas-constants` | No change. `TemporaryStatTypeStopPortion` already exists. |
| `libs/atlas-packet` | No change. |
| `atlas-ui` | No change. |

## 8. Non-Functional Requirements

- **Latency**: the gate adds at most one paginated REST read to `atlas-buffs`
  per in-scope consume request. This is the same call `resolveZombified` already
  makes later in the same flow for standard consumables. The design SHOULD avoid
  issuing two separate reads for the same request where both the lock gate and
  the zombify resolver run; if a single read cannot be shared cleanly, the
  duplication is acceptable and must be called out explicitly in `design.md`.
- **Availability**: `atlas-buffs` unavailability MUST NOT block potion use
  (FR-4). No retry loop, no blocking backoff on this path.
- **Security / authority**: the gate is the anti-cheat boundary. It MUST NOT be
  bypassable by any field the client controls (item id, slot, quantity, pet id,
  update time).
- **Observability**: rejections are logged with character id and item id at
  Debug/Info; buffs read failures at Warn with the error attached. Existing
  `degrade.Observe` conventions in `atlas-channel` are not required here since
  the failure is handled in `atlas-consumables`.
- **Multi-tenancy**: all reads and produced events remain tenant-scoped via the
  existing context; no tenant identifier is introduced or hard-coded.
- **Testing**: the predicate and the gate decision MUST be unit-testable without
  a live `atlas-buffs`, following the `resolveZombified` precedent of taking the
  `buff.Processor` interface as a parameter so the fail-open branch is directly
  exercisable with the existing mock in `character/buff/mock/`.

## 9. Open Questions

- **Q1 — Reference behaviour for item scope.** No in-repo or WZ evidence was
  found stating which item classifications the original server blocks under the
  potion lock. "All USE-inventory consumables" (FR-2) is a product decision made
  during the spec interview, not a verified reference behaviour. If IDA or WZ
  evidence later contradicts it, FR-2 is the requirement to revisit.
- **Q2 — Shared buffs read.** Whether the lock gate and `resolveZombified` can
  share one `GetByCharacterId` call depends on where the character/consumable
  data fetch currently sits relative to `RequestItemConsume`. Resolve in
  `design.md`; either outcome satisfies this PRD.
- **Q3 — Client message on rejection.** FR-7 specifies unstick only. If
  playtesting shows the client needs an explicit "you cannot use that now"
  message, that is a follow-up, not part of this task.
- **Q4 — `STOP_MOTION`.** The sibling stat is equally unenforced. Explicitly out
  of scope here (§2); worth a separate task.

## 10. Acceptance Criteria

- [ ] A `[]buff.Model` containing an unexpired `STOP_PORTION` change resolves the
      new predicate to `true`; one containing only an expired `STOP_PORTION`
      change, or no `STOP_PORTION` change, resolves to `false`. Unit test.
- [ ] The stat type string is sourced from
      `libs/atlas-constants/character/temporary_stat.go`; grep for a
      `"STOP_PORTION"` string literal in `services/atlas-consumables/` returns
      zero hits outside test fixtures.
- [ ] `RequestItemConsume` with an in-scope item (classification 200/201/202/205
      or transformation) for a locked character returns `ErrPotionLocked`, does
      not call `RequestReserve`, and does not register a one-time handler. Unit
      test with a mocked `buff.Processor` and a mocked compartment processor.
- [ ] The same call emits exactly one `ERROR` event on
      `EVENT_TOPIC_CONSUMABLE_STATUS` with `body.error == "POTION_LOCKED"`.
- [ ] `RequestItemConsume` with an out-of-scope item (e.g. town-warp scroll, pet
      food, monster card) for a locked character proceeds unchanged and issues no
      buffs read. Unit test asserts the mocked `buff.Processor` was not called.
- [ ] `RequestItemConsume` with an in-scope item for an unlocked character
      proceeds exactly as before — same reservation call, same handler
      registration. Unit test.
- [ ] When the mocked `buff.Processor.GetByCharacterId` returns an error, an
      in-scope consume proceeds (fail-open) and a Warn is logged. Unit test.
- [ ] `consumeErrorType(ErrPotionLocked)` returns `ErrorTypePotionLocked`; every
      other error still returns its existing value. Unit test.
- [ ] `atlas-channel`'s consumable status consumer routes `POTION_LOCKED` to the
      unstick action and does not fall through to the no-action branch. Unit test.
- [ ] The pet auto-pot path is covered transitively: a test or a documented trace
      in the task folder shows `pet_item_use.go` reaching the gated
      `RequestItemConsume` with no second, ungated code path.
- [ ] Zombify behaviour (task-256) is unchanged: existing
      `consumable/processor_test.go` zombify cases still pass.
- [ ] Flagless `tools/verify.sh` exits 0.
- [ ] `backend-guidelines-reviewer` and `task-reviewer` both clear the change.
