# task-227 — Implementation Context

Companion to [`plan.md`](plan.md). Everything here was read from source in this worktree or decompiled during design; nothing is recalled. File:line references are to this branch.

---

## 1. What already exists (do not rebuild it)

| Claim | Evidence |
|---|---|
| The 540 classifier arm exists and is already version-scoped (52/53 pre-v95, 53/54 on GMS ≥ 95), split by item-id prefix `5400xxx` / `5401xxx` | `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:1117-1140` |
| …and contains dead code: lines 1132-1138 are an exact duplicate of the `5401` branch at 1125-1131 | same |
| Both purchase serverbound codecs already decode in full | `libs/atlas-packet/cash/serverbound/shop_operation_buy_name_change.go`, `..._buy_world_transfer.go`; handlers at `cash_shop_operation.go:182-194` (log-only) |
| Mode bytes 46/49 are already config-resolved via `options["operations"]` | `cash_shop_operation.go:198-215` (`isCashShopOperation`) |
| The success/failure clientbound bodies already exist | `libs/atlas-packet/cash/clientbound/shop_operation_body.go:454` (`CashShopTransferWorldFailedBody`), `:658` (`CashShopNameChangeBuyDoneBody`), `:667` (`CashShopTransferWorldDoneBody`) |
| A name validator and a name-availability endpoint already exist and are already shared | `character/processor.go:268-289` (`CheckNameValidity`), `character/name_validity_resource.go`, consumed over REST by `services/atlas-character-factory/.../character/name_validity_requests.go` |
| `getForName` is already tenant-wide and already case-insensitive | `character/provider.go:30-39` (`LOWER(name) = LOWER(?)`) — the per-world narrowing happens in the processor, not the query |
| `NAME_CHANGED` is emitted but has **zero** consumers outside atlas-character | `character/producer.go:261-272`, wired at `character/processor.go:1926`; repo-wide grep |
| atlas-character **already** consumes its own status topic | `kafka/consumer/character/consumer.go:24` subscribes `character_event_status`; `handleLevelChangedStatusEvent` (`:301`), `handleJobChangedStatusEvent` (`:312`) |
| `LOGOUT` is already published with world/channel/map | `character/producer.go:66-79` (`logoutEventProvider`) |
| atlas-inventory has **no world column** — compartments key on `(TenantId, Id, CharacterId, InventoryType)` | `services/atlas-inventory/atlas.com/inventory/compartment/entity.go:14-20`, `asset/entity.go:22-30` |
| Version is a **tenant** property, so two worlds in one tenant cannot run different client versions | `libs/atlas-tenant/tenant.go:21-25` |
| Pink text already has a body builder | `services/atlas-channel/atlas.com/channel/socket/writer/world_message.go:107` |

## 2. Key decisions and why

- **One table, two request types.** The lifecycle, consumption/refund contract, expiry sweep, cancel endpoint and notification delivery are identical between the two flows; two tables would mean two of everything for one differing column. The discriminator costs one `type` check at apply time. (design §3.1)
- **The record lives in atlas-character**, not atlas-cashshop and not a new service. The reservation index must sit in the same database as the uniqueness check it guards — that is the only place FR-3.3 and FR-3.2 can be made atomic with respect to each other. A new service would own one table whose every read joins `characters`, at the cost of ~8 hand-maintained registration lists. (design §3.2)
- **The safe point is `LOGOUT`, with a LOGIN catch-up.** Both entry points require a live channel session, so a request is always created while online and the next transition is necessarily logout. The rejected alternative — hooking atlas-login's `CharacterListWorldHandleFunc` — would block a synchronous compensating saga at the most latency-sensitive point in the login flow. (design §3.3)
- **Refund idempotency is enforced at the record, not the handler.** The refund is emitted only by the transition that actually moves `status` out of `PENDING`, inside the same transaction, through the outbox. A redelivered cancel finds `RowsAffected == 0` and emits nothing. This is the codebase's known at-least-once failure mode, so it gets the plan's most important test. (design §3.10; plan Task 6)
- **Escrow systems block rather than auto-settle** (departure from PRD FR-4.4). Each is player-visible and player-fixable with its own close flow; auto-cancelling an auction someone already bid on is not reversible by compensation; and it removes the three highest-risk compensating steps. (design §3.6, risk R5 — revisit before Task 11 if auto-settlement was genuinely intended)
- **GM and family gates are added beyond the PRD.** The v83 client's own `CCashShop::CheckTransferWorldPossible` (`@0x4734e5`) refuses to send the request for a guild master, a GM, or a character in a family — a server that permits it produces a state the client considers impossible. (design §1.6)
- **The availability check is synchronous REST, not Kafka.** The client blocks its dialog on the result and nothing is mutated, so a saga would add hundreds of milliseconds to a keystroke-batch interaction for no atomicity benefit. (design §3.5)
- **`GET /characters/name-availability` (PRD §5) is not created** — the existing `/characters/name-validity` is extended with a `scope` parameter and a `reserved` reason instead. A second name for an existing route is not an API. (design §3.4)
- **Offline notification is a column, not a queue.** `notified_at` on the record, drained by the LOGIN catch-up. Nothing else in the codebase needs a generic deferred-notification queue today, so building one for a single caller is the wrong ratio. (design §3.9, resolves OQ-8)
- **No new index on `characters`.** A tenant-wide unique index would fail the migration on any tenant with a pre-existing cross-world duplicate — unsurveyable from this checkout — and the failure mode is a service that will not start. The tenant-wide check is a query; only the *reservation* gets an index, on the new table where every row is ours. (design §3.4, resolves OQ-5)
- **Expiry is tenant-configurable through a new `imprint-configs` resource**, modelled on `services/atlas-trades/atlas.com/trades/configuration/`, defaulting to 168h when unseeded. atlas-character consumes no tenant configuration today, so this adds the smallest complete instance of the existing pattern rather than an env var. (plan Task 8)

## 3. Open questions, and where each is answered

| OQ | State | Where |
|---|---|---|
| OQ-1 540 item ids / prefix→feature | **Blocking; resolved by derivation.** Two independent routes: atlas-data per-version 540 templates, or the client's own classifier arm. Stops with BLOCKED rather than guessing. | plan Task 1 Steps 5-6 |
| OQ-2 per-version `CashSlotItemType` | **Already implemented** — 52/53 pre-v95, 53/54 on GMS ≥ 95 | `character_cash_item_use.go:1117-1140` |
| OQ-3 which safe point fires first | **Logout, always** | design §3.3; plan Task 9 |
| OQ-4 inventory world-scoped? | **No** — purely character-scoped; FR-4.5's inventory step is a no-op | `compartment/entity.go:14-20` |
| OQ-5 tenant-wide duplicate names in live data | **Unsurveyable; sidestepped** by a query-level check with no new constraint | design §3.4; plan Task 5 |
| OQ-6 per-world version divergence | **Impossible** — version is on the tenant | `libs/atlas-tenant/tenant.go:21-25` |
| OQ-7 reason-code enumerations | Derived per version during Task 1 Step 7; server-side taxonomy is already fixed (design §6) | plan Tasks 1, 17, 18 |
| OQ-8 offline notification delivery | **`notified_at` column** + LOGIN re-emit | plan Tasks 2, 6, 9 |
| OQ-9 does the client accept `CANCEL_*` outside the cash shop | Derived during Task 20; **behaviour does not depend on the answer** — the consumer sends pink text alongside the packet either way | plan Tasks 20, 27 |

## 4. Traps specific to this task

- **The IDB symbols in this neighbourhood are transposed or the matrix is.** `@0x47359c`, symboled `SendCheckNameChangePossiblePacket`, builds `COutPacket(18)` = `0x012`, which `STATUS.md:532` calls `WORLD_TRANSFER`. And `@0x470480`, symboled `OnBuyNameChange`, calls `CheckTransferWorldPossible` and formats transfer-world error strings. Writing codecs against a transposed pair yields two structurally-plausible decoders that are silently swapped, and byte-fixture tests pass on both. **This is why Task 1 blocks everything.** (design §1.8)
- **The check packets carry the account's second password.** v83 `CCashShop::OnBuyNameChange` calls `ask_SPW()` and passes the result into the send. Every serverbound handler in atlas-channel logs `p.String()` at debug, so redaction must live in the struct's `String()`, not at the call site. (design §1.7)
- **A socket handler with a missing validator is silently dropped** at config load. Every handler registered in Phase E needs one.
- **A seed-template writer requires a non-empty `fname`** and validates against an exact corpus count.
- **New opcodes absent from a *live* tenant's config are dropped** even when the seed template has them — seeding a template is not the same as reconciling a live tenant.
- **A registry `fname` edit stales the matrix**, which is why plan Task 35 Step 5 re-greps the matrix after Phase E's writer registrations rather than trusting Task 23's result.
- **The Kafka contract in `kafka/consumer/pendingchange/` is a cross-module copy** with no compile-time link to atlas-character's definition. A field name or json tag that drifts decodes to a zero-valued body at runtime with no build error — the exact failure the trade/mist/npc-shop mirror guards exist to catch. Mirror the tags exactly.
- **atlas-ui `npm run build` type-checks the tests**, so a green Vitest run alone is not verification here. `npm` needs nvm 22; `tools/lint.sh --check` false-fails without it.
- **`api.getOne` unwraps the JSON:API `.data` envelope but `api.post`/`api.delete` do not** — read and write calls resolve to different shapes and both need normalizing (see `teleport-rocks.service.ts`).
- **`tools/service-registration-guard.sh` is not needed** — this task adds no new service.

## 5. Patterns to copy, by file

| Need | Copy from |
|---|---|
| Sub-package (entity/model/builder/administrator/provider/rest) | `services/atlas-character/atlas.com/character/saved_location/` |
| Sub-package REST resource with a `characterId` path prefix | `services/atlas-character/atlas.com/character/teleport_rock/resource.go` |
| Typed validation sentinels mapped to both HTTP status and wire reason | `services/atlas-character/atlas.com/character/teleport_rock/processor.go:20-42` |
| Transactional emit (`database.ExecuteTransaction` + `message.Emit(outbox.EmitProvider(...))`) | `character/processor.go:1888-1893` |
| Status-event provider | `character/producer.go:261-272` (`nameChangedEventProvider`) |
| Kafka consumer + `InitConsumers`/`InitHandlers` currying | `kafka/consumer/character/consumer.go:21-60` |
| Ticker task (`Run()` + `SleepTime()`, otel span, per-tenant context) | `session/task.go` |
| Tenant configuration client (registry + `Extract` zero-folding + `DefaultConfig`) | `services/atlas-trades/atlas.com/trades/configuration/` |
| Saga step handler | `saga/handler.go:1371` (`handleIncreaseBuddyCapacity`) |
| Saga compensator | `saga/compensator.go:1631` (`compensateMesoSackUse`) |
| Saga compensation test harness | `saga/trade_compensation_test.go`, `saga/meso_sack_compensation_test.go` |
| Cash item-use dispatch arm, incl. EnableActions discipline | `character_cash_item_use.go:334-400` (expiration extender) |
| Version-scoped cash-slot-type helper | `character_cash_item_use.go:867-877` (`viciousHammerCashSlotItemType`) |
| atlas-channel REST client | `services/atlas-channel/atlas.com/channel/character/requests.go` |
| atlas-ui service (JSON:API typing + envelope normalization) | `services/atlas-ui/src/services/api/teleport-rocks.service.ts` |
| atlas-ui hooks (query keys, `enabled` gate, invalidation) | `services/atlas-ui/src/lib/hooks/api/useTeleportRocks.ts` |
| atlas-ui character card | `services/atlas-ui/src/components/features/characters/TeleportRockListCard.tsx` |
| Coverage manifest | `docs/tasks/task-206-cash-shop-coupon-codes/coverage-manifest.yaml` |

## 6. Cross-service Kafka contracts this task consumes

| Command / event | Constant | File |
|---|---|---|
| Guild leave | `CommandTypeLeave = "LEAVE"`, topic `COMMAND_TOPIC_GUILD` | `services/atlas-guilds/atlas.com/guilds/kafka/message/guild/kafka.go:12,22` |
| Party leave | `CommandPartyLeave = "LEAVE"`, topic `COMMAND_TOPIC_PARTY` | `services/atlas-parties/atlas.com/parties/party/kafka.go:6,9` |
| Buddy delete | `CommandTypeRequestDelete = "REQUEST_DELETE"`, topic `COMMAND_TOPIC_BUDDY_LIST` | `services/atlas-buddies/atlas.com/buddies/kafka/message/list/kafka.go:12,18` |
| Character status | `EnvEventTopicCharacterStatus`, `StatusEventTypeNameChanged` / `Login` / `Logout` | `services/atlas-character/atlas.com/character/kafka/message/character/kafka.go:223-236` |
| Saga asset actions | `DestroyAsset = "destroy_asset"`, `AwardAsset = "award_asset"` | `libs/atlas-saga/model.go:67,73` |

## 7. Eligibility-gate REST route prefixes (read the registration; do not guess a sub-path)

| Service | Prefix | Registered in |
|---|---|---|
| atlas-world | `/worlds`, `/worlds/{worldId}/channels` | `services/atlas-world/.../resource.go` |
| atlas-account | `/accounts` | `services/atlas-account/.../resource.go:31` |
| atlas-ban | `/bans` | `services/atlas-ban/.../resource.go:25` |
| atlas-guilds | `/guilds` | `services/atlas-guilds/.../resource.go:22` |
| atlas-families | `GET /families/tree/{characterId}` | `services/atlas-families/atlas.com/family/family/resource.go:28` |
| atlas-trades | `/trades/rooms` | `services/atlas-trades/.../resource.go:50` |
| atlas-merchant | `/characters/{characterId}` | `services/atlas-merchant/.../resource.go:20` |
| atlas-mts | `/characters/{characterId}/mts/holding` | `services/atlas-mts/.../resource.go:35` |

## 8. Services deliberately NOT wired, and why

Recorded here so an absence is never mistaken for an oversight.

- **atlas-parties, atlas-messengers, atlas-marriages, atlas-trades** — hold the character name only in in-memory registries rebuilt at login (`parties/character/registry.go:36`, `messengers/character/registry.go:35`, `marriages/character/model.go:5`, `trades/trade/builder.go:55`). FR-2.4 forbids applying a rename to a live character, so those caches always rebuild *after* the rename lands. **Load-bearing on FR-2.4**: relax the offline-only constraint and all four become required `NAME_CHANGED` consumers.
- **atlas-merchant** — `blacklists.name` and `merchant_visits.name` are name-*keyed* rows. Rewriting a blacklist entry on rename changes a moderation feature's behaviour ("does a blocked player escape their block by renaming?" is a product question); `merchant_visits` is a log. (design §3.8, §10)
- **atlas-storage** — storage is keyed `(tenant, world, account)` and shared by every character the account owns in that world, so it never moves and a non-empty source storage does not block the transfer. The FR-4.7 pink-text warning is the entire mitigation. (design §1.5 / FR-4.6)
- **atlas-inventory** — no world column; the character-world update moves the inventory by construction. (design §1.5)
