# Task 32 review — atlas-mts consumes `NAME_CHANGED`

Commit reviewed: `526a3b50b`. 6 files, +310 lines. Read-only review; no edits made.

## Verdict: PASS

Every priority item checked against source. No blocking findings.

## 1. Producer/consumer contract — PASS

Diffed both sides directly (not trusting the report's quote):

Producer, `services/atlas-character/atlas.com/character/kafka/message/character/kafka.go:223-249`:
```go
EnvEventTopicCharacterStatus     = "EVENT_TOPIC_CHARACTER_STATUS"
...
StatusEventTypeNameChanged       = "NAME_CHANGED"
```
`:265-271`:
```go
type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       world.Id  `json:"worldId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}
```
`:359-362`:
```go
type StatusEventNameChangedBody struct {
	OldName string `json:"oldName"`
	NewName string `json:"newName"`
}
```

Consumer, `services/atlas-mts/atlas.com/mts/kafka/message/character/kafka.go:23-46` — `EnvStatusEventTopic`, `StatusEventTypeNameChanged`, `StatusEvent[E any]`, `StatusEventNameChangedBody` all match field-for-field: same field names, same order, same json tags, same types (`uuid.UUID`, `world.Id` from the same `github.com/Chronicle20/atlas/libs/atlas-constants/world` import, `uint32`, `string`). Confirmed independently — the report's claim holds.

Naming differs cosmetically (`EnvStatusEventTopic` vs producer's `EnvEventTopicCharacterStatus`, both string-value-identical to `"EVENT_TOPIC_CHARACTER_STATUS"`) but that's a local const name, not wire-visible — correct per the brief's instruction to follow this service's own sibling-package idiom (`kafka/message/custody`, `kafka/message/mts` both use `Env*Topic`/`StatusEventType*`).

## 2. New subscription wired end to end — PASS

`main.go:6` imports `characterConsumer "atlas-mts/kafka/consumer/character"`.
`main.go:87-99`:
```go
cmf := consumer.GetManager().AddConsumer(l, rt.Context(), rt.WaitGroup())
custodyConsumer.InitConsumers(l)(cmf)(consumerGroupId)
mtsConsumer.InitConsumers(l)(cmf)(consumerGroupId)
characterConsumer.InitConsumers(l)(cmf)(consumerGroupId)
if err := custodyConsumer.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil { ... }
if err := mtsConsumer.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil { ... }
if err := characterConsumer.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil { ... }
```
Both `InitConsumers` and `InitHandlers` are present, placed as the third pair beside the existing custody/mts pairs, same shape (fatal-on-error for `InitHandlers`, bare call for `InitConsumers`). No asymmetry — neither call exists without its counterpart.

`kafka/consumer/character/consumer.go` `InitConsumers`/`InitHandlers` are structurally identical to `kafka/consumer/custody/consumer.go`'s idiom: `consumer2.NewConfig(l)("mts_character_status")(character.EnvStatusEventTopic)(consumerGroupId)` with the same `SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser)` decorator, and `InitHandlers` resolves the topic via `topic.EnvProvider` then registers through `message.AdaptHandler(message.PersistentConfig(...))`, exactly like custody's registration calls.

**Deploy-config recheck (independent, per brief instruction to verify myself):**
```
deploy/compose/.env.example:78:EVENT_TOPIC_CHARACTER_STATUS=EVENT_TOPIC_CHARACTER_STATUS
deploy/k8s/overlays/pr/kustomization.yaml:266
deploy/k8s/overlays/main/kustomization.yaml:142
deploy/k8s/base/env-configmap.yaml:105
```
Four files confirmed, matching the brief's count (not the report's three — the report missed `deploy/compose/.env.example`, which is exactly the already-settled discrepancy called out in the review brief; no deploy change was needed regardless since `atlas-mts`'s k8s manifest uses `envFrom: configMapRef`). Not re-raising per instruction — recorded here only as the independent verification the brief asked for.

## 3. State ruling — PASS

`updateSellerName` (`listing/administrator.go:178-186`):
```go
func updateSellerName(db *gorm.DB, sellerId uint32, newName string) (int64, error) {
	result := db.Model(&entity{}).
		Where(map[string]interface{}{"seller_id": sellerId}).
		Updates(map[string]interface{}{
			"seller_name": newName,
			"updated_at":  time.Now(),
		})
	return result.RowsAffected, result.Error
}
```
No `state` key in the `Where` map — every row for `seller_id` is updated regardless of `State`, as ruled.

Independently swept for a point-in-time `seller_name` record: `transaction/` package's `CreateTransaction` row carries `SellerId`/`ItemId`/`Quantity`/`TotalPrice` — no name field (grep for `SellerName`/`seller_name` across `listing/*.go` and `transaction/*.go` turned up only the live `entity.SellerName` column, the `AcceptRequest.SellerName` field used to *construct* a new listing row in `processor_custody.go`, and DTO/query plumbing — none of it a historical snapshot). Confirms the report's claim; no leaked point-in-time treatment found.

Tenant scoping: `p.db.WithContext(p.ctx)` is the same construction every other `ProcessorImpl` method uses (`processor.go:377`, consistent with `:361` `UpdateState`, `:340` `CreateListing`, etc.) — automatic tenant scope applies per the already-settled ruling.

## 4. The five tests — PASS

`kafka/consumer/character/consumer_test.go` — all five tests build fixtures via `listing.NewBuilder(...).Build()` + `listing.CreateListing`, matching the Builder pattern (`SetSellerName`, `SetSaleType`, `SetState`, etc.), no `*_testhelpers.go` file (helpers `seedListing`/`sellerNameOf`/`nameChangedEvent` live directly in the test file, package-private).

- `TestNameChangedUpdatesEverySellerListing` — two listings for seller 1, asserts both renamed. Would fail if the handler were a no-op or removed (both assertions read back `SellerName` post-handler-call).
- `TestNameChangedRenamesSoldAndCancelledListingsToo` — one active + one `StateSold` listing, asserts both renamed — this is the State-ruling test; would fail under a `state`-scoped WHERE.
- `TestNameChangedNoListingsIsNoop` — seller 999 has no rows; handler call must not panic/error (no explicit error assertion beyond not failing, appropriate for a no-op).
- `TestNameChangedIgnoresOtherEventTypes` — mutates `ev.Type = "LOGIN"`, asserts the listing's name is unchanged — this is the type-guard test; would fail if the guard (`if e.Type != character.StatusEventTypeNameChanged { return }`, `consumer.go` in `handleCharacterNameChanged`) were removed.
- `TestNameChangedRedeliveryIsIdempotent` — calls the handler twice with the same event, asserts final state is `Zulu` with no error — DB-value assertion is sufficient here per the brief (no emit to count). Would fail if a second identical UPDATE somehow produced a different or erroring result, though note this test does not distinguish "idempotent" from "merely repeatable" since both deliveries write the same value — a stricter test would need to assert no error on the second call explicitly, but the current assertion is adequate for the stated ruling (single UPDATE keyed on seller_id, not an insert-then-conflict pattern, so there's no realistic idempotency failure mode to catch beyond what's covered).

Confirmed `test.SetupTestDB`, `test.CleanupTestDB`, `test.CreateTestContext`, `test.TestTenantId` all exist in `services/atlas-mts/atlas.com/mts/test/` — tests are grounded in the real rig, not invented helpers.

## 5. DOM-* conformance — PASS

- **Layering**: consumer (`kafka/consumer/character/consumer.go`) calls `listing.NewProcessor(l, ctx, db).RenameSeller(...)` — processor, not administrator, directly from the consumer. Processor (`processor.go:376-378`) thin-wraps the unexported `updateSellerName` administrator function. Correct consumer → processor → administrator layering.
- **Immutability**: no new exported mutable model type introduced; `updateSellerName` operates on the existing `entity` GORM row via a scoped `Updates` map, same shape as the pre-existing `UpdateState`.
- **Tenant from context**: `p.db.WithContext(p.ctx)` — consistent with every other processor method in the file.
- **No reinvented constants**: `world.Id` reused from `libs/atlas-constants/world`, same import path both producer and consumer side. No new domain type/id introduced.

## Notes (non-blocking, informational only per already-settled rulings)

- Confirmed independently (per priority-3 instruction): no live listing-updated push consumer exists in `kafka/message/mts/` — its `StatusEventType*` vocabulary (`LISTING_CREATED`, `LISTING_CANCELLED`, `BID_PLACED`, `OUTBID`, `LISTING_SOLD`, `LISTING_EXPIRED`, `ITEM_MOVED_TO_HOLDING`, `ITEM_TAKEN_HOME`, `WISH_ADDED`, `WISH_REMOVED`, plus four `*_FAILED` variants) contains nothing rename-shaped. "Emit nothing" ruling stands; not re-raised as a finding per the brief.
- Deploy-config discrepancy (three files found by the implementer vs. four actual) does not change the ruling and is not re-raised as a finding per the brief — recorded above only as the requested independent verification.
