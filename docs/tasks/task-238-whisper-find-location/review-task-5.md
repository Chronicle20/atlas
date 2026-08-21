# Review — Task 5: atlas-maps cash-shop presence transitions (CHARACTER_ENTER / CHARACTER_EXIT)

Range reviewed: `9254bd6b5..296ae8b40` (single commit `296ae8b40`).

## Scope

`git diff --stat` shows exactly three files touched, all inside the stated
surface:

```
services/atlas-maps/atlas.com/maps/kafka/consumer/cashshop/consumer.go       |  39 +++--
services/atlas-maps/atlas.com/maps/kafka/consumer/cashshop/consumer_test.go  | 164 +++++++++++++++++++++
services/atlas-maps/atlas.com/maps/main.go                                   |   2 +-
```

`atlas-cashshop` has no diff in this range — confirmed via `git diff --stat
9254bd6b5..296ae8b40 -- atlas-cashshop` (empty). Matches the constraint.

## Requirement-by-requirement

1. **Both transitions are conditional (`SetStateIfOnline`), never `SetState`.**
   Confirmed at `services/atlas-maps/atlas.com/maps/kafka/consumer/cashshop/consumer.go`
   in both `handleStatusEventEnterFunc` and `handleStatusEventExitFunc` — each
   calls `location.NewProcessor(l, ctx, db).SetStateIfOnline(...)`, never
   `SetState`. `grep -n SetState` on the diff shows only `SetStateIfOnline`
   call sites.

   Traced the conditional guard itself (dependency of this task, not part of
   its diff, but load-bearing so verified): `character/location/administrator.go:53-64`
   (`setLocationState`) — when `conditional=true` it adds
   `q.Where("state <> ?", string(characterconst.PresenceStateOffline))`
   before the `Update`. A row already `OFFLINE` fails the `WHERE` clause and
   the `Update` is a no-op (0 rows affected, `gorm` returns nil error). This
   genuinely refuses to write over an `OFFLINE` row — not a comment-only
   promise.

2. **Mapping.** `CHARACTER_ENTER` → `PresenceStateInCashShop`,
   `CHARACTER_EXIT` → `PresenceStateInField` — both directions correct at
   `consumer.go` (enter handler sets `PresenceStateInCashShop`, exit handler
   sets `PresenceStateInField`).

3. **`OFFLINE` asymmetry preserved, not "harmonised."** Task 5's handlers use
   only `SetStateIfOnline`; no `SetState` call was added or altered here.
   Task 4's LOGIN/CHANNEL_CHANGED path (outside this diff) is untouched.
   Correct per the explicit instruction that harmonising would be a false
   finding.

4. **No TTL / sweeper invented.** No timer, ticker, or timeout constant
   appears anywhere in the diff.

5. **Idempotency under at-least-once delivery.** `SetStateIfOnline` is a
   `WHERE ... UPDATE` keyed on `(tenant_id, character_id)`; a replayed ENTER
   sets the same `IN_CASH_SHOP` value twice — no visible side effect beyond a
   redundant write. Test `TestEnterHandler_IsIdempotent` exercises this
   directly.

6. **Structural change — `db` threading.**
   - `InitHandlers(l logrus.FieldLogger, db *gorm.DB)` signature change at
     `consumer.go:33`, matches the brief and the character-consumer shape at
     `kafka/consumer/character/consumer.go:40` exactly (same parameter order,
     same closure pattern via `handleStatusEvent*Func(db)`).
   - `main.go:93`: `cashshop.InitHandlers(l, db)(consumer.GetManager().RegisterHandler)`
     — the `nil` literal argument is gone; `db` (already in scope, same
     variable passed to `character.InitHandlers(l, db)` two lines above at
     `main.go:90`) is passed instead.
   - `go build ./...` from `services/atlas-maps/atlas.com/maps` exits clean —
     confirms the call site compiles against the new signature.
   - The pre-existing `nil` passed to `_map.NewProcessor(l, ctx,
     producer.ProviderImpl(l)(ctx), nil)` inside both handlers is untouched,
     as the brief specified (that processor's in-memory registry path does
     not use a DB handle) — verified this is a different `nil`/different
     processor than the `location.Processor` that now carries `db`, so this
     is not a residual defect, it is the documented pre-existing shape.

7. **Test honesty.** Five new tests in `consumer_test.go`, none from a
   `*_testhelpers.go` file (setup lives in the test file itself, using the
   existing `location.NewProcessor` / `field.NewBuilder` builders — no new
   test-only constructor type was introduced). Each test:
   - `TestEnterHandler_SetsInCashShop` / `TestExitHandler_SetsInField` — seed
     a live row in the opposite state, invoke the handler, assert the new
     state. These fail without the change (old handlers had no `location`
     write at all, so `stateOf` would still read the seeded state).
   - `TestEnterHandler_IsIdempotent` — invokes the enter handler twice,
     asserts the state is still `IN_CASH_SHOP`. This does not distinguish
     conditional-vs-unconditional (both `SetState` and `SetStateIfOnline`
     would pass this test starting from a live row) — it genuinely pins
     idempotency but not conditionality; that's fine, the resurrection tests
     below cover conditionality specifically.
   - `TestExitHandler_DoesNotResurrectOfflineCharacter` /
     `TestEnterHandler_DoesNotResurrectOfflineCharacter` — seed `OFFLINE`,
     invoke the handler, assert the row is still `OFFLINE`. This is the test
     that would fail under `SetState` (unconditional) and passes only under
     `SetStateIfOnline` — it is the one genuinely pinning the load-bearing
     constraint of this task.

   Report claims all five pass; `go build ./...` and `go vet
   ./kafka/consumer/cashshop/...` from this review both came back clean,
   consistent with the report's build/test evidence. Per instructions this
   review does not re-run the full suite the implementer already ran.

## Not evaluable

None — the diff is small and self-contained, and every symbol it depends on
(`SetStateIfOnline`, `location.Migration`, `location.Model.State()`,
`characterconst.PresenceState*`) was locatable and inspectable within the
review surface.

## Verdict rationale

All requirements met, the load-bearing conditional-write constraint is
verified down to the SQL `WHERE` clause (not just the handler-level comment),
the structural `db`-threading is complete and matches the sibling consumer
it was modelled on, and the new tests include one that specifically
distinguishes conditional from unconditional behavior. No blocking findings.
