# bug — Maple Life (B-Type) is rejected by the slot-limit gate instead of adding a slot

Task: task-246-maple-life-character-creation
PR: atlas-pr-1466
Branch: task-246-maple-life-character-creation
Reported: 2026-08-27

## Reproduced

Reported from live play, not re-run locally. Tenant
`947e7bf0-8835-4c42-8b0a-3e052cecdc45`, region GMS, ms.version 83.1, world 0,
account 1, 4 characters in world 0.

Player used a Maple Life (B-Type) coupon for character creation.

## Observed

`atlas-channel` logged and answered `MAPLELIFE_ERROR/UNKNOWN_ERROR`:

```
{"log.level":"warning","service.name":"atlas-channel","ms.version":"83.1","region":"GMS",
 "world.id":0,"channel.id":0,"tenant":"947e7bf0-8835-4c42-8b0a-3e052cecdc45",
 "message":"Account [1] attempted Maple Life creation in world [0] with [4] of [4] slots already used."}
```

Source: `services/atlas-channel/atlas.com/channel/socket/handler/maple_life_create.go:149-152`
("Gate 3 — slot limit").

## Expected

The two Maple Life coupons are NOT the same product. `String.wz` `Cash.img`
(read from the local Cosmic corpus, `String.wz/Cash.img.xml:1582-1589`) is
explicit:

- `5431000` — **Maple Life (A-Type)**:
  `"*Warning: If you do not have an empty Character slot, it cannot be used."`
  → the current gate is CORRECT for this id.
- `5432000` — **Maple Life (B-Type)**:
  `"*Warning : If all 12 character slots are full, it cannot be used."` and
  `"This package has the Maple Life item plus an Extra Character Slot coupon.
  Using this item will automatically create a new slot."`
  → B-Type must ADD a character slot (hard cap 12), not be rejected at the
  current slot count.

(For completeness, `5430000` — "Extra Character Slot Coupon", desc
`"Increases the character creation slot by 1"` — is a third, separate id. It
does not reach this handler: `get_cashslot_item_type`'s 543 branch routes
`itemId/1000-5431 > 1` to cash-slot type 57/58, and only `5431xxx`/`5432xxx`
to 65/66 — see `character_cash_item_use.go:1600-1613` and
`derivation.md §1.1-§1.4`. It is out of scope for this bug.)

## Root cause

Two defects, one nested inside the other.

### 1. The gate does not distinguish the two coupon ids (task-246's own defect)

`handleMapleLifeCreate` is reached for BOTH `5431000` and `5432000` — the arm
that routes into it keys on `item.GetClassification(itemId) ==
item.ClassificationCharacterCreation` (`character_cash_item_use.go:891`),
which is classification 543 and therefore covers both ids. `itemId` is passed
into the handler (`maple_life_create.go:103`) and is used for the ownership
re-check, but the slot gate at `:149` applies A-Type semantics
unconditionally. PRD FR-4.4 ("Creation MUST respect the account's
character-slot limit") never drew the A/B distinction, so no phase caught it.

### 2. There is no character-slot persistence anywhere in the platform (pre-existing, outside task-246)

Verified by sweep, not spot-check:

- `services/atlas-account/atlas.com/account/account/entity.go:15-28` — the
  `accounts` GORM entity has **no** character-slots column at all.
- `services/atlas-account/atlas.com/account/account/rest.go:74` —
  `Transform` hardcodes `CharacterSlots: 4` for every account, every tenant.
- `services/atlas-account/.../account/model.go` has no slots field; the only
  two matches for "slots" in the whole service are the two lines above.
- Consumers read that hardcoded 4: `atlas-login` account model/rest
  (`login/account/rest.go:20,51`), which feeds the character-list wire value
  (`login/socket/handler/character_list_world.go:55`,
  `a.CharacterSlots()`), and `atlas-channel` account model
  (`channel/account/rest.go:19,99`, `channel/account/model.go:40`) — the
  value this bug's gate reads.
- The existing slot-increase path is a **stub**:
  `services/atlas-cashshop/.../kafka/consumer/cashshop/consumer.go:118-125`,
  `handleCommandRequestCharacterSlotIncreaseByItem`, unconditionally emits
  `ErrorStatusEventProvider(..., "UNKNOWN_ERROR", ...)`. Its producer side
  (`channel/cashshop/producer.go:127-135`,
  `channel/cashshop/processor.go:93-96`) exists and is wired, but nothing on
  the far side ever increments anything.

So the B-Type behaviour cannot be made correct by editing the channel gate
alone: "add a slot" has nowhere to be written, and the added slot would not
survive to the login character list. A channel-only change that merely
*permits* creation up to 12 would let an account hold more characters than
the 4 slots every other service still reports.

## Ruling (user, 2026-08-27)

Both open scope questions were put to the user and answered:

1. **Scope B — full persistence.** Implement the A/B distinction in the
   channel gate AND real character-slot persistence, so B-Type genuinely adds
   a slot and the login character list reflects it. Landing inside PR-1466 is
   accepted.
2. **Cap is 12, per world.** The slot value becomes per-(account, world),
   matching the gate's existing per-world character count
   (`charactersInWorldFunc`). The `5430000` string's "6 per server world" is
   NOT the cap; 12 per world is. Do not re-litigate this from WZ.

Open question 4 (does the client pre-gate B-Type at 12 before sending) is
NOT a blocker: the server gate is authoritative either way. Do not spend an
IDA pass on it.

## Fix

### F1 — per-world character-slot persistence in `atlas-account`

The `accounts` entity stays as it is; slots are per-(tenant, account, world),
so they need their own table rather than a column on `accounts`.

- `services/atlas-account/atlas.com/account/account/entity.go` — add a new
  entity for per-world slots (tenant id, account id, world id, slots), and
  register it in `Migration` at `:10` (AutoMigrate is this service's only
  migration mechanism). Default 4, matching today's hardcoded value.
- `services/atlas-account/atlas.com/account/account/rest.go` — `Transform`
  at `:74` currently hardcodes `CharacterSlots: 4`. Decide deliberately
  whether the flat `RestModel.CharacterSlots` field survives: it is
  account-scoped and the value is now world-scoped, so a world-scoped
  sub-resource is the honest shape. Whatever is chosen, no consumer may be
  left silently reading a stale flat 4.
- `services/atlas-account/atlas.com/account/account/model.go`, `builder.go`,
  `provider.go`, `processor.go` — read + increment for one (account, world),
  with the 12 cap enforced in the processor, not only at the caller. An
  increment past 12 is an error, not a clamp.
- `services/atlas-account/atlas.com/account/account/resource.go` — the
  routes are registered at `:32-40`; add the world-scoped slot read and the
  increment. Follow the existing `rest.RegisterHandler` / `ParseAccountId`
  idiom exactly.
- Tests: this package already has `processor_test.go`,
  `database_layer_test.go`, `resource_test.go`, `rest_test.go`,
  `builder_test.go` — extend them, do not add a new parallel harness. Use
  the Builder pattern for setup (CLAUDE.md); no `*_testhelpers.go`.

### F2 — `atlas-login` reads the per-world value

- `services/atlas-login/atlas.com/login/account/rest.go:20,51`,
  `model.go:18,37-38,73,165-167,188,209` — carry the world-scoped value.
- `services/atlas-login/atlas.com/login/socket/handler/character_list_world.go:55`
  — `a.CharacterSlots()` must become the (account, world) value. The world
  id is already in hand at that point (`p.WorldId()` / `w.Id()`, `:25`,
  `:49`), so no new plumbing is needed to know which world.

### F3 — `atlas-channel` reads and increments

- `services/atlas-channel/atlas.com/channel/account/rest.go:19,99`,
  `model.go:17,40-41,66,138-139,158` — carry the world-scoped value.
- `services/atlas-channel/atlas.com/channel/account/` — add the increment
  call against F1's new endpoint.
- The existing cashshop stub
  (`services/atlas-cashshop/.../kafka/consumer/cashshop/consumer.go:118-125`,
  `handleCommandRequestCharacterSlotIncreaseByItem`) is NOT in scope. Do not
  route Maple Life through it and do not implement it; the channel calls
  atlas-account directly. Leave the stub exactly as it is.

### F4 — the channel gate itself

- `services/atlas-channel/atlas.com/channel/socket/handler/maple_life_create.go`
  — gate 3 (`:138-152`) branches on `itemId`:
  - `5431000` (A-Type): unchanged — reject when `len(chars) >= int(slots)`.
  - `5432000` (B-Type): reject only when the world is at the 12 cap;
    otherwise increment the account's slots for this world by 1, then
    proceed.
  - Any other id reaching this handler is a routing defect — fail closed with
    a distinct log line rather than picking a default.
  - Check `libs/atlas-constants/` before defining the item ids or the 12 cap
    (CLAUDE.md: check it before defining any new domain type, alias, or
    numeric constant). `item.Id(5431000)` is currently a bare literal in the
    test at `maple_life_create_test.go:38`; named constants belong in the
    constants lib, not in the handler.
- **Ordering and rollback.** The increment must not leak a slot when the
  factory call at `:204` fails. Prefer incrementing only after
  `createMapleLifeFunc` succeeds, if the 12-cap check can be made on the
  pre-increment count — that is the ordering with no compensation to write.
  If the increment must precede the seed, it needs an explicit rollback on
  the factory-error path at `:205-212`, and the rollback must be tested.
  State which ordering was chosen and why in the report.
- `services/atlas-channel/atlas.com/channel/socket/handler/maple_life_create_test.go`
  — the existing slot-gate test pins `mapleLifeCreateTestItemId =
  item.Id(5431000)` (`:38`). Add B-Type coverage: below cap (creates, slot
  incremented), at the 12 cap (rejected), and the failure-path case for
  whichever ordering F4 chose.

### Verification scope

This crosses three services and a REST contract. Module-local
`go build`/`go test` for `atlas-account`, `atlas-login`, `atlas-channel` is
the implementer's bar; the repo-wide gate runs separately.

## Not yet answered

- The exact REST shape of the world-scoped slot resource (sub-resource path
  vs. keeping the flat field) is left to the implementer, but it must not
  leave any consumer reading a stale account-scoped 4. Record the choice in
  the report.

## Resolution

(unresolved — implementation dispatched 2026-08-27)
