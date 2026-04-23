# Merchant ↔ Channel Integration — Tasks

Last Updated: 2026-03-17

## Phase 1: Fix atlas-merchant Data Model ✅

- [x] **1.1** Add `worldId` (byte), `channelId` (byte), `instanceId` (uuid.UUID) to `shop.Model` with accessors
- [x] **1.2** Add `WorldId`, `ChannelId`, `InstanceId` columns to `shop.Entity` with GORM tags + migration
- [x] **1.3** Update `NewModel` / builder to accept and store worldId, channelId, instanceId
- [x] **1.4** Update `Make(entity)` to populate worldId, channelId, instanceId from entity
- [x] **1.5** Update `PLACE_SHOP` command handler to persist worldId, channelId from command envelope + instanceId from body
- [x] **1.6** Add `InstanceId` field to `CommandPlaceShopBody`
- [x] **1.7** Replace `getByMapId` with `getByField(worldId, channelId, mapId, instanceId)` in provider.go
- [x] **1.8** Add REST route: `GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/merchants`
- [x] **1.9** Update `StatusEventShopOpenedBody` to include `worldId`, `channelId`, `instanceId`
- [x] **1.10** Update Redis registry keys to include world/channel/instance scoping
- [x] **1.11** Update or deprecate existing `GET /merchants?mapId=` endpoint
- [x] **1.12** Build + test atlas-merchant

## Phase 2: Add MESSAGE_SENT Event ✅

- [x] **2.1** Add `StatusEventMessageSent` constant + `StatusEventMessageSentBody` struct (shopId, characterId, content, slot)
- [x] **2.2** Update `SEND_MESSAGE` handler to resolve visitor slot and emit `MESSAGE_SENT` event
- [x] **2.3** Build + test atlas-merchant

## Phase 3: InteractionChat Packet ✅

- [x] **3.1** Add `CharacterInteractionModeChat` and `CharacterInteractionModeChatThing` constants to interaction_writer_body.go
- [x] **3.2** Create `InteractionChat` struct (mode, chatThing, slot, message) in interaction_writer.go
- [x] **3.3** Implement `Encode` / `Decode` on `InteractionChat`
- [x] **3.4** Create `CharacterInteractionChatBody(slot, name, content)` factory — resolves CHAT + CHAT_THING modes, formats `"{name} : {content}"`
- [x] **3.5** Add round-trip test
- [x] **3.6** Build + test atlas-packet

## Phase 4: atlas-channel — Merchant Command Producers ✅

- [x] **4.1** Create `channel/kafka/message/merchant/kafka.go` — command envelope + constants
- [x] **4.2** Create `channel/merchant/producer.go` — 11 command provider functions:
  - [x] PlaceShopCommandProvider
  - [x] OpenShopCommandProvider
  - [x] CloseShopCommandProvider
  - [x] EnterMaintenanceCommandProvider
  - [x] ExitMaintenanceCommandProvider
  - [x] AddListingCommandProvider
  - [x] RemoveListingCommandProvider
  - [x] PurchaseBundleCommandProvider
  - [x] EnterShopCommandProvider
  - [x] ExitShopCommandProvider
  - [x] SendMessageCommandProvider
- [x] **4.3** Create `channel/merchant/processor.go` — Processor interface + dispatch methods
- [x] **4.4** Build atlas-channel (compile check)

## Phase 5: atlas-channel — Wire Socket Handlers ✅

- [x] **5.1** Wire `CREATE` (0x00) + MerchantShopMiniRoomType → `PLACE_SHOP`
- [x] **5.2** Wire `VISIT` (0x04) + merchant context → `ENTER_SHOP`
- [x] **5.3** Wire `OPEN` (0x0B) + merchant context → `OPEN_SHOP`
- [x] **5.4** Wire `EXIT` (0x0A) → `EXIT_SHOP` (visitor) or `CLOSE_SHOP` (owner)
- [x] **5.5** Wire `CHAT` (0x06) + merchant context → `SEND_MESSAGE`
- [x] **5.6** Wire `MERCHANT_PUT_ITEM` (0x21) → `ADD_LISTING`
- [x] **5.7** Wire `MERCHANT_BUY` (0x22) → `PURCHASE_BUNDLE`
- [x] **5.8** Wire `MERCHANT_REMOVE_ITEM` (0x26) → `REMOVE_LISTING`
- [x] **5.9** Wire `MERCHANT_MAINTENANCE_OFF` (0x27) → `EXIT_MAINTENANCE`
- [x] **5.10** Wire `MERCHANT_EXIT` (0x29) → `EXIT_SHOP`
- [x] **5.11** Handle `ENTER_MAINTENANCE` — owner enters via `CASH_TRADE_OPEN` nProc=4 + MerchantShopMiniRoomType
- [x] **5.12** Build + test atlas-channel

## Phase 6: atlas-channel — Merchant Event Consumers ✅

- [x] **6.1** Create `channel/kafka/message/merchant/kafka.go` — status + listing event structs
- [x] **6.2** Create `channel/kafka/consumer/merchant/consumer.go` — InitHandlers + InitConsumers
- [x] **6.3** Handle `SHOP_OPENED` → spawn MiniRoom for all characters on map
- [x] **6.4** Handle `SHOP_CLOSED` → despawn MiniRoom for owner
- [x] **6.5** Handle `VISITOR_ENTERED` → broadcast to shop viewers via REST shop query
- [x] **6.6** Handle `VISITOR_EXITED` → logged (broadcast deferred — needs full room state tracking)
- [x] **6.7** Handle `VISITOR_EJECTED` → InteractionEnterResultError to ejected player
- [x] **6.8** Handle `MAINTENANCE_ENTERED` → logged (broadcast deferred — needs full room state tracking)
- [x] **6.9** Handle `MAINTENANCE_EXITED` → logged (broadcast deferred — needs full room state tracking)
- [x] **6.10** Handle `LISTING_PURCHASED` → logged (broadcast deferred — needs full room state tracking)
- [x] **6.11** Handle `CAPACITY_FULL` → InteractionEnterResultError (FULL) to joiner
- [x] **6.12** Handle `PURCHASE_FAILED` → InteractionEnterResultError (UNABLE) to buyer
- [x] **6.13** Handle `FREDERICK_NOTIFICATION` → HiredMerchantOperation FreeFormNotice to character
- [x] **6.14** Handle `MESSAGE_SENT` → InteractionChat broadcast to shop viewers via REST shop query
- [x] **6.15** Register merchant consumer in main.go (status + listing topics)
- [x] **6.16** Build + test atlas-channel

## Phase 7: atlas-channel — Field Entry Shop Spawning ✅

- [x] **7.1** Create `channel/merchant/rest.go` — RestModel + Extract
- [x] **7.2** Create `channel/merchant/requests.go` — requestInField with `/worlds/{w}/channels/{c}/maps/{m}/instances/{i}/merchants`
- [x] **7.3** Create `channel/merchant/model.go` — local shop model
- [x] **7.4** Add `ForEachInField(f, operator)` to merchant processor
- [x] **7.5** Create `spawnMerchantShopsForSession()` in map consumer
- [x] **7.6** Add goroutine in `enterMap()` for merchant shop spawning
- [x] **7.7** Build + test atlas-channel
- [x] **7.8** Docker build verification for atlas-merchant + atlas-channel

## Deferred

- [ ] `MERCHANT_ORGANIZE` (0x28) — no backend command
- [ ] `MERCHANT_WITHDRAW_MESO` (0x2B) — no backend command
- [ ] `HiredMerchantOperationHandleFunc` — cash shop item trigger, needs IDA
- [ ] Name change (0x2D) — no backend command
- [ ] Blacklist operations (0x30, 0x31) — no backend model
- [ ] Visit list viewing (0x2E) — no backend query
- [ ] Black list viewing (0x2F) — no backend model
- [ ] `UPDATE_LISTING` command — no client trigger identified
- [ ] `RETRIEVE_FREDERICK` command — deferred with HiredMerchantOperation

## Future Work

- [ ] Full mini-room session tracking in atlas-channel for proper broadcast to all room viewers
- [ ] VISITOR_EXITED broadcast to remaining viewers
- [ ] MAINTENANCE_ENTERED/EXITED broadcast to room viewers
- [ ] LISTING_PURCHASED listing refresh broadcast to room viewers
- [ ] SHOP_CLOSED despawn broadcast to all field characters (currently owner only)
