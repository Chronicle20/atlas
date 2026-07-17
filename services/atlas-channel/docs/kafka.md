# Kafka Documentation

## Topics Consumed

### EVENT_TOPIC_ACCOUNT_STATUS
- Direction: Event
- Message Type: `StatusEvent`
- Type Discriminators: `LOGGED_IN`, `LOGGED_OUT`
- Purpose: Receives account login/logout status events to maintain account registry

### EVENT_TOPIC_ACCOUNT_SESSION_STATUS
- Direction: Event
- Message Type: `StatusEvent[StateChangedEventBody]`, `StatusEvent[ErrorStatusEventBody]`
- Purpose: Receives session state changes for player login, channel changes, and error handling

### EVENT_TOPIC_ASSET_STATUS
- Direction: Event
- Message Type: `StatusEvent[CreatedStatusEventBody]`, `StatusEvent[UpdatedStatusEventBody]`, `StatusEvent[QuantityChangedEventBody]`, `StatusEvent[MovedStatusEventBody]`, `StatusEvent[DeletedStatusEventBody]`, `StatusEvent[AcceptedStatusEventBody]`, `StatusEvent[ReleasedStatusEventBody]`, `StatusEvent[ExpiredStatusEventBody]`
- Envelope: `StatusEvent[E]` with fields: CharacterId (uint32), CompartmentId (uuid.UUID), AssetId (uint32), TemplateId (uint32), Slot (int16), Type (string), Body (E)
- Purpose: Receives inventory asset lifecycle events. CREATED/ACCEPTED add items to client UI, UPDATED refreshes item display, QUANTITY_CHANGED updates stack counts, MOVED repositions items (with appearance update for equip changes), DELETED/RELEASED remove items from client UI, EXPIRED sends expiration notifications (general, cash, or replacement messages)

### EVENT_TOPIC_BUDDY_LIST_STATUS
- Direction: Event
- Message Type: `StatusEvent[BuddyAddedBody]`, `StatusEvent[BuddyRemovedBody]`, `StatusEvent[BuddyUpdatedBody]`, `StatusEvent[BuddyChannelChangeBody]`, `StatusEvent[CapacityChangeBody]`, `StatusEvent[ErrorBody]`
- Type Discriminators: `BUDDY_ADDED`, `BUDDY_REMOVED`, `BUDDY_UPDATED`, `BUDDY_CHANNEL_CHANGE`, `CAPACITY_CHANGE`, `ERROR`
- Error Codes: BUDDY_LIST_FULL, OTHER_BUDDY_LIST_FULL, ALREADY_BUDDY, CANNOT_BUDDY_GM, CHARACTER_NOT_FOUND, UNKNOWN_ERROR
- Purpose: Receives buddy list operation results and state changes

### EVENT_TOPIC_CASH_COMPARTMENT_STATUS
- Direction: Event
- Message Type: `StatusEvent[StatusEventAcceptedBody]`, `StatusEvent[StatusEventReleasedBody]`
- Envelope: `StatusEvent[E]` with fields: AccountId (uint32), CharacterId (uint32), CompartmentId (uuid.UUID), CompartmentType (byte), Type (string), Body (E)
- Purpose: Receives cash shop compartment transfer events. ACCEPTED (item moved from character to cash shop) fetches the new cash-shop asset and notifies the client. RELEASED (item moved from cash shop to character) looks up the asset by CashId in the character's inventory and notifies the client.

### EVENT_TOPIC_CASH_SHOP_STATUS
- Direction: Event
- Message Type: Cash shop status events
- Type Discriminators: `CHARACTER_ENTER`, `CHARACTER_EXIT`, `INVENTORY_CAPACITY_INCREASED`, `PURCHASE`, `ERROR`, `CASH_ITEM_MOVED_TO_INVENTORY`
- Purpose: Receives cash shop operation results

### EVENT_TOPIC_CHARACTER_BUFF_STATUS
- Direction: Event
- Message Type: `StatusEvent[AppliedStatusEventBody]`, `StatusEvent[ExpiredStatusEventBody]`
- Envelope: `StatusEvent[E]` with fields: WorldId (world.Id), CharacterId (uint32), Type (string), Body (E)
- Purpose: Receives character buff applied and expired events. APPLIED sends buff give packet to character and foreign buff packet to map. EXPIRED sends buff cancel packet to character and foreign cancel to map. Body contains SourceId (int32), Level (byte), Duration (int32), Changes ([]StatChange with Type string and Amount int32), CreatedAt, ExpiresAt.

### EVENT_TOPIC_CHAIR_STATUS
- Direction: Event
- Message Type: Chair status events
- Type Discriminators: `USED`, `ERROR`, `CANCELLED`
- Envelope fields: worldId, channelId, mapId, instance, chairType, chairId
- Error Type: INTERNAL
- Purpose: Receives chair sit/stand events

### EVENT_TOPIC_CHALKBOARD_STATUS
- Direction: Event
- Message Type: Chalkboard status events
- Type Discriminators: `SET`, `CLEAR`
- Purpose: Receives chalkboard update events

### EVENT_TOPIC_CHARACTER_CHAT
- Direction: Event
- Message Type: `ChatEvent[GeneralChatBody]`, `ChatEvent[MultiChatBody]`, `ChatEvent[WhisperChatBody]`, `ChatEvent[MessengerChatBody]`, `ChatEvent[PetChatBody]`, `ChatEvent[PinkTextChatBody]`
- Envelope: `ChatEvent[E]` with fields: WorldId (world.Id), ChannelId (channel.Id), MapId (_map.Id), Instance (uuid.UUID), ActorId (uint32), Message (string), Type (string), Body (E)
- Purpose: Receives character chat messages for broadcast. GENERAL broadcasts to map sessions (with BalloonOnly flag). BUDDY/PARTY/GUILD/ALLIANCE (multi-chat types) deliver to specified recipients. WHISPER delivers to target character. MESSENGER delivers to messenger room recipients. PET broadcasts pet chat to map sessions. PINK_TEXT delivers to specified recipients.

### EVENT_TOPIC_CHARACTER_STATUS
- Direction: Event
- Message Type: `StatusEvent[StatusEventStatChangedBody]`, `StatusEvent[StatusEventMapChangedBody]`, `StatusEvent[ExperienceChangedStatusEventBody]`, `StatusEvent[FameChangedStatusEventBody]`, `StatusEvent[MesoChangedStatusEventBody]`, `StatusEvent[LevelChangedStatusEventBody]`
- Type Discriminators: `STAT_CHANGED`, `MAP_CHANGED`, `JOB_CHANGED`, `EXPERIENCE_CHANGED`, `LEVEL_CHANGED`, `MESO_CHANGED`, `FAME_CHANGED`
- Experience Distribution Types: WHITE, YELLOW, CHAT, MONSTER_BOOK, MONSTER_EVENT, PLAYTIME, WEDDING, SPIRIT_WEEK, PARTY, ITEM, INTERNET_CAFE, RAINBOW_WEEK, PARTY_RING, CAKE_PIE
- Purpose: Receives character stat, map, experience, fame, meso, and level change events

### EVENT_TOPIC_COMPARTMENT_STATUS
- Direction: Event
- Message Type: `StatusEvent[ReservationCancelledEventBody]`, `StatusEvent[MergeCompleteEventBody]`, `StatusEvent[SortCompleteEventBody]`
- Envelope: `StatusEvent[E]` with fields: CharacterId (uint32), CompartmentId (uuid.UUID), Type (string), Body (E)
- Purpose: Receives compartment-level events. RESERVATION_CANCELLED re-enables client actions. MERGE_COMPLETE and SORT_COMPLETE trigger client inventory refresh for the affected type and re-enable actions.

### EVENT_TOPIC_CONSUMABLE_STATUS
- Direction: Event
- Message Type: Consumable events
- Type Discriminators: `ERROR`, `SCROLL`
- Error Type: PET_CANNOT_CONSUME
- Purpose: Receives consumable item use events

### EVENT_TOPIC_CONVERSATION_REWARD_NOTICE
- Direction: Event
- Message Type: `EventBody`
- Type Discriminators: `item_gain`, `item_loss`
- Envelope Fields: characterId, kind, itemId, quantity
- Purpose: Receives item-gain/item-loss notices produced by atlas-saga-orchestrator when a conversation-sourced AwardAsset/DestroyAsset/DestroyAssetFromSlot step completes with ShowEffect=true. Renders the gain effect (quest-reward-style character effect) or the loss chat line on the target session; no-ops when the character session is not present on this channel.

### EVENT_TOPIC_DOOR_STATUS
- Direction: Event
- Message Type: `StatusEvent[E]` with envelope fields worldId, channelId, mapId (area field), instance, pairId, ownerCharacterId, partyId, forCharacterId, type, body
- Type Discriminators: `CREATED`, `REMOVED`, `SLOT_CHANGED`
- Body Types: `CreatedBody` (areaDoorId, townDoorId, townMapId, slot, townPortalId, areaX, areaY, townX, townY, skillId, skillLevel, expiresAt), `RemovedBody` (areaDoorId, townDoorId, townMapId, slot, reason), `SlotChangedBody` (areaDoorId, townDoorId, townMapId, oldSlot, newSlot, townPortalId, townX, townY, areaX, areaY)
- Remove Reasons (RemovedBody.reason): `EXPIRY`, `LOGOUT`, `CHANNEL_CHANGED`, `LEFT_FIELD`, `RECAST`, `PARTY_LEFT`, `CANCELLED`
- Header Parsers: span, tenant
- Start Offset: last
- Purpose: Receives Mystic Door create/remove/slot-change events; channel renders the door to area and town sessions and updates the party town-portal array

### EVENT_TOPIC_DROP_STATUS
- Direction: Event
- Message Type: Drop status events
- Type Discriminators: `CREATED`, `EXPIRED`, `PICKED_UP`, `CONSUMED`
- Body Fields: itemId, quantity, meso, type, x, y, ownerId, ownerPartyId, dropTime, dropperUniqueId, playerDrop
- Purpose: Receives drop spawn/pickup events

### EVENT_TOPIC_EXPRESSION
- Direction: Event
- Message Type: `Event`
- Envelope Fields: characterId, worldId, channelId, mapId, instance, expression
- Purpose: Receives character expression (emote) events

### EVENT_TOPIC_FAME_STATUS
- Direction: Event
- Message Type: Fame status events
- Type Discriminators: `ERROR`
- Error Types: NOT_TODAY, NOT_THIS_MONTH, INVALID_NAME, NOT_MINIMUM_LEVEL, UNEXPECTED
- Purpose: Receives fame change events

### EVENT_TOPIC_GACHAPON_REWARD_WON
- Direction: Event
- Message Type: `RewardWonEvent` with fields: CharacterId (uint32), WorldId (byte), ItemId (uint32), Quantity (uint32), Tier (string), GachaponId (string), GachaponName (string), AssetId (uint32)
- Purpose: Receives gachapon reward win events. Looks up the asset by AssetId in the character's inventory compartment and broadcasts a world megaphone message.

### EVENT_TOPIC_GUILD_STATUS
- Direction: Event
- Message Type: Guild status events
- Type Discriminators: `REQUEST_AGREEMENT`, `CREATED`, `DISBANDED`, `EMBLEM_UPDATED`, `MEMBER_STATUS_UPDATED`, `MEMBER_TITLE_UPDATED`, `MEMBER_LEFT`, `MEMBER_JOINED`, `NOTICE_UPDATED`, `CAPACITY_UPDATED`, `TITLES_UPDATED`, `ERROR`
- Purpose: Receives guild lifecycle and membership events

### EVENT_TOPIC_GUILD_THREAD_STATUS
- Direction: Event
- Message Type: Guild thread status events
- Type Discriminators: `CREATED`, `UPDATED`, `DELETED`, `REPLY_ADDED`, `REPLY_DELETED`
- Purpose: Receives guild BBS thread operation results

### EVENT_TOPIC_INSTANCE_TRANSPORT
- Direction: Event
- Message Type: `Event[TransitEnteredEventBody]`
- Envelope: `Event[E]` with fields: TransactionId (uuid.UUID), WorldId (world.Id), CharacterId (uint32), Type (string), Body (E)
- Purpose: Receives instance transport events. TRANSIT_ENTERED sends Clock packet and optional ScriptProgress message to character. Body contains RouteId (uuid.UUID), InstanceId (uuid.UUID), ChannelId (channel.Id), DurationSeconds (uint32), Message (string).

### EVENT_TOPIC_INVITE_STATUS
- Direction: Event
- Message Type: Invite status events
- Type Discriminators: `CREATED`, `ACCEPTED`, `REJECTED`
- Invite Types: PARTY, BUDDY, GUILD, MESSENGER
- Purpose: Receives invite operation results

### EVENT_TOPIC_MAP_STATUS
- Direction: Event
- Message Type: `StatusEvent[CharacterEnter]`, `StatusEvent[CharacterExit]`, `StatusEvent[WeatherStart]`, `StatusEvent[WeatherEnd]`, `StatusEvent[MapTimerStarted]`
- Envelope Fields: transactionId, worldId, channelId, mapId, instance
- Type Discriminators: `CHARACTER_ENTER`, `CHARACTER_EXIT`, `WEATHER_START`, `WEATHER_END`, `MAP_TIMER_STARTED`
- Purpose: Receives character map entry/exit, weather start/end, and map timer started events. MAP_TIMER_STARTED body contains CharacterId (uint32) and Seconds (uint32); the handler targets the single character via `IfPresentByCharacterId` and sends a `ClockWriter` packet built from `NewTimerClock(seconds)`.

### EVENT_TOPIC_MEGAPHONE
- Direction: Event
- Message Type: `BroadcastEvent`
- Type Discriminators (Tier): `MEGAPHONE`, `SUPER`, `ITEM`, `TRIPLE`
- Scope Values: `CHANNEL`, `WORLD`
- Start Offset: last
- Purpose: Receives megaphone broadcast events for the stateless megaphone tiers. CHANNEL scope renders the WorldMessage only to the sender's channel; any other scope (WORLD) renders to every channel in the world. Builds the tier-specific WorldMessage mode body (MEGAPHONE/SUPER/ITEM/TRIPLE, sender name decorated with medal) and announces it via `WorldMessageWriter` to every session on the pod's channel.

### EVENT_TOPIC_MERCHANT_STATUS
- Direction: Event
- Message Type: `StatusEvent[StatusEventShopOpenedBody]`, `StatusEvent[StatusEventShopClosedBody]`, `StatusEvent[StatusEventVisitorBody]`, `StatusEvent[StatusEventCapacityFullBody]`, `StatusEvent[StatusEventShopCreateFailedBody]`, `StatusEvent[StatusEventPurchaseFailedBody]`, `StatusEvent[StatusEventFrederickNotificationBody]`, `StatusEvent[StatusEventMessageSentBody]`, `StatusEvent[StatusEventShopUpdatedBody]`, `StatusEvent[StatusEventEnterFailedBody]`
- Envelope: `StatusEvent[E]` with fields: CharacterId (uint32), Type (string), Body (E)
- Type Discriminators (with registered handlers): `SHOP_OPENED`, `SHOP_SETUP`, `SHOP_CLOSED`, `MAINTENANCE_ENTERED`, `MAINTENANCE_EXITED`, `VISITOR_ENTERED`, `VISITOR_EXITED`, `VISITOR_EJECTED`, `CAPACITY_FULL`, `SHOP_CREATE_FAILED`, `PURCHASE_FAILED`, `FREDERICK_NOTIFICATION`, `MESSAGE_SENT`, `SHOP_UPDATED`, `ENTER_FAILED`
- Purpose: Receives personal shop / hired merchant status events. SHOP_SETUP sends the freshly created (Draft) shop's editing room to the owner without spawning the map object. SHOP_OPENED spawns the map object at go-live (personal shop: a mini-room box on the owner's avatar with a permit-derived sign skin; hired merchant: a standalone employee NPC). SHOP_CLOSED despawns the map object. VISITOR_ENTERED sends the shop interior to the entering visitor and broadcasts an ENTER avatar to existing viewers. VISITOR_EXITED sends LEAVE to the exiting visitor and the remaining viewers. VISITOR_EJECTED sends LEAVE (with the `leaveReason` table key when present) to the ejected visitor. MAINTENANCE_ENTERED sends the shop interior to the owner; MAINTENANCE_EXITED closes the management UI for a hired merchant or refreshes the room for a personal shop. CAPACITY_FULL sends the enter error (mode full), or a SHOP_LINK "full" result when the arrival is a pending owl warp. SHOP_CREATE_FAILED maps a placement reason to a mini-room error mode. PURCHASE_FAILED sends the enter error (mode unable). FREDERICK_NOTIFICATION sends a hired-merchant free-form notice (a world-message notice on JMS). MESSAGE_SENT broadcasts chat to shop viewers. SHOP_UPDATED refreshes the shop view after withdraw/organize. ENTER_FAILED sends the faithful enter error for a rejected visitor (undergoing maintenance, room closed, or blacklisted).
- MAINTENANCE_ENTERED/MAINTENANCE_EXITED and VISITOR_ENTERED/VISITOR_EXITED/VISITOR_EJECTED share the `StatusEventVisitorBody` body (fields: shopId, characterId, slot, leaveReason). SHOP_SETUP shares the `StatusEventShopOpenedBody` body with SHOP_OPENED.
- StatusEventShopOpenedBody fields: shopId, shopType (byte), worldId, channelId, mapId, instanceId, title, x, y
- StatusEventShopClosedBody fields: shopId, closeReason (byte)
- StatusEventShopCreateFailedBody fields: worldId, channelId, reason (string; `TOO_CLOSE_TO_PORTAL`, `TOO_CLOSE_TO_SHOP`, `NOT_FREE_MARKET`, `UNABLE`)
- StatusEventPurchaseFailedBody fields: shopId, reason (string)
- StatusEventFrederickNotificationBody fields: daysSinceStorage (uint16)
- StatusEventMessageSentBody fields: shopId, characterId, slot (byte), content (string)
- StatusEventShopUpdatedBody fields: shopId
- StatusEventEnterFailedBody fields: shopId, reason (string; `UNDERGOING_MAINTENANCE`, `ROOM_CLOSED`, `BLACKLISTED`)
- Note: `BLACKLIST_UPDATED` (`StatusEventBlacklistUpdatedBody`) is defined on the topic but has no registered handler in the channel service.
- Header Parsers: span, tenant
- Start Offset: last

### EVENT_TOPIC_MERCHANT_LISTING
- Direction: Event
- Message Type: `ListingEvent[ListingEventPurchasedBody]`
- Envelope: `ListingEvent[E]` with fields: ShopId (string), Type (string), Body (E)
- Type Discriminators: `LISTING_PURCHASED`
- Body Fields (ListingEventPurchasedBody): listingIndex (uint16), buyerCharacterId (uint32), bundleCount (uint16), bundlesRemaining (uint16)
- Purpose: Receives merchant listing purchase events. LISTING_PURCHASED re-fetches the shop and sends the shop-view refresh to the owner and all visitors; for a personal shop it also notifies the owner of the sold item (per-session sold-item ledger).
- Header Parsers: span, tenant
- Start Offset: last

### EVENT_TOPIC_MESSENGER_STATUS
- Direction: Event
- Message Type: Messenger status events
- Type Discriminators: `CREATED`, `JOINED`, `LEFT`, `ERROR`
- Purpose: Receives messenger room operation results

### EVENT_TOPIC_MIST
- Direction: Event
- Message Type: `Event[CreatedBody]`, `Event[DestroyedBody]`
- Envelope: `Event[E]` with fields: Tenant (uuid.UUID), WorldId (world.Id), ChannelId (channel.Id), MapId (_map.Id), Instance (uuid.UUID), MistId (uuid.UUID), Type (string), Body (E)
- Type Discriminators: `MIST_CREATED`, `MIST_DESTROYED`
- Body Fields: CreatedBody (ownerType, ownerId, sourceSkillId, sourceSkillLevel, type, originX, originY, ltX, ltY, rbX, rbY, duration); DestroyedBody (reason)
- Header Parsers: span, tenant
- Start Offset: last
- Purpose: Receives mist/affected-area (skill-zone) create and destroy events from atlas-maps; broadcasts the corresponding AffectedAreaCreated/AffectedAreaRemoved packet to sessions in the field's map.

### EVENT_TOPIC_MONSTER_BOOK_STATUS
- Direction: Event
- Message Type: `StatusEvent[CardAddedBody]`, `StatusEvent[CoverChangedBody]`, `StatusEvent[StatsChangedBody]`
- Envelope: `StatusEvent[E]` with fields: TenantId (uuid.UUID), CharacterId (uint32), EventId (uuid.UUID), Type (string), Body (E)
- Type Discriminators: `CARD_ADDED`, `COVER_CHANGED`, `STATS_CHANGED`
- Purpose: Receives monster-book card-added, cover-changed, and collection-stats-changed events from atlas-monster-book.

### EVENT_TOPIC_MONSTER_STATUS
- Direction: Event
- Message Type: `StatusEvent[StatusEventCreatedBody]`, `StatusEvent[StatusEventDestroyedBody]`, `StatusEvent[StatusEventDamagedBody]`, `StatusEvent[StatusEventKilledBody]`, `StatusEvent[StatusEventStartControlBody]`, `StatusEvent[StatusEventStopControlBody]`, `StatusEvent[StatusEventAggroChangedBody]`, `StatusEvent[StatusEffectAppliedBody]`, `StatusEvent[StatusEffectExpiredBody]`, `StatusEvent[StatusEffectCancelledBody]`, `StatusEvent[StatusEventDamageReflectedBody]`
- Envelope: `StatusEvent[E]` with fields: WorldId (world.Id), ChannelId (channel.Id), MapId (_map.Id), Instance (uuid.UUID), UniqueId (uint32), MonsterId (uint32), Type (string), Body (E)
- Purpose: Receives monster lifecycle and status events. CREATED spawns monster visually. DESTROYED/KILLED despawn monster. START_CONTROL/STOP_CONTROL manage monster controller assignment; START_CONTROL's `controllerHasAggro` is read from the event body and passed to `StartControlMonsterBody`, which selects `ControlMonsterTypeActiveRequest` (true) or `ControlMonsterTypeActiveInit` (false) on the wire. AGGRO_CHANGED is consumed by `handleStatusEventAggroChanged`, which loads the monster via `monster.NewProcessor(l, ctx).GetById` and re-sends `MonsterControlWriter` to the controller's session with the new aggro state — no STOP_CONTROL is emitted to the client because the active/passive control type carries the state change. DAMAGED shows HP bar (boss=map-wide, else party-only) and, for `damageSource` values `MONSTER_ATTACK` or `DAMAGE_OVER_TIME`, also broadcasts a MonsterDamage packet. Player-inflicted (`CHARACTER_ATTACK`) damage is intentionally not echoed because the attack broadcast from the socket handler already renders the damage to observers; `HEAL` is a 0-damage HP-bar refresh and also skipped. STATUS_APPLIED sends MonsterStatSet packet. STATUS_EXPIRED/STATUS_CANCELLED send MonsterStatReset packet. DAMAGE_REFLECTED applies reflected damage to character HP.

### EVENT_TOPIC_MOUNT_STATUS
- Direction: Event
- Message Type: `StatusEvent[StatusEventBody]`
- Envelope: `StatusEvent[E]` with fields: WorldId (world.Id), CharacterId (uint32), Type (string), Body (E)
- Type Discriminators: `SET`, `TICK`, `FEED`
- Body Fields: level, exp, tiredness, levelUp, tooTired
- Purpose: Receives mount level/exp/tiredness change events from atlas-mounts. Broadcasts a SetTamingMobInfo packet to the rider's map (self + observers). When `tooTired` is set, auto-dismounts the rider (cancels the active mount-riding buff) and sends a world-message notice.

### EVENT_TOPIC_MTS_STATUS
- Direction: Event
- Message Type: `StatusEvent[E]` with envelope fields WorldId, SellerId/BuyerId/CharacterId (per body), Type, Body
- Type Discriminators: `LISTING_CREATED`, `LISTING_CANCELLED`, `ITEM_TAKEN_HOME`, `LISTING_CREATE_FAILED`, `LISTING_CANCEL_FAILED`, `TAKE_HOME_FAILED`, `LISTING_SOLD`, `BUY_FAILED`, `BID_FAILED`, `BID_PLACED`, `OUTBID`, `WISH_ADDED`, `WISH_REMOVED`
- Purpose: Receives MTS marketplace lifecycle events from atlas-mts. Writes the matching clientbound MtsOperation result to the originating character's session (resolved by the event's seller/buyer/character id).

### EVENT_TOPIC_NOTE_STATUS
- Direction: Event
- Message Type: Note status events
- Type Discriminators: `CREATED`, `UPDATED`, `DELETED`
- Purpose: Receives note send/receive events

### EVENT_TOPIC_NPC_SHOP_STATUS
- Direction: Event
- Message Type: NPC shop status events
- Type Discriminators: `ENTERED`, `EXITED`, `ERROR`
- Error Fields: levelLimit, reason
- Purpose: Receives NPC shop enter/exit/error events

### EVENT_TOPIC_PARTY_STATUS
- Direction: Event
- Message Type: Party status events
- Type Discriminators: `CREATED`, `JOINED`, `LEFT`, `EXPEL`, `DISBAND`, `CHANGE_LEADER`, `ERROR`
- Purpose: Receives party lifecycle events

### EVENT_TOPIC_PARTY_MEMBER_STATUS
- Direction: Event
- Message Type: Party member status events
- Type Discriminators: `LOGIN`, `LOGOUT`
- Purpose: Receives party member login/logout tracking events

### EVENT_TOPIC_PARTY_QUEST_STATUS
- Direction: Event
- Message Type: Party quest status events
- Type Discriminators: `STAGE_CLEARED`, `CHARACTER_LEFT`
- Body Fields: stageIndex, channelId, mapIds, fieldInstances
- Purpose: Receives party quest progress events

### EVENT_TOPIC_PET_STATUS
- Direction: Event
- Message Type: `StatusEvent[SpawnedStatusEventBody]`, `StatusEvent[DespawnedStatusEventBody]`, `StatusEvent[CommandResponseStatusEventBody]`, `StatusEvent[ClosenessChangedStatusEventBody]`, `StatusEvent[FullnessChangedStatusEventBody]`, `StatusEvent[LevelChangedStatusEventBody]`, `StatusEvent[SlotChangedStatusEventBody]`, `StatusEvent[ExcludeChangedStatusEventBody]`
- Envelope: `StatusEvent[E]` with fields: PetId (uint32), OwnerId (uint32), Type (string), Body (E)
- Purpose: Receives pet lifecycle and stat change events. SPAWNED/DESPAWNED manage pet visibility. COMMAND_RESPONSE handles pet interaction results. CLOSENESS_CHANGED and FULLNESS_CHANGED refresh the pet's cash inventory asset. LEVEL_CHANGED triggers level-up effects. SLOT_CHANGED updates pet stat and position. EXCLUDE_CHANGED updates pet exclude list.

### EVENT_TOPIC_QUEST_STATUS
- Direction: Event
- Message Type: `StatusEvent[QuestStartedEventBody]`, `StatusEvent[QuestCompletedEventBody]`, `StatusEvent[QuestForfeitedEventBody]`, `StatusEvent[QuestProgressUpdatedEventBody]`
- Purpose: Receives quest state change events

### EVENT_TOPIC_REACTOR_STATUS
- Direction: Event
- Message Type: `StatusEvent[CreatedStatusEventBody]`, `StatusEvent[DestroyedStatusEventBody]`, `StatusEvent[HitStatusEventBody]`
- Body Fields: classification, name, state, eventState, delay, direction, x, y, updateTime
- Purpose: Receives reactor spawn, destroy, and hit events

### EVENT_TOPIC_SAGA_STATUS
- Direction: Event
- Message Type: `StatusEvent[StatusEventCompletedBody]`, `StatusEvent[StatusEventFailedBody]`
- Saga Types: storage_operation
- Error Codes: NOT_ENOUGH_MESOS, INVENTORY_FULL, STORAGE_FULL, UNKNOWN
- Purpose: Receives saga transaction completion and failure events

### EVENT_TOPIC_SESSION_STATUS
- Direction: Event
- Message Type: `StatusEvent`
- Type Discriminators: `CREATED`, `DESTROYED`
- Issuers: LOGIN, CHANNEL
- Purpose: Receives session created/destroyed events

### EVENT_TOPIC_SKILL_STATUS
- Direction: Event
- Message Type: `StatusEvent[StatusEventCreatedBody]`, `StatusEvent[StatusEventUpdatedBody]`, `StatusEvent[StatusEventCooldownAppliedBody]`, `StatusEvent[StatusEventCooldownExpiredBody]`
- Envelope: `StatusEvent[E]` with fields: CharacterId (uint32), SkillId (uint32), Type (string), Body (E)
- Purpose: Receives skill lifecycle events. CREATED/UPDATED send skill change packets to character. COOLDOWN_APPLIED sends cooldown start packet. COOLDOWN_EXPIRED sends cooldown end packet.

### EVENT_TOPIC_STORAGE_STATUS
- Direction: Event
- Message Type: `StatusEvent[MesosUpdatedEventBody]`, `StatusEvent[ArrangedEventBody]`, `StatusEvent[ErrorEventBody]`, `StatusEvent[ProjectionCreatedEventBody]`
- Error Codes: STORAGE_FULL, NOT_ENOUGH_MESOS, ONE_OF_A_KIND, GENERIC
- Purpose: Receives storage operation results. MESOS_UPDATED sends updated meso count to client. ARRANGED refreshes the full storage view. ERROR maps error codes to client error messages. PROJECTION_CREATED fetches projection data and displays the storage UI.

### EVENT_TOPIC_STORAGE_COMPARTMENT_STATUS
- Direction: Event
- Message Type: `StorageCompartmentEvent[CompartmentAcceptedEventBody]`, `StorageCompartmentEvent[CompartmentReleasedEventBody]`
- Purpose: Receives storage compartment deposit/withdraw events. ACCEPTED (item deposited into storage) sends updated storage assets for the affected inventory type. RELEASED (item withdrawn from storage) sends updated storage assets for the affected inventory type. Both use projection data when available, falling back to direct storage data.

### EVENT_TOPIC_SUMMON_STATUS
- Direction: Event
- Message Type: `StatusEvent[StatusEventCreatedBody]`, `StatusEvent[StatusEventMovedBody]`, `StatusEvent[StatusEventAttackedBody]`, `StatusEvent[StatusEventDamagedBody]`, `StatusEvent[StatusEventDestroyedBody]`, `StatusEvent[E]` (SKILL)
- Envelope: `StatusEvent[E]` with fields: WorldId (world.Id), ChannelId (channel.Id), MapId (_map.Id), Instance (uuid.UUID), SummonId (uint32), OwnerCharacterId (uint32), SkillId (uint32), Type (string), Body (E)
- Type Discriminators: `CREATED`, `MOVED`, `ATTACKED`, `DAMAGED`, `DESTROYED`, `SKILL`
- Purpose: Receives summon (puppet/Beholder/etc.) lifecycle and movement events from atlas-summons; broadcasts the corresponding summon spawn/move/attack/damage/remove packet to map sessions.

### EVENT_TOPIC_TRANSPORT_STATUS
- Direction: Event
- Message Type: `StatusEvent[ArrivedStatusEventBody]`, `StatusEvent[DepartedStatusEventBody]`
- Envelope: `StatusEvent[E]` with fields: RouteId (uuid.UUID), Type (string), Body (E)
- Purpose: Receives transport route state changes. ARRIVED/DEPARTED indicate route arrival at or departure from a map.

### EVENT_TOPIC_WALLET_STATUS
- Direction: Event
- Message Type: `StatusEvent[StatusEventUpdatedBody]`
- Envelope: `StatusEvent[E]` with fields: AccountId (uint32), Type (string), Body (E)
- Type Discriminators: `UPDATED`
- Body Fields: credit, points, prepaid, transactionId
- Purpose: Receives NX wallet balance-updated events. Refreshes the on-screen NX/points counter for whichever cash scene (MTS or the regular cash shop) the character currently occupies, using the balances carried on the event itself; a character in neither scene has nothing refreshed.

### EVENT_TOPIC_WORLD_BROADCAST_STATUS
- Direction: Event
- Message Type: `StatusEvent` (`worldbroadcast` message package; a flat struct, not the generic `StatusEvent[E]` envelope used elsewhere on this topic list)
- Type Discriminators (Type): `QUEUED`, `STARTED`, `ENDED`
- Family Values: `TV`, `AVATAR`
- Start Offset: last
- Purpose: Receives world-broadcast queue status events for the Maple TV and avatar megaphone families, gated by `sc.IsWorld` (fans out to every channel in the world). QUEUED sends a `TvSendMessageResultWriter` success ack to the sender, TV family only. STARTED renders and announces to every session on the pod's channel: `TvSetMessageWriter` (TV family; message type resolved from the tenant `messageTypes` writer table) or the SetAvatarMegaphone writer (AVATAR family). ENDED announces the TvClearMessage or ClearAvatarMegaphone writer to every session on the pod's channel.

### COMMAND_TOPIC_CHANNEL_STATUS
- Direction: Command (inbound)
- Message Type: Channel status commands
- Type Discriminators: `STATUS_REQUEST`
- Purpose: Receives channel status request commands

### COMMAND_TOPIC_GUILD
- Direction: Command (inbound)
- Message Type: Guild commands
- Type Discriminators: `REQUEST_NAME`, `REQUEST_EMBLEM`
- Purpose: Receives guild operation commands that require socket interaction (name input, emblem display)

### COMMAND_TOPIC_SYSTEM_MESSAGE
- Direction: Command (inbound)
- Message Type: `Command[SendMessageBody]`, `Command[PlayPortalSoundBody]`, `Command[ShowInfoBody]`, `Command[ShowInfoTextBody]`, `Command[UpdateAreaInfoBody]`, `Command[ShowHintBody]`, `Command[ShowGuideHintBody]`, `Command[ShowIntroBody]`, `Command[FieldEffectBody]`, `Command[UiLockBody]`, `Command[UiDisableBody]`
- Envelope: `Command[E]` with fields: TransactionId (uuid.UUID), WorldId (world.Id), ChannelId (channel.Id), CharacterId (uint32), Type (string), Body (E)
- Purpose: Receives system message commands from other services. SEND_MESSAGE displays notices/popups/pink/blue text. PLAY_PORTAL_SOUND plays portal sound effect. SHOW_INFO/SHOW_INFO_TEXT display info paths or text. UPDATE_AREA_INFO updates area-specific info. SHOW_HINT displays hint overlay. SHOW_GUIDE_HINT displays guide hint. SHOW_INTRO displays intro sequence. FIELD_EFFECT triggers field visual effect. UI_LOCK/UI_DISABLE control UI state.
- Message Types for SEND_MESSAGE: NOTICE, POP_UP, PINK_TEXT, BLUE_TEXT

---

## Topics Produced

### COMMAND_TOPIC_ACCOUNT_SESSION
- Direction: Command
- Message Type: `Command[ProgressStateCommandBody]`, `Command[LogoutCommandBody]`
- Purpose: Issues session state progression and logout commands

### COMMAND_TOPIC_BUDDY_LIST
- Direction: Command
- Message Type: `Command[RequestAddBody]`, `Command[RequestDeleteBody]`
- Purpose: Issues buddy list add and delete request commands

### COMMAND_TOPIC_CASH_SHOP
- Direction: Command
- Message Type: Cash shop commands
- Type Discriminators: REQUEST_PURCHASE, REQUEST_INVENTORY_INCREASE_BY_TYPE, REQUEST_INVENTORY_INCREASE_BY_ITEM, REQUEST_STORAGE_INCREASE, REQUEST_STORAGE_INCREASE_BY_ITEM, REQUEST_CHARACTER_SLOT_INCREASE_BY_ITEM, MOVE_FROM_CASH_INVENTORY, MOVE_TO_CASH_INVENTORY
- Purpose: Issues cash shop operation commands

### COMMAND_TOPIC_CHAIR
- Direction: Command
- Message Type: `Command[UseBody]`, `Command[CancelBody]`
- Envelope Fields: worldId, channelId, mapId, instance, chairType, chairId
- Purpose: Issues chair sit/stand commands

### COMMAND_TOPIC_CHALKBOARD
- Direction: Command
- Message Type: `Command[SetBody]`, `Command[ClearBody]`
- Purpose: Issues chalkboard update/close commands

### COMMAND_TOPIC_CHANNEL_STATUS
- Direction: Command
- Message Type: Channel status commands
- Purpose: Issues channel heartbeat and status commands

### COMMAND_TOPIC_CHARACTER
- Direction: Command
- Message Type: `Command[RequestDistributeApCommandBody]`, `Command[RequestDistributeSpCommandBody]`, `Command[RequestDropMesoCommandBody]`, `Command[ChangeHPCommandBody]`, `Command[ChangeMPCommandBody]`
- Purpose: Issues character stat distribution, meso drop, and HP/MP change commands

### COMMAND_TOPIC_CHARACTER_BUFF
- Direction: Command
- Message Type: `Command[ApplyBody]`, `Command[CancelBody]`
- Envelope Fields: worldId, channelId, mapId, instance, characterId
- Purpose: Issues buff application/removal commands

### COMMAND_TOPIC_CHARACTER_CHAT
- Direction: Command
- Message Type: `Command[GeneralChatBody]`, `Command[MultiChatBody]`, `Command[WhisperChatBody]`, `Command[MessengerChatBody]`, `Command[PetChatBody]`
- Purpose: Issues character chat commands. GENERAL (field-scoped with BalloonOnly flag), BUDDY/PARTY/GUILD/ALLIANCE (multi-chat with Recipients list), WHISPER (with RecipientName), MESSENGER (with Recipients list), PET (with OwnerId, PetSlot, Type, Action, Balloon)

### COMMAND_TOPIC_CHARACTER_MOVEMENT
- Direction: Command
- Message Type: `Command`
- Envelope Fields: worldId, channelId, mapId, instance, objectId, observerId, x, y, stance
- Purpose: Issues character movement commands

### COMMAND_TOPIC_COMPARTMENT
- Direction: Command
- Message Type: `Command[EquipCommandBody]`, `Command[UnequipCommandBody]`, `Command[MoveCommandBody]`, `Command[DropCommandBody]`, `Command[MergeCommandBody]`, `Command[SortCommandBody]`
- Envelope: `Command[E]` with fields: CharacterId (uint32), InventoryType (byte), Type (string), Body (E)
- Purpose: Issues inventory compartment operation commands. EQUIP/UNEQUIP handle equipment slot changes with source/destination. MOVE repositions items within a compartment. DROP drops items to the map (includes field coordinates and quantity). MERGE and SORT reorganize compartment contents.

### COMMAND_TOPIC_CONSUMABLE
- Direction: Command
- Message Type: `Command[RequestItemConsumeBody]`, `Command[RequestScrollBody]`
- Purpose: Issues consumable item use commands. REQUEST_ITEM_CONSUME uses a consumable item (Source, ItemId, Quantity). REQUEST_SCROLL applies a scroll to equipment (ScrollSlot, EquipSlot, WhiteScroll, LegendarySpirit).

### COMMAND_TOPIC_DOOR
- Direction: Command
- Message Type: `Command[E]` with envelope fields worldId, channelId, mapId, instance, ownerCharacterId, type, body
- Type Discriminators: `SPAWN`, `REMOVE`
- Body Types: `SpawnBody` (skillId, skillLevel, x, y), `RemoveBody` (reason)
- Purpose: Issues Mystic Door spawn commands (on cast) and remove commands (on buff cancel) to atlas-doors

### COMMAND_TOPIC_DROP
- Direction: Command
- Message Type: `Command[RequestReservationBody]`
- Purpose: Issues drop pickup reservation commands

### COMMAND_TOPIC_EXPRESSION
- Direction: Command
- Message Type: `Command`
- Envelope Fields: characterId, worldId, channelId, mapId, instance, expression
- Purpose: Issues character expression commands

### COMMAND_TOPIC_FAME
- Direction: Command
- Message Type: `Command[RequestChangeFameBody]`
- Purpose: Issues fame change commands. REQUEST_CHANGE carries CharacterId, WorldId at envelope level with ChannelId, MapId, Instance, TargetId, Amount in body.

### COMMAND_TOPIC_GUILD
- Direction: Command
- Message Type: Guild commands
- Type Discriminators: REQUEST_CREATE, REQUEST_INVITE, CREATION_AGREEMENT, CHANGE_EMBLEM, CHANGE_NOTICE, CHANGE_TITLES, CHANGE_MEMBER_TITLE, LEAVE
- Purpose: Issues guild operation commands

### COMMAND_TOPIC_GUILD_THREAD
- Direction: Command
- Message Type: `Command[CreateBody]`, `Command[UpdateBody]`, `Command[DeleteBody]`, `Command[AddReplyBody]`, `Command[DeleteReplyBody]`
- Purpose: Issues guild BBS thread commands

### COMMAND_TOPIC_INVITE
- Direction: Command
- Message Type: `Command[AcceptBody]`, `Command[RejectBody]`
- Purpose: Issues invite accept/reject commands. ACCEPT carries ReferenceId and TargetId. REJECT carries OriginatorId and TargetId. Commands are world-scoped (WorldId and InviteType at envelope level).

### COMMAND_TOPIC_MERCHANT
- Direction: Command
- Message Type: `Command[CommandPlaceShopBody]`, `Command[CommandOpenShopBody]`, `Command[CommandCloseShopBody]`, `Command[CommandEnterShopBody]`, `Command[CommandExitShopBody]`, `Command[CommandSendMessageBody]`, `Command[CommandEnterMaintenanceBody]`, `Command[CommandExitMaintenanceBody]`, `Command[CommandAddListingBody]`, `Command[CommandRemoveListingBody]`, `Command[CommandPurchaseBundleBody]`, `Command[CommandRecordItemSearchBody]`, `Command[CommandWithdrawMesoBody]`, `Command[CommandOrganizeListingsBody]`, `Command[CommandBlacklistBody]`
- Envelope: `Command[E]` with fields: WorldId (world.Id), ChannelId (channel.Id), CharacterId (uint32), Type (string), Body (E)
- Type Discriminators: `PLACE_SHOP`, `OPEN_SHOP`, `CLOSE_SHOP`, `ENTER_SHOP`, `EXIT_SHOP`, `SEND_MESSAGE`, `ENTER_MAINTENANCE`, `EXIT_MAINTENANCE`, `ADD_LISTING`, `REMOVE_LISTING`, `PURCHASE_BUNDLE`, `RECORD_ITEM_SEARCH`, `WITHDRAW_MESO`, `ORGANIZE_LISTINGS`, `ADD_BLACKLIST`, `REMOVE_BLACKLIST`
- Purpose: Issues personal shop / hired merchant commands. PLACE_SHOP registers a new shop (shopType, title, mapId, instanceId, x, y, permitItemId). OPEN_SHOP / CLOSE_SHOP toggle shop lifecycle. ENTER_SHOP (carries visitorName) / EXIT_SHOP track visitor presence. SEND_MESSAGE relays shop chat (content). ENTER_MAINTENANCE / EXIT_MAINTENANCE toggle owner management mode. ADD_LISTING adds an item to the shop (resolved itemId, itemType, assetId, and a full item snapshot, plus inventoryType, slot, bundleCount, bundleSize, pricePerBundle). REMOVE_LISTING removes a listing by index. PURCHASE_BUNDLE buys a listing (listingIndex, bundleCount). RECORD_ITEM_SEARCH increments the owl item-search count (itemId). WITHDRAW_MESO withdraws the shop's accrued meso. ORGANIZE_LISTINGS compacts the listing order. ADD_BLACKLIST / REMOVE_BLACKLIST maintain the shop blacklist (name; ADD may carry a bannedCharacterId to eject a current visitor).
- Note: The `merchant2.Command` type also defines `UPDATE_LISTING` (`CommandUpdateListingBody`); the channel service defines no producer for it.

### COMMAND_TOPIC_MESSENGER
- Direction: Command
- Message Type: `Command[CreateBody]`, `Command[LeaveBody]`, `Command[RequestInviteBody]`
- Purpose: Issues messenger operation commands

### COMMAND_TOPIC_MONSTER
- Direction: Command
- Message Type: `Command[DamageCommandBody]`, `Command[UseSkillCommandBody]`, `Command[ApplyStatusCommandBody]`, `Command[CancelStatusCommandBody]`
- Envelope: `Command[E]` with fields: WorldId (world.Id), ChannelId (channel.Id), MapId (_map.Id), Instance (uuid.UUID), MonsterId (uint32), Type (string), Body (E)
- Purpose: Issues monster commands. DAMAGE applies damage (CharacterId, Damage, AttackType). USE_SKILL triggers monster skill usage (CharacterId, SkillId, SkillLevel). APPLY_STATUS applies debuffs (SourceType, SourceCharacterId, SourceSkillId, SourceSkillLevel, Statuses map, Duration, TickInterval). CANCEL_STATUS removes status effects (StatusTypes list).

### COMMAND_TOPIC_MONSTER_BOOK
- Direction: Command
- Message Type: `Command[SetCoverBody]`
- Envelope: `Command[E]` with fields: TenantId (uuid.UUID), CharacterId (uint32), EventId (uuid.UUID), Type (string), Body (E)
- Type Discriminators: `SET_COVER`
- Purpose: Issues a monster-book cover-change command (CoverCardId) to atlas-monster-book. (The `CARD_PICKED_UP` command type is defined on the shared envelope but the channel service defines no producer for it.)

### COMMAND_TOPIC_MONSTER_MOVEMENT
- Direction: Command
- Message Type: `Command`
- Envelope Fields: worldId, channelId, mapId, instance, objectId, observerId, x, y, stance
- Purpose: Issues monster movement commands

### COMMAND_TOPIC_MTS
- Direction: Command
- Message Type: `Command[CreateListingCommandBody]`, `Command[BuyCommandBody]`, `Command[PlaceBidCommandBody]`, `Command[CancelListingCommandBody]`, `Command[TakeHomeCommandBody]`, `Command[RegisterWishCommandBody]`, `Command[RemoveWishCommandBody]`
- Type Discriminators: `CREATE_LISTING`, `BUY`, `PLACE_BID`, `CANCEL_LISTING`, `TAKE_HOME`, `REGISTER_WISH`, `REMOVE_WISH`
- Purpose: Issues MTS marketplace commands to atlas-mts, keyed on the acting character. CREATE_LISTING/BUY/PLACE_BID/CANCEL_LISTING/TAKE_HOME carry a TransactionId (saga-driven); REGISTER_WISH/REMOVE_WISH do not.

### COMMAND_TOPIC_NOTE
- Direction: Command
- Message Type: `Command[CreateBody]`, `Command[DiscardBody]`
- Purpose: Issues note send/delete commands

### COMMAND_TOPIC_NPC
- Direction: Command
- Message Type: `Command[StartConversationBody]`, `Command[ContinueConversationBody]`, `Command[EndConversationBody]`
- Purpose: Issues NPC interaction commands

### COMMAND_TOPIC_NPC_CONVERSATION
- Direction: Command
- Message Type: `Command[SimpleBody]`, `Command[TextBody]`, `Command[StyleBody]`, `Command[NumberBody]`, `Command[SlideMenuBody]`
- Envelope Fields: worldId, channelId, characterId, npcId, speaker, endChat, secondaryNpcId, message
- Purpose: Issues NPC conversation state commands

### COMMAND_TOPIC_NPC_SHOP
- Direction: Command
- Message Type: `Command[EnterBody]`, `Command[ExitBody]`, `Command[BuyBody]`, `Command[SellBody]`, `Command[RechargeBody]`
- Purpose: Issues NPC shop transaction commands

### COMMAND_TOPIC_PARTY
- Direction: Command
- Message Type: `Command[CreateBody]`, `Command[LeaveBody]`, `Command[ChangeLeaderBody]`, `Command[RequestInviteBody]`
- Purpose: Issues party operation commands

### COMMAND_TOPIC_PET
- Direction: Command
- Message Type: `Command[SpawnCommandBody]`, `Command[DespawnCommandBody]`, `Command[AttemptCommandCommandBody]`, `Command[SetExcludeCommandBody]`
- Purpose: Issues pet spawn, despawn, command attempt, and exclude list commands

### COMMAND_TOPIC_PET_MOVEMENT
- Direction: Command
- Message Type: `Command`
- Envelope Fields: worldId, channelId, mapId, instance, objectId, observerId, x, y, stance
- Purpose: Issues pet movement commands

### COMMAND_TOPIC_PORTAL
- Direction: Command
- Message Type: `Command[EnterBody]`, `WarpCommand`
- Envelope Fields: worldId, channelId, mapId, instance, portalId
- Purpose: Issues portal commands. ENTER triggers portal entry by portal ID. WARP triggers direct map warp by target map ID.

### COMMAND_TOPIC_QUEST
- Direction: Command
- Message Type: `QuestCommand[StartQuestCommandBody]`, `QuestCommand[CompleteQuestCommandBody]`, `QuestCommand[ForfeitQuestCommandBody]`, `QuestCommand[RestoreItemCommandBody]`
- Purpose: Issues quest start, complete, forfeit, and item restore commands

### COMMAND_TOPIC_QUEST_CONVERSATION
- Direction: Command
- Message Type: Quest conversation commands
- Purpose: Issues quest conversation commands

### COMMAND_TOPIC_REACTOR
- Direction: Command
- Message Type: `Command[HitCommandBody]`
- Purpose: Issues reactor hit commands

### COMMAND_TOPIC_SAGA
- Direction: Command
- Message Type: Saga commands
- Purpose: Issues saga orchestration commands

### COMMAND_TOPIC_SKILL
- Direction: Command
- Message Type: `Command[SetCooldownBody]`
- Purpose: Issues skill cooldown commands. SET_COOLDOWN sets cooldown timer for a skill (SkillId, Cooldown).

### COMMAND_TOPIC_SKILL_MACRO
- Direction: Command
- Message Type: `Command[UpdateBody]`
- Body: Contains array of MacroBody (id, name, shout, skillIds)
- Purpose: Issues skill macro update commands

### COMMAND_TOPIC_STORAGE
- Direction: Command
- Message Type: `Command[ArrangeCommandBody]`, `Command[UpdateMesosCommandBody]`, `CloseStorageCommand`
- Mesos Operations: SET, ADD, SUBTRACT
- Purpose: Issues storage commands. ARRANGE triggers item merge and sort within storage. UPDATE_MESOS deposits or withdraws mesos (ADD/SUBTRACT operations). CLOSE_STORAGE clears NPC context for a character.

### COMMAND_TOPIC_SUMMON
- Direction: Command
- Message Type: `Command[SpawnCommandBody]`, `Command[MoveCommandBody]`, `Command[AttackCommandBody]`, `Command[DamageCommandBody]`
- Envelope: `Command[E]` with fields: WorldId (world.Id), ChannelId (channel.Id), MapId (_map.Id), Instance (uuid.UUID), SummonId (uint32), Type (string), Body (E)
- Type Discriminators: `SPAWN`, `MOVE`, `ATTACK`, `DAMAGE`
- Purpose: Issues summon commands to atlas-summons. SPAWN creates an owner-bound summon (carries AURA_OF_THE_BEHOLDER/HEX_OF_THE_BEHOLDER levels for a Beholder summon). MOVE repositions a summon and rebroadcasts the raw movement blob. ATTACK credits the owner for a list of {monster, reported damage} targets. DAMAGE decrements a puppet summon's HP.

### COMMAND_TOPIC_TAMING_MOB_FOOD
- Direction: Command
- Message Type: `Command[RequestFeedBody]`
- Envelope: `Command[E]` with fields: TransactionId (uuid.UUID), WorldId (world.Id), ChannelId (channel.Id), MapId (_map.Id), Instance (uuid.UUID), CharacterId (character.Id), Type (string), Body (E)
- Type Discriminators: `REQUEST_FEED`
- Purpose: Issues a taming-mob (mount) food-consumption request (Slot, ItemId) to the consumables service. Performs no item mutation itself; consumables decrements the item.

---

## Message Types

### StatusEvent
Generic status event envelope with type discriminator and typed body. Used across asset, compartment, pet, storage, and other event topics.

### StorageCompartmentEvent
Storage-specific compartment event envelope with WorldId, AccountId, CharacterId, Type, and typed body.

### Command
Generic command envelope with type discriminator and typed body. Used for all outbound commands.

### ChatEvent
Chat event envelope with field context (WorldId, ChannelId, MapId, Instance), ActorId, Message, type discriminator, and typed body. Used for character chat events.

### RewardWonEvent
Flat event (not envelope-wrapped) for gachapon reward notifications, containing character, item, and gachapon details.

### QuestCommand
Quest-specific command envelope used for quest operations (start, complete, forfeit, restore item).

### ListingEvent
Merchant listing event envelope with ShopId, Type discriminator, and typed body. Used for the merchant listing topic.

### BroadcastEvent
Flat event (not envelope-wrapped) for the stateless megaphone tiers (MEGAPHONE/SUPER/ITEM/TRIPLE), containing tier, scope, worldId, channelId, characterId, senderName, senderMedal, messages, whispersOn, and an optional item snapshot. Used for `EVENT_TOPIC_MEGAPHONE`.

### StatusEvent (World Broadcast)
Flat event (not envelope-wrapped; distinct from the generic `StatusEvent[E]` envelope above), defined in the `worldbroadcast` message package. Contains type, family, worldId, characterId, waitSeconds, totalWaitSeconds, channelId, senderName, senderMedal, messages, whispersOn, itemId, tvMessageType (semantic key, not a wire byte), senderLook, receiverName, and receiverLook. Used for `EVENT_TOPIC_WORLD_BROADCAST_STATUS`.

---

## Transaction Semantics

- Consumer group ID follows pattern: `Channel Service - {SERVICE_ID}`
- Consumers start from `LastOffset` for real-time event processing
- Tenant ID passed in Kafka headers for multi-tenant filtering
- Span context passed in headers for distributed tracing
- Messages keyed by character ID (or account ID for storage commands, or map ID for door commands) for ordering guarantees
- Producer uses `producer.Provider` type = `func(token string) producer.MessageProducer`
- All topics resolved via `topic.EnvProvider` from environment variables
- Brokers configured via `BOOTSTRAP_SERVERS` environment variable
