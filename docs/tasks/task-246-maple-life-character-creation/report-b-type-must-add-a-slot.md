# report — bug-b-type-must-add-a-slot

Brief: `bug-b-type-must-add-a-slot.md`
Round 1 agent: task-implementer (sonnet). Reported **PARTIAL** (tool-call cap).
Round 1 could not write this file itself (tool policy); the controller
transcribed its returned report verbatim below on 2026-08-27.

## Round 1 — PARTIAL

**Status: PARTIAL.** F1, F2, and the `libs/atlas-constants` additions are
complete, committed, and pass module-local build+test. F3 and F4
(atlas-channel) are **not started**.

**Commit:** `41c935660` — "fix(task-246): per-world character-slot persistence
(atlas-account, atlas-login)", on branch
`task-246-maple-life-character-creation` in this worktree. Branch and worktree
root verified after commit.

### REST shape chosen

`GET`/`POST accounts/{accountId}/worlds/{worldId}/character-slots`, a
world-scoped JSON:API sub-resource, type `character-slots`, body
`{"worldId": <byte>, "slots": <int16>}`.

- **GET** returns the current count, defaulting to
  `character.DefaultCharacterSlotsPerWorld` (4) when no row exists — no row is
  created on a bare read.
- **POST** increments by 1, enforces the 12 cap, returns **409 Conflict** (from
  the processor's `ErrCharacterSlotCapReached`) rather than clamping.

The flat `RestModel.CharacterSlots` field was **removed** (not kept alongside)
in atlas-account and atlas-login, so nothing can silently keep reading the
stale flat 4. atlas-channel's copy is still present — that removal is round 2's
step 1.

### F4 ordering chosen — increment-after-success, no rollback (decided, NOT yet implemented)

- The 12-cap check reads current slots via `GetCharacterSlots` **before**
  calling `createMapleLifeFunc`.
- The `IncrementCharacterSlots` call happens **only after** the factory call
  succeeds — the ordering the brief prefers, because it needs no compensating
  rollback.
- Residual, documented risk: if the increment call fails after a successful
  factory create, the character exists but the slot is not persisted. Logged as
  a warning, not treated as saga-worthy.

### Completed in round 1 — build/test evidence

- `libs/atlas-constants`: added `item/maple_life.go`
  (`MapleLifeATypeId=5431000`, `MapleLifeBTypeId=5432000`) and
  `character/slots.go` (`DefaultCharacterSlotsPerWorld=4`,
  `MaxCharacterSlotsPerWorld=12`), each with tests.
  `go build ./... && go test ./item/... ./character/...` → `ok`.
- `atlas-account`: new `CharacterSlotEntity` (table `account_character_slots`,
  unique index on tenant/account/world, autoincrement ID PK matching the
  existing `Entity` style), `CharacterSlotModel`/`CharacterSlotBuilder`,
  provider read, administrator create/update, processor
  `GetCharacterSlots`/`IncrementCharacterSlots` (+`ErrCharacterSlotCapReached`),
  `CharacterSlotRestModel`, `rest.ParseAccountIdAndWorldId`, two new routes,
  409 mapping. Added `libs/atlas-constants` as a new go.mod dependency to reuse
  the shared constants rather than re-declare literals (DOM-21). Tests extended
  in `builder_test.go`, `database_layer_test.go` (migration), `processor_test.go`
  (4 new cases: default, increment+persist, per-world isolation, cap rejection
  with no partial write), `resource_test.go` (full HTTP round-trip incl. 409).
  `go build ./... && go test ./...` → all `ok`.
- `atlas-login`: removed the flat `characterSlots` field from
  `Model`/`Builder`/`RestModel` entirely; added `CharacterSlotRestModel`,
  `requestCharacterSlots`, `Processor.GetCharacterSlots(accountId, worldId world.Id)`;
  updated `MockProcessor`; `character_list_world.go` now calls the per-world
  processor method using `w.Id()`, already in scope. `builder_test.go` updated
  to drop removed assertions. `go build ./... && go vet ./... && go test ./...`
  → all `ok`.

### Round 1 self-review notes

- No TODOs/stubs/placeholders in the completed portion. Builder pattern
  followed, no `*_testhelpers.go`. Constants checked/added in
  `libs/atlas-constants` before use (DOM-21). 409 chosen for cap-reached,
  matching this service's existing `ErrAccountLoggedIn` → 409 precedent.
- One scope trim, flagged rather than silently omitted: no dedicated low-level
  `CharacterSlotEntity` CRUD test was added to `database_layer_test.go` beyond
  the one-line migration fix. The same code paths are exercised by
  `processor_test.go` and the full-HTTP `resource_test.go`. The brief named
  that file, so this is a deliberate deviation for a reviewer to rule on.

### Files touched in round 1

- `libs/atlas-constants/item/maple_life.go` (+test)
- `libs/atlas-constants/character/slots.go` (+test)
- `services/atlas-account/atlas.com/account/account/{entity,model,builder,provider,administrator,processor,resource,rest}.go`
  (+`builder_test.go`, `database_layer_test.go`, `processor_test.go`, `resource_test.go`)
- `services/atlas-account/atlas.com/account/rest/handler.go`
- `services/atlas-account/atlas.com/account/go.mod`
- `services/atlas-login/atlas.com/login/account/{model,rest,requests,processor}.go` (+`builder_test.go`)
- `services/atlas-login/atlas.com/login/account/mock/mock.go`
- `services/atlas-login/atlas.com/login/socket/handler/character_list_world.go`

## Contracts round 2 must match

Do not redesign these; round 1 already shipped the far side of each.

- atlas-account processor:
  `GetCharacterSlots(accountId uint32, worldId byte) (CharacterSlotModel, error)`,
  `IncrementCharacterSlots(...) (CharacterSlotModel, error)`.
- atlas-login (implemented) / atlas-channel (to match):
  `Processor.GetCharacterSlots(accountId uint32, worldId world.Id) (int16, error)`,
  `IncrementCharacterSlots(accountId uint32, worldId world.Id) (int16, error)`.
- Wire: `GET`/`POST accounts/{accountId}/worlds/{worldId}/character-slots` →
  `{"worldId": byte, "slots": int16}`, 409 on cap.
- Constants: `item.MapleLifeATypeId`, `item.MapleLifeBTypeId`,
  `character.MaxCharacterSlotsPerWorld`,
  `character.DefaultCharacterSlotsPerWorld` in `libs/atlas-constants`. Use
  these; do not re-literal them in atlas-channel.

## Remaining work — atlas-channel (F3, F4), file by file

1. `account/model.go` — remove `characterSlots` / `CharacterSlots()` /
   `SetCharacterSlots` (mirror atlas-login's edit).
2. `account/rest.go` — remove `RestModel.CharacterSlots` and its use in
   `Extract()`; add `CharacterSlotRestModel`.
3. `account/requests.go` — add the `CharacterSlots` route const plus
   `requestCharacterSlots` (GET) and `requestIncrementCharacterSlots` (POST,
   mirroring `requestRecordPicAttempt`'s shape; body content is ignored
   server-side).
4. `account/processor.go` — add `GetCharacterSlots` / `IncrementCharacterSlots`
   to `Processor` and `ProcessorImpl`.
5. `socket/handler/maple_life_create.go` — change the `accountSlotsFunc` seam to
   take `worldId world.Id`; add an `incrementAccountSlotsFunc` seam; import the
   constants above (no bare literals). Gate 3 becomes a switch:
   - A-Type unchanged (`len(chars) >= slots`).
   - B-Type rejects on `slots >= MaxCharacterSlotsPerWorld` — **the cap is on
     slots, not on character count** — and does *not* increment inline; the
     increment moves to just before `maplelife.GetRegistry().Put(...)` on the
     factory-success path, logged (not failed) on error.
   - Any other itemId fails closed with a distinct "routing defect" log line.
6. `maple_life_create_test.go` — update the `accountSlotsFunc` stub signature
   (+worldId), add the `incrementAccountSlotsFunc` seam/field, add
   B-Type-below-cap / B-Type-at-cap / B-Type-factory-failure-does-not-increment
   cases; switch the existing A-Type test constant to `item.MapleLifeATypeId`.
7. `socket/handler/cash_shop_entry.go` — **found by round 1's sweep, absent from
   the bug file's original inventory**: ~line 97 uses `a.CharacterSlots()` for
   the cash-shop UI. It must move to
   `account.NewProcessor(l, ctx).GetCharacterSlots(s.AccountId(), s.WorldId())`
   or the package will not compile once step 1 lands. No existing test file for
   this handler was found.
8. Verify: `cd services/atlas-channel/atlas.com/channel && go build ./... &&
   go vet ./... && go test ./...`, plus a re-run of the atlas-account and
   atlas-login module tests as a sanity check on the shared constants.

## Round 2 — DONE

Round 2 agent: task-implementer (sonnet), continuation of round 1's PARTIAL.
Completed F3 and F4 (atlas-channel), matching round 1's shipped contracts
exactly (no redesign).

### Files touched in round 2

- `services/atlas-channel/atlas.com/channel/account/model.go` — removed the
  flat `characterSlots` field / `CharacterSlots()` / `SetCharacterSlots`
  (mirrors atlas-login's round-1 edit).
- `services/atlas-channel/atlas.com/channel/account/rest.go` — removed
  `RestModel.CharacterSlots` and its use in `Extract()`; added
  `CharacterSlotRestModel` (GET response / ignored POST body), matching
  atlas-account's and atlas-login's shape byte for byte.
- `services/atlas-channel/atlas.com/channel/account/requests.go` — added the
  `CharacterSlots` route const
  (`accounts/%d/worlds/%d/character-slots`), `requestCharacterSlots` (GET),
  and `requestIncrementCharacterSlots` (POST, empty `CharacterSlotRestModel{}`
  body — atlas-account's increment route takes no input handler at all, so
  the body content is genuinely ignored server-side; an empty struct is sent
  purely so `jsonapi.Marshal` has a `MarshalIdentifier` to serialize).
- `services/atlas-channel/atlas.com/channel/account/processor.go` — added
  `GetCharacterSlots(accountId uint32, worldId world.Id) (int16, error)` and
  `IncrementCharacterSlots(accountId uint32, worldId world.Id) (int16, error)`
  to both `Processor` and `ProcessorImpl`, matching atlas-login's already-shipped
  signature exactly.
- `services/atlas-channel/atlas.com/channel/socket/handler/maple_life_create.go`
  — `accountSlotsFunc` seam now takes `worldId world.Id` and calls
  `account.NewProcessor(l, ctx).GetCharacterSlots(accountId, worldId)`; added
  the `incrementAccountSlotsFunc` seam calling
  `account.NewProcessor(l, ctx).IncrementCharacterSlots(accountId, worldId)`.
  Gate 3 is now a `switch itemId`:
  - `item.MapleLifeATypeId`: unchanged, `len(chars) >= int(slots)`.
  - `item.MapleLifeBTypeId`: rejects only when `slots >= character.MaxCharacterSlotsPerWorld`
    — **the cap check reads the SLOT count, never the character count** (see
    below for the test that proves this) — and does not increment inline.
  - default: fails closed with a distinct "routing defect" log line
    (`"...which is neither the A-Type nor B-Type coupon; routing defect."`).
  The increment call was placed immediately before
  `maplelife.GetRegistry().Put(...)`, i.e. only on the factory-success path,
  per round 1's binding ordering decision (no rollback needed). A failed
  increment is logged as a warning, not treated as saga-worthy, matching
  round 1's documented residual risk.
- `services/atlas-channel/atlas.com/channel/socket/handler/maple_life_create_test.go`
  — updated the `accountSlotsFunc` stub signature (+`worldId`), added the
  `incrementAccountSlotsFunc` seam/field with call-recording, switched
  `mapleLifeCreateTestItemId` to `item.MapleLifeATypeId`, added a
  `dispatchItem(itemId, sub)` helper (dispatch's general form, needed because
  the B-Type cases submit a different itemId than the suite default), and
  four new test functions:
  - `TestMapleLifeCreateBTypeBelowCapIncrementsSlots` — below cap, creates,
    and asserts exactly one `incrementAccountSlotsFunc` call with the
    session's accountId/worldId. Deliberately sets
    `env.charactersInWorld = 999` to prove the B-Type path never reads the
    character count.
  - `TestMapleLifeCreateBTypeAtCapIsRejected` — `accountSlots ==
    MaxCharacterSlotsPerWorld`, `charactersInWorld = 0`: rejected, zero
    factory calls, zero increment calls. Proves the cap check is on slots,
    not characters (a character-count-based gate would have let this through
    at `charactersInWorld = 0`).
  - `TestMapleLifeCreateBTypeFactoryFailureDoesNotIncrement` — factory
    returns an error; asserts `incrementSlotsCalls == 0` — the ordering
    contract's core assertion.
  - `TestMapleLifeCreateRejectsUnroutedItemId` — a third 543-classification
    id (`5433000`, which is neither A-Type nor B-Type) reaches gate 3's
    default arm and fails closed.
- `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_maple_life_test.go`
  — `installMapleLifeCreateObservationSeams`'s `accountSlotsFunc` stub also
  needed the `worldId world.Id` parameter added; found by `go vet`, not named
  in the brief's file inventory (a second, pre-existing caller of the same
  package-var seam).
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_entry.go`
  — the `a.CharacterSlots()` call at the (former) line 97 now calls
  `account.NewProcessor(l, ctx).GetCharacterSlots(s.AccountId(), s.WorldId())`;
  on error, destroys the session the same way the other account/character/
  buddylist lookups a few lines above it already do (no test file existed
  for this handler, consistent with round 1's finding — none added, since
  the brief's inventory did not ask for a new one and no existing suite
  covers this handler).

### B-Type cap-on-slots confirmation

Confirmed, and directly asserted by two tests:

- `TestMapleLifeCreateBTypeAtCapIsRejected` sets `charactersInWorld = 0`
  (nowhere near any character-count cap) and `accountSlots =
  MaxCharacterSlotsPerWorld` (12), and asserts the request is rejected with
  zero factory calls — if gate 3's B-Type arm read the character count
  instead of the slot count, this case would incorrectly proceed.
- `TestMapleLifeCreateBTypeBelowCapIncrementsSlots` sets
  `charactersInWorld = 999` (an absurd, deliberately-wrong character count)
  and `accountSlots = MaxCharacterSlotsPerWorld - 1`, and asserts the request
  proceeds — if gate 3's B-Type arm read the character count, this case would
  incorrectly reject.
- The handler source itself (`maple_life_create.go`) never calls
  `charactersInWorldFunc` inside the `item.MapleLifeBTypeId` case of the
  switch — only inside `item.MapleLifeATypeId`.

### Factory-failure-leaves-slot-count-unincremented confirmation

`TestMapleLifeCreateBTypeFactoryFailureDoesNotIncrement` sets
`e.mapleLifeErr = errors.New("unknown error")` (factory call fails) and
asserts `len(e.incrementSlotsCalls) == 0` after dispatch — the increment
call is placed strictly after the `createMapleLifeFunc` call returns
successfully in the handler source, so a factory failure returns before the
increment seam is ever reached. No rollback exists because nothing is
written before that point.

### Build/test evidence

```
$ cd services/atlas-channel/atlas.com/channel && go build ./... && go vet ./... && go test ./...
```
`go build ./...` → clean, no output.
`go vet ./...` → clean, no output.
`go test ./...` → all packages `ok`, including
`ok  	atlas-channel/account	(cached)` and
`ok  	atlas-channel/socket/handler	2.136s` (no `FAIL` anywhere in the run).

Sanity re-run of the earlier two services' module tests, unmodified by round
2, to confirm the shared constants and contracts still hold:

```
$ cd services/atlas-account/atlas.com/account && go build ./... && go test ./...
```
→ `ok atlas-account`, `ok atlas-account/account`.

```
$ cd services/atlas-login/atlas.com/login && go build ./... && go test ./...
```
→ all packages `ok`.

### Round 2 self-review notes

- No TODOs/stubs/placeholders. Constants (`item.MapleLifeATypeId`,
  `item.MapleLifeBTypeId`, `character.MaxCharacterSlotsPerWorld`) are used
  from `libs/atlas-constants`, never re-literalled in the handler (one
  narrow exception: the new routing-defect test's `item.Id(5433000)`, a
  throwaway "some other 543 id" value that has no meaning outside that one
  test case and is not a domain constant).
- `gofmt` found one alignment issue in the new struct-literal fields in
  `maple_life_create_test.go` (`accountSlotCalls`/`incrementSlotsCalls`
  additions shifted column alignment); `gofmt -w` fixed it, `gofmt -l`
  confirmed clean afterward on every file touched.
- `cash_shop_entry.go`'s new error path destroys the session on a
  `GetCharacterSlots` failure, matching the existing pattern for the three
  lookups immediately above it in the same function (account/character/
  buddylist) rather than inventing a different failure mode.
- Builder pattern already existed in `account/model.go`'s `builder`; no new
  `*_testhelpers.go` files added.
- One deviation from a literal reading of the brief: step 6 named
  `maple_life_create_test.go` as the only test file to update, but `go vet`
  surfaced a second caller of the same `accountSlotsFunc` package-var seam in
  `character_cash_item_use_maple_life_test.go` (a compile-time consequence of
  changing a shared test seam's signature, not a scope choice). Fixed inline;
  flagged here for the reviewer.

### Contracts round 2 matched (no redesign)

- `account.Processor.GetCharacterSlots(accountId uint32, worldId world.Id)
  (int16, error)` / `IncrementCharacterSlots(accountId uint32, worldId
  world.Id) (int16, error)` — signature-for-signature identical to
  atlas-login's round-1 shipped `Processor`.
- Wire: `GET`/`POST accounts/{accountId}/worlds/{worldId}/character-slots` →
  `{"worldId": byte, "slots": int16}` — unchanged from round 1's atlas-account
  route; atlas-channel is purely a client of it.
- 409 (`ErrCharacterSlotCapReached` on atlas-account's side) is never
  special-cased on the atlas-channel side: the client-side switch already
  refuses to call `IncrementCharacterSlots` past the 12 cap (its own
  pre-check reads the same slot count atlas-account would reject against),
  so the 409 path is defense-in-depth atlas-channel does not need to
  interpret specially — a generic `err != nil` on the increment call is
  logged and swallowed either way, per the agreed ordering.
