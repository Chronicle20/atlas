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

## Round 3 — atlas-character seam

Round 3 agent: task-implementer (sonnet). Fixed the review's blocking
finding #1 (`reviews/bug-b-type-must-add-a-slot.md`, item 1): the
`pending_change` package's `checkCharacterSlot` (world-transfer gate 4) was
still reading the removed flat `accounts/{accountId}` `characterSlots`
attribute, always unmarshalling to 0 and rejecting every world transfer with
`no_character_slot`.

### Fix

- `pending_change/rest.go`: removed `accountRestModel` (the stale flat
  `characterSlots` projection of `GET accounts/{accountId}`); added
  `characterSlotRestModel` (`{worldId world.Id, slots int16}` via
  `json:"worldId"`/`json:"slots"`), the projection of the new per-world
  sub-resource. Matches atlas-login's and atlas-channel's
  `CharacterSlotRestModel` shape.
- `pending_change/requests.go`: replaced `requestAccount`/`accountSlots`
  (which hit `GET accounts/{accountId}`) with `requestCharacterSlots(accountId,
  worldId)` → `GET accounts/{accountId}/worlds/{worldId}/character-slots`, and
  `accountSlots(l, ctx, accountId, worldId world.Id) (int16, error)` reading
  `rm.Slots`. Same route/response shape round 1 and round 2 already shipped
  for atlas-login and atlas-channel — no redesign.
- `pending_change/processor_eligibility.go`: `gateDeps.accountSlots` signature
  now takes `worldId world.Id`; `productionGateDeps` wiring unchanged (same
  function name, new signature). `checkCharacterSlot` now calls
  `p.gates.accountSlots(p.l, p.ctx, c.AccountId(), destinationWorldId)` —
  the same `destinationWorldId` already used for
  `character.GetForAccountInWorld()`, so the remote cap and the local count
  are read for the same world.
- `pending_change/processor_eligibility_test.go`: updated every
  `accountSlots` stub (`passingGateDeps`, the gate-4 test, the ordering
  short-circuit panic stub, the error-injection table) to the new
  4-argument signature. Added
  `TestEligibilityGate4ReadsDestinationWorldSlotsAndAllowsBelowCap`: seeds
  one existing character in `destinationWorldId`, stubs `accountSlots` to
  capture the `accountId`/`worldId` it was called with and return a cap of
  2 (room for one more), and asserts the transfer is **eligible**
  (`ok == true`, `reason == ""`) and that the stub was called with
  `c.AccountId()` and `destinationWorldId` specifically — not the account's
  home world or any other value. This is the "new behavior" assertion the
  review required: a below-cap transfer against the destination world's
  slot count is allowed, not just that the old shape still compiles.

### Sweep for other readers of the removed attribute

```
$ grep -rn "characterSlots\|CharacterSlots\|accountRestModel\|character-slots" --include="*.go" .
```
(run from `services/atlas-character/atlas.com/character`) — the only hits
after the fix are the new `characterSlotRestModel`/`requestCharacterSlots`
names and their doc comments; no other reader of the old flat attribute
exists in this module.

### Build/test evidence

```
$ cd services/atlas-character/atlas.com/character && go build ./... && go vet ./... && go test ./...
```
`go build ./...` → clean, no output. `go vet ./...` → clean, no output.
`go test ./...` → every package `ok`, including
`ok  	atlas-character/pending_change	209.865s` (no `FAIL` anywhere in the
run; exit code 0).

### Self-review notes

- No TODOs/stubs/placeholders. Followed the existing per-remote-gate
  function pattern in `requests.go` (one narrow REST client per gate) rather
  than inventing a new shape.
- `world.Id` (`type Id byte`, confirmed via `go doc`) used for the wire
  field consistent with this file's existing `world` import, matching
  atlas-login's `byte`-typed field at the JSON level (both marshal/unmarshal
  identically as a small integer).
- Did not touch atlas-ui, atlas-account, atlas-login, or atlas-channel —
  out of this unit's scope per the dispatch brief; the review's finding #2
  (atlas-ui `characterSlots` display) belongs to a separate unit.
- Committed on branch `task-246-maple-life-character-creation`, on top of
  round 2's `f32efa66d`: `62733da76` — "fix(atlas-character): read
  destination-world character-slot cap for gate 4". Branch and worktree
  root verified after commit (the assigned
  `.worktrees/task-246-maple-life-character-creation` worktree). Tree clean
  of this unit's changes after commit (only an unrelated, pre-existing
  `agent-ledger.tsv` modification and an untracked
  `reviews/task-246-maple-life-character-creation` path remained, neither
  touched by this round).

## Round 4 — lint fix + atlas-ui per-world panel

Round 4 agent: task-implementer (sonnet). Two independent units: (1) an
`errcheck` lint fix in atlas-account's F1 test, from the repo gate; (2)
atlas-ui's `characterSlots` readers, broken since the flat attribute was
removed in round 1.

### Unit 1 — errcheck fix (atlas-account)

`account/resource_test.go`'s two `getSlots`/`postIncrement` closures used
bare `defer resp.Body.Close()`, which `golangci-lint`'s `errcheck` linter
flags (unchecked error return). Grepped the repo for the existing idiom
(`defer func() { _ = resp.Body.Close() }()`, used throughout
`atlas-reward-pools`, `atlas-ban`, `atlas-cashshop`, etc.) and matched it —
did not invent a new pattern.

Verified with the pinned linter (`golangci-lint v2.13.1`, installed fresh
via `go install .../golangci-lint/v2/cmd/golangci-lint@v2.13.1` since the
pre-existing local binary was v2.12.2 and refused to load the v1.27
`.golangci.yml`), scoped with `--new-from-rev` against the branch's
merge-base with `origin/main` (`b0459f791`), matching how `tools/lint.sh`
itself gates new code — not a bare full-tree run, which surfaces pre-existing
unrelated findings elsewhere in the module that are not this unit's to fix:

```
$ golangci-lint run --allow-parallel-runners -c .golangci.yml \
    --new-from-rev b0459f791d9031990f96575ade08d8aab07b1ffb ./...
0 issues.
```

`go build ./... && go test ./...` from the module root: all packages `ok`,
including `ok atlas-account/account 1.543s`.

Committed alone: `912307b7d` — "fix(atlas-account): check resp.Body.Close
error in resource_test.go".

### Unit 2 — atlas-ui per-world characters panel

**Break confirmed before editing:** `AccountAttributes.characterSlots`,
`transformAccount`'s `Number(data.attributes.characterSlots)`, the dashboard
stats' `totalCharacterSlots`/`averageCharacterSlots`, and
`CharactersPanel`'s `account.attributes.characterSlots` all read an
attribute atlas-account no longer serializes on `GET accounts/{accountId}`
(round 1 removed it in favor of the world-scoped sub-resource) — each of
these was reading `undefined` and computing `NaN`.

**New plumbing, following the existing `wallet.service.ts`/`useWallet.ts`
single-nested-sub-resource pattern (the closest existing precedent for a
`GET accounts/{accountId}/worlds/{worldId}/...`-shaped resource):**

- `src/types/models/character-slots.ts` (new) — `CharacterSlots` /
  `CharacterSlotsAttributes` (`{worldId: number, slots: number}`), mirroring
  atlas-account's `CharacterSlotRestModel` wire shape
  (`{"worldId": byte, "slots": int16}`) byte-for-byte at the JSON level.
- `src/services/api/character-slots.service.ts` (new) —
  `characterSlotsService.getCharacterSlots(accountId, worldId, options)` →
  `api.getOne<CharacterSlots>("/api/accounts/{accountId}/worlds/{worldId}/character-slots", options)`,
  with a `transformCharacterSlots` coercing `worldId`/`slots` to `Number`
  (same defensive-coercion pattern as `accounts.service.ts#transformAccount`).
- `src/lib/hooks/api/useCharacterSlots.ts` (new) — `useCharacterSlots(tenant,
  accountId, worldId, options)`, a plain `useQuery` (no mutation — nothing in
  atlas-ui increments slots; that's the in-game B-Type coupon flow, round
  2/3's territory), keyed by `characterSlotsKeys.detail(tenant, accountId,
  worldId)`, `enabled: !!tenant?.id && !!accountId`, matching `useWallet`'s
  shape.

**Grouping-by-world, per the user ruling (one section per world, each with
its own slot count and its own characters — no summed total):**

- `src/components/features/accounts/WorldCharactersSection.tsx` (new) — one
  world's slice of the panel. Calls `useCharacterSlots(tenant, account.id,
  worldId)` itself (so the "call one hook per array element" problem is
  solved by making each world its own component instance, not by calling a
  variable number of hooks inside `CharactersPanel`). Renders that world's
  name + a `{filled}/{slots} slots` line, the filled/empty tile grid (reusing
  the existing `FilledSlotTile`/`EmptySlotTile`), and an over-capacity hint —
  all scoped to that world's own character count and slot count. A
  `LOADING_SKELETON_TILE_COUNT = 4` constant is a UI-only skeleton-grid
  placeholder while `useCharacterSlots` is in flight; it carries no game-rule
  meaning and is not a re-literal of the backend's
  `character.DefaultCharacterSlotsPerWorld` (that constant is
  atlas-account's, not atlas-ui's, to know).
- `src/components/features/accounts/CharactersPanel.tsx` (rewritten) — no
  longer reads `account.attributes.characterSlots` or filters characters by
  `accountId` alone across all worlds. Instead: filters characters by
  `accountId` once (`accountCharacters`, still correct — a character has
  exactly one `worldId`), then renders one `<WorldCharactersSection>` per
  entry in `tenantConfigQuery.data?.attributes?.worlds` (index = `worldId`,
  the same indexing `FilledSlotTile` already relies on for `worldName`
  lookup), passing each section only the characters whose `worldId` matches
  that section's world. The shared `ApplyPresetDialog` (which already has
  its own per-request world picker, defaulting to world 0) is left
  untouched — out of this unit's named file inventory, and changing its
  default-world behavior per clicked tile was not requested.

**Dashboard aggregate decision (`accounts.service.ts#getAccountStats` /
`useAccounts.ts#useAccountStats`):** dropped `totalCharacterSlots` and
`averageCharacterSlots` entirely rather than attempting to recompute them.
Reasoning, written into the source as a doc comment on `getAccountStats`:
`characterSlots` used to be free because it was already inlined on every
fetched `Account`; slots are now a separate per-(account, world)
sub-resource, so producing a tenant-wide total would mean issuing
`accounts.length * worlds.length` additional network calls from a function
that *already* drains every account for the tenant (`getAllAccounts`) — and
the resulting number would sum unrelated per-world caps into one figure that
isn't a meaningful single statistic. Confirmed by grep
(`grep -rln "getAccountStats\|totalCharacterSlots\|averageCharacterSlots" src`)
that no page currently renders either field — `useAccountStats` is exported
but has zero call sites outside its own hook file and its test — so this is
a genuine no-op removal, not a silent UI regression. A per-world breakdown
belongs on the per-world view this unit just built
(`CharactersPanel`/`WorldCharactersSection`), not a single tenant-wide
scalar.

### Test suites updated (compile-only fixes vs. new-behavior assertions)

Compile-only fixes — dropped the now-nonexistent `characterSlots` field
from mock `Account` fixtures, no behavioral assertions changed:
`components/features/accounts/__tests__/BirthDateDialog.test.tsx`,
`lib/hooks/api/__tests__/useAccountByName.test.tsx`,
`lib/hooks/api/__tests__/useAccounts.test.tsx` (also dropped
`totalCharacterSlots`/`averageCharacterSlots` from `mockStats`),
`lib/hooks/api/__tests__/useCreateAndPollAccount.test.ts`,
`pages/__tests__/AccountsPage.test.tsx`,
`services/api/__tests__/accounts.service.test.ts`,
`tests/integration/services/services.integration.test.ts` (this file is a
pre-existing orphan — excluded from both `tsconfig.app.json`'s `include:
["src"]` and Vitest's `include: ["src/**/*.test.{ts,tsx}"]`, and already
used `jest.mock`/stale service signatures unrelated to this change; fixed the
named `characterSlots` literals for correctness since the brief named the
path, but did not attempt a full rewrite of its pre-existing staleness —
out of scope).

New-behavior assertion —
`components/features/accounts/__tests__/CharactersPanel.test.tsx` rewritten:
added a `useCharacterSlotsMock` (mocking the new per-world hook, keyed by
the `worldId` argument it's called with) alongside the existing
`useCharactersMock`/`useTenantConfigurationMock`. All five pre-existing
cases updated to source slots from the mocked per-world hook instead of
`account.attributes.characterSlots`. Added one new test,
`"groups characters by world: each world shows only its own slot count and
its own characters, not a summed total"`, which is the load-bearing
assertion for the user's grouping ruling: two worlds (Scania: 3 slots/1
character, Bera: 2 slots/2 characters — exactly at capacity), asserts (via
`within(section)`) that each world's `FilledSlotTile` names, empty-tile
count, and `{filled}/{slots} slots` text are scoped to that world only —
specifically that Bera being *at* capacity (2/2) does **not** trigger the
over-capacity hint that a naive summed-across-worlds check (5 slots total, 3
characters total) would have hidden, and that neither section's slot count
equals the sum of both (3+2=5). This is the case that would fail under the
old "one flat total" reading and passes under the per-world grouping this
unit implemented.

### Build/lint/test evidence (foreground, read — not assumed)

```
$ cd services/atlas-ui && npm run build
```
`tsc -b && vite build` → clean, no type errors (test files included, per
`tsconfig.app.json`'s strict include). Vite build succeeded (`✓ built in
5.79s`); the only output was Rolldown's pre-existing "chunk >500kB" size
warning, unrelated to this change.

```
$ npm run test
```
`Test Files  259 passed (259)`, `Tests  2132 passed (2132)`. The single
stderr line (`Not implemented: navigation to another Document`) is a
pre-existing jsdom console note from an unrelated test, not a failure.

Also ran the specific touched suites in isolation first, before the full
sweep, to read their output directly:
```
$ npx vitest run src/components/features/accounts src/services/api/__tests__/accounts.service.test.ts \
    src/lib/hooks/api/__tests__/useAccounts.test.tsx src/lib/hooks/api/__tests__/useAccountByName.test.tsx \
    src/lib/hooks/api/__tests__/useCreateAndPollAccount.test.ts src/pages/__tests__/AccountsPage.test.tsx
Test Files  10 passed (10)
     Tests  65 passed (65)
```

```
$ npm run lint
```
`9 problems (0 errors, 9 warnings)` — confirmed by grep
(`npm run lint 2>&1 | grep -i "CharactersPanel\|WorldCharactersSection\|character-slots\|useCharacterSlots\|accounts.service"`
→ no matches, exit 1) that none of the 9 pre-existing `react-hooks` warnings
touch any file this unit changed; all are in unrelated coupon/reward-pool/
tenant dialogs and two pre-existing `exhaustive-deps` warnings in
`AccountsPage.tsx`/`QuestsPage.tsx`.

```
$ npx prettier --check <every file this unit touched>
Prettier: All files formatted correctly
```

### Round 4 self-review notes

- No TODOs/stubs/placeholders. Builder-pattern equivalent (no
  `*_testhelpers.go` — this is atlas-ui, but the same "no ad hoc test-only
  constructor sprawl" spirit is followed: test fixtures are plain object
  literals matching the existing file's own style, no new helper module).
- `character.MaxCharacterSlotsPerWorld`/`DefaultCharacterSlotsPerWorld`
  (`libs/atlas-constants/character`) were **not** re-literalled into
  atlas-ui — the `LOADING_SKELETON_TILE_COUNT = 4` constant in
  `WorldCharactersSection.tsx` is explicitly documented as a UI-only
  placeholder with no game-rule meaning, not a copy of the backend default.
- Deliberate scope trim: did not add dedicated unit tests for
  `character-slots.service.ts` or `useCharacterSlots.ts` in isolation — the
  brief's inventory named only the *existing* suites to update, and the
  closest existing precedent (`wallet.service.ts`/`useWallet.ts`, the
  pattern this unit copied) has no dedicated test file of its own either.
  Both are exercised indirectly through `CharactersPanel`/
  `WorldCharactersSection`'s mocked-hook test coverage. Flagged here for a
  reviewer to rule on if direct coverage is wanted.
- Deliberate scope trim: `ApplyPresetDialog` was not touched to pre-select
  the clicked tile's world (its own form's world picker still defaults to
  world 0 regardless of which world's empty tile was clicked). Not named in
  the brief's file inventory; flagged as a UX follow-up, not a defect in
  this unit's stated scope.
- Verified in the foreground per the brief's instruction — no backgrounded
  build/test/lint calls, every command's real output was read before this
  report was written.

### Commits

- `912307b7d` — "fix(atlas-account): check resp.Body.Close error in
  resource_test.go" (Unit 1, alone).
- Unit 2 commit follows immediately after this report is written (see final
  status line for its SHA).
