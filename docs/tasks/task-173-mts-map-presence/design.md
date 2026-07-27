# task-173 — MTS characters must leave the map like cash-shop characters

## Problem

A character who is in the MTS (Maple Trade Space / ITC auction scene) is still
rendered in the map to another character who **enters that map after** the MTS
character was already there. Symptom: character A enters the MTS (correctly
disappears from everyone currently in the map); then character B enters or
re-enters that same map and sees a ghost of A standing in it.

Entering/exiting the MTS while both characters are already in the map works
correctly — the despawn and respawn broadcasts fire as expected. The gap is
purely the **map-enter snapshot** for a player who joins after A is already in
the MTS.

## Root cause

The list of "characters already present in this map" sent to a joining player is
built from the channel's **in-memory session registry**, filtered only by exact
field match:

`services/atlas-channel/atlas.com/channel/session/processor.go:101`
— `InFieldModelProvider`:

```go
if s.CharacterId() != 0 && s.Field().Equals(f) {
    result = append(result, s)
}
```

This is the single production source feeding
`map/processor.go:44` `CharacterIdsInMapModelProvider` →
`kafka/consumer/map/consumer.go` `fetchOtherCharactersInMap` (map-enter
self-spawn) and the inter-character spawn broadcast.

- **Cash shop** does not exhibit the bug because entering it migrates the client
  onto a separate server connection; the channel socket closes and the session
  is destroyed and removed from the registry
  (`socket/init.go:54` → `session/processor.go` `Destroy`). Once removed, it can
  never appear in `InFieldModelProvider`.
- **MTS** renders in-place on the *same* channel connection: `EnterMtsHandleFunc`
  (`socket/handler/mts_entry.go:79`) pushes the ITC scene via `SetItcWriter`
  (the cash-shop `CStage`) and only sets a session flag
  `SetCashScene(CashSceneMts)` (`mts_entry.go:94`). The session stays alive in
  the registry with its **original map field**, so `s.Field().Equals(f)` still
  matches and A is spawned as a ghost for the entering player B.

The leave/despawn path and the MTS-exit path already work: both entry handlers
emit the shared `cashshop.Enter` status event (atlas-maps → MAP exit →
`CharacterDespawn` broadcast), and MTS exit resets `CashScene` to
`CashSceneNone` (`socket/handler/map_change.go:39`, the cash-shop-return branch)
and re-enters/re-spawns. So the flag is correctly set on entry and cleared on
exit — nothing on the map-enter *read* path consults it.

`CashScene` values (`session/model.go:19-21`):
`CashSceneNone = 0`, `CashSceneCashShop = 1`, `CashSceneMts = 2`.

## Chosen approach — filter the in-field query by cash scene

Restore the invariant that "in field" means "physically present in the map."
Cash shop satisfies this by removing the session from the registry; MTS violates
it by keeping the session alive. Make the in-field query skip any session that is
in a cash scene:

`session/processor.go:101` `InFieldModelProvider` — add the scene guard to the
inclusion predicate:

```go
if s.CharacterId() != 0 && s.CashScene() == CashSceneNone && s.Field().Equals(f) {
    result = append(result, s)
}
```

Rationale for the exact condition:

- Key on `== CashSceneNone` (exclude **any** cash scene) rather than
  `!= CashSceneMts`. It is the correct invariant — a cash-shop-scened session
  should never be "in the field" either — and it is defensive: if a cash-shop
  session ever lingers in the registry, it is excluded too. In normal operation
  cash-shop sessions are already gone, so this only changes behaviour for MTS.

### Why this location

`session.InFieldModelProvider` has exactly **one** production consumer chain
(`map/processor.go:44` → `CharacterIdsInMapModelProvider` →
`GetCharacterIdsInMap` / `ForSessionsInMap`), so the blast radius is fully
understood. Filtering here excludes MTS-scened sessions from both:

1. the map-enter "who is already here" snapshot (fixes the reported ghost), and
2. field-scoped broadcasts driven off the same provider (map chat, weather
   buffs, field effects/BGM),

which is exactly what cash shop already gets for free by session removal. An MTS
character is not visually in the map and should neither be seen nor receive
field broadcasts — consistent with cash shop.

### Why not the alternatives

- **Sentinel/clear the session `Field()` on MTS entry.** `Field()` is read across
  wallet routing, the exit/return migration, saga asset resolution, and the ITC
  handlers; mutating it risks breaking return-to-map and MTS operations. More
  surface, more risk, for no benefit over the scene filter.
- **Migrate/destroy the session like cash shop.** The ITC flow is deliberately
  in-place on the live connection and depends on the persistent session plus
  `CashSceneMts` routing (`kafka/consumer/wallet/consumer.go:76`). Re-architecting
  it to migrate off the connection is large and unnecessary.

## Scope

- Change: the same cash-scene guard in **both** field-membership queries in
  `services/atlas-channel/atlas.com/channel/session/processor.go`:
  - `InFieldModelProvider` — feeds the map-enter snapshot and field-scoped
    broadcasts (the reported ghost).
  - `InMapAllInstancesModelProvider` — its sole production consumer is the
    transport arrival/departure broadcast (`kafka/consumer/route/consumer.go:70`,
    `:91` via `map/processor.go:88` `ForSessionsInMapAllInstances`). Without the
    guard an MTS-scened session in a transport map would still receive
    `FieldTransportStateWriter` announcements — the same "queried as present when
    not physically there" defect. Added in the same change for consistency.
- No packet, template, Kafka, or atlas-maps changes. The entry event, despawn
  broadcast, and exit/reset paths are unchanged and already correct.
- No behaviour change for cash-shop sessions (already absent from the registry)
  or for ordinary in-map sessions (`CashScene() == CashSceneNone`).

## Testing

Unit tests in `session/processor_test.go`, alongside the existing
`TestInFieldModelProvider_*` cases:

1. **Excludes MTS-scened sessions** — register two sessions on field `f`, one
   with `CashSceneNone` and one set to `CashSceneMts`; assert
   `InFieldModelProvider(f)` returns only the `None` session.
2. **Excludes cash-shop-scened sessions** — same shape with `CashSceneCashShop`;
   assert exclusion (guards the invariant defensively).
3. **Regression: unscoped sessions still returned** — the existing exact-match
   test must still pass (a default `CashSceneNone` session is included).

Existing `TestInFieldModelProvider_*` and `map/processor_test.go` cases use
default (`CashSceneNone`) sessions and must remain green.

### Verification

- `go test -race ./...` clean in `services/atlas-channel`.
- `go vet ./...` clean.
- `go build ./...` clean.
- `docker buildx bake atlas-channel` — not required unless `atlas-channel`'s
  `go.mod` is touched (it is not), but run the standard channel build/test gates.
- Manual/live confirmation of the exact repro: A enters MTS, then B enters the
  map, and B no longer sees A.
