# Saga

## Responsibility

Coordinates distributed transactions across multiple Atlas microservices using the saga pattern. Maintains transaction consistency by tracking step execution and performing compensation on failure. High-level transfer actions are expanded at runtime into concrete step pairs by fetching asset data from inventory services.

## Core Models

### Saga

Represents a distributed transaction containing ordered steps.

| Field | Type | Description |
|-------|------|-------------|
| transactionId | uuid.UUID | Unique identifier for the transaction |
| sagaType | Type | Category of the saga |
| initiatedBy | string | Originator of the saga |
| steps | []Step[any] | Ordered list of steps to execute |

### Step

Represents a single action within a saga.

| Field | Type | Description |
|-------|------|-------------|
| stepId | string | Unique identifier for the step |
| status | Status | Current execution status |
| action | Action | Action type to execute |
| payload | any | Action-specific data |
| createdAt | time.Time | Step creation timestamp |
| updatedAt | time.Time | Last modification timestamp |
| result | map[string]any | Completion data carried between steps (nil if unset) |

### Saga Types

| Type | Description |
|------|-------------|
| inventory_transaction | Inventory-related transactions |
| quest_reward | Quest reward distribution |
| trade_transaction | Player-to-player trading |
| character_creation | Character creation workflows |
| storage_operation | Account storage operations |
| character_respawn | Character respawn handling |
| gachapon_transaction | Gachapon machine reward transactions |

### Step Status

| Status | Description |
|--------|-------------|
| pending | Step awaiting execution |
| completed | Step executed successfully |
| failed | Step execution failed |

### Actions

| Action | Description |
|--------|-------------|
| award_asset | Awards items to a character inventory |
| award_experience | Awards experience points |
| award_level | Awards levels |
| award_mesos | Awards mesos currency |
| award_currency | Awards cash shop currency |
| award_fame | Awards fame |
| warp_to_random_portal | Warps to a random portal in a field |
| warp_to_portal | Warps to a specific portal |
| destroy_asset | Destroys an inventory asset by template ID |
| destroy_asset_from_slot | Destroys an inventory asset from a specific slot |
| equip_asset | Equips an item |
| unequip_asset | Unequips an item |
| change_job | Changes character job |
| change_hair | Changes character hair style |
| change_face | Changes character face style |
| change_skin | Changes character skin color |
| create_skill | Creates a character skill |
| update_skill | Updates a character skill |
| validate_character_state | Validates character conditions |
| request_guild_name | Requests guild name change |
| request_guild_emblem | Requests guild emblem change |
| request_guild_disband | Requests guild disband |
| request_guild_capacity_increase | Requests guild capacity increase |
| create_invite | Creates an invitation |
| create_character | Creates a new character |
| create_and_equip_asset | Creates and equips an asset |
| increase_buddy_capacity | Increases buddy list capacity |
| gain_closeness | Increases pet closeness |
| spawn_monster | Spawns monsters |
| spawn_reactor_drops | Spawns reactor drops |
| complete_quest | Completes a quest |
| start_quest | Starts a quest |
| set_quest_progress | Sets quest progress info |
| apply_consumable_effect | Applies consumable item effects |
| cancel_consumable_effect | Cancels consumable item effects |
| send_message | Sends a system message (synchronous) |
| deposit_to_storage | Deposits to account storage |
| update_storage_mesos | Updates storage mesos |
| show_storage | Shows storage UI |
| transfer_to_storage | Transfers item to storage (expanded to accept_to_storage + release_from_character) |
| withdraw_from_storage | Withdraws item from storage (expanded to accept_to_character + release_from_storage) |
| accept_to_storage | Accepts item to storage (internal, created by expansion) |
| release_from_character | Releases item from character (internal, created by expansion) |
| accept_to_character | Accepts item to character (internal, created by expansion) |
| release_from_storage | Releases item from storage (internal, created by expansion) |
| transfer_to_cash_shop | Transfers item to cash shop (expanded to accept_to_cash_shop + release_from_character) |
| withdraw_from_cash_shop | Withdraws item from cash shop (expanded to accept_to_character + release_from_cash_shop) |
| accept_to_cash_shop | Accepts item to cash shop (internal, created by expansion) |
| release_from_cash_shop | Releases item from cash shop (internal, created by expansion) |
| set_hp | Sets character HP to an absolute value |
| deduct_experience | Deducts character experience (floor at 0) |
| cancel_all_buffs | Cancels all active buffs on character |
| reset_stats | Resets character stats (for job advancement) |
| play_portal_sound | Plays portal sound effect (synchronous) |
| show_info | Shows info/tutorial effect (synchronous) |
| show_info_text | Shows info text message (synchronous) |
| update_area_info | Updates area info (synchronous) |
| show_hint | Shows hint box (synchronous) |
| show_guide_hint | Shows pre-defined guide hint by ID (synchronous) |
| show_intro | Shows intro/direction effect (synchronous) |
| field_effect | Shows field effect (synchronous) |
| ui_lock | Locks/unlocks UI and disables/enables UI input (synchronous) |
| block_portal | Blocks a portal for a character (synchronous) |
| unblock_portal | Unblocks a portal for a character (synchronous) |
| start_instance_transport | Starts an instance-based transport (synchronous, REST call) |
| save_location | Saves character's current location for later return (synchronous, REST call) |
| warp_to_saved_location | Warps character to a saved location and deletes it (synchronous, REST call) |
| select_gachapon_reward | Selects a random reward from a gachapon machine (synchronous, REST call) |
| emit_gachapon_win | Emits gachapon win event for announcements (synchronous) |
| register_party_quest | Validates party requirements and registers a party quest (synchronous) |
| leave_party_quest | Leaves a party quest (synchronous) |
| warp_party_quest_members_to_map | Resolves party members and warps all to a map (synchronous) |
| update_pq_custom_data | Updates party quest instance custom data (synchronous) |
| hit_reactor | Resolves reactor by name and produces hit command (synchronous) |
| broadcast_pq_message | Broadcasts message to party quest members (synchronous) |
| stage_clear_attempt_pq | Attempts to clear the current PQ stage (synchronous) |
| enter_party_quest_bonus | Enters the bonus stage of a party quest (synchronous, terminal failure) |
| field_effect_weather | Shows weather effect to all characters in a field (synchronous) |

### AssetData

A flat structure representing all asset properties, defined in `kafka/message/asset/kafka.go`. Used for carrying asset data in transfer operations (accept/release steps). Contains all fields for any item type (equipable, consumable, cash, pet) in a single struct.

| Field | Type | Description |
|-------|------|-------------|
| expiration | time.Time | Expiration time |
| createdAt | time.Time | Creation timestamp |
| quantity | uint32 | Item quantity |
| ownerId | uint32 | Owner identifier |
| flag | uint16 | Item flags |
| rechargeable | uint64 | Recharge amount |
| strength | uint16 | Strength stat bonus |
| dexterity | uint16 | Dexterity stat bonus |
| intelligence | uint16 | Intelligence stat bonus |
| luck | uint16 | Luck stat bonus |
| hp | uint16 | HP bonus |
| mp | uint16 | MP bonus |
| weaponAttack | uint16 | Weapon attack bonus |
| magicAttack | uint16 | Magic attack bonus |
| weaponDefense | uint16 | Weapon defense bonus |
| magicDefense | uint16 | Magic defense bonus |
| accuracy | uint16 | Accuracy bonus |
| avoidability | uint16 | Avoidability bonus |
| hands | uint16 | Hands bonus |
| speed | uint16 | Speed bonus |
| jump | uint16 | Jump bonus |
| slots | uint16 | Available upgrade slots |
| locked | bool | Whether the item is locked |
| spikes | bool | Whether spikes are applied |
| karmaUsed | bool | Whether karma scissors were used |
| cold | bool | Cold protection flag |
| canBeTraded | bool | Whether the item can be traded |
| levelType | byte | Level type for leveling items |
| level | byte | Item level |
| experience | uint32 | Item experience |
| hammersApplied | uint32 | Number of hammers applied |
| equippedSince | *time.Time | When the item was equipped |
| cashId | int64 | Cash serial number |
| commodityId | uint32 | Commodity identifier |
| purchaseBy | uint32 | Purchaser character ID |
| petId | uint32 | Pet identifier |

## Invariants

- Transaction ID must be non-nil
- Saga type must be non-empty
- Step IDs must be unique within a saga
- Step ordering must follow: completed steps before pending steps
- A failing saga has exactly one failed step
- Status transitions: pending -> completed, pending -> failed, completed -> failed, failed -> pending
- State consistency is validated before every cache write

## State Transitions

### Saga Lifecycle

1. Saga created with all steps in pending status
2. Steps execute sequentially from first pending step
3. On step completion: step marked completed (optionally with result data), next step executes
4. On step failure: step marked failed, compensation begins
5. Compensation reverses completed steps in reverse order
6. Saga removed from cache on completion or full compensation

### Step State Machine

```
pending -> completed (success)
pending -> failed (failure)
completed -> failed (compensation trigger)
failed -> pending (compensation applied)
```

### Step Expansion

High-level transfer actions are expanded at runtime before execution:
- `transfer_to_storage` expands to `accept_to_storage` + `release_from_character`
- `withdraw_from_storage` expands to `accept_to_character` + `release_from_storage`
- `transfer_to_cash_shop` expands to `accept_to_cash_shop` + `release_from_character`
- `withdraw_from_cash_shop` expands to `accept_to_character` + `release_from_cash_shop`

During expansion, the orchestrator fetches full asset data from the source inventory service and pre-populates the concrete step payloads with the flat AssetData structure.

## Processors

### Processor

Manages saga execution lifecycle.

| Method | Description |
|--------|-------------|
| GetAll | Returns all sagas for the tenant |
| GetById | Returns a saga by transaction ID |
| Put | Adds or updates a saga, then steps |
| MarkFurthestCompletedStepFailed | Marks the last completed step as failed |
| MarkEarliestPendingStep | Updates the first pending step status |
| MarkEarliestPendingStepCompleted | Marks the first pending step as completed |
| StepCompleted | Handles step completion event |
| StepCompletedWithResult | Handles step completion with result data carried forward |
| AddStep | Adds a step after the current step |
| AddStepAfterCurrent | Adds a step after the first pending step |
| Step | Executes the next pending step or triggers compensation |

### Handler

Executes action-specific logic for each step type. Delegates to domain-specific processors (compartment, character, skill, guild, storage, cashshop, party quest, reactor, etc.).

| Method | Description |
|--------|-------------|
| GetHandler | Returns the handler function for an action type |

### Compensator

Performs compensation actions for failed steps.

| Method | Description |
|--------|-------------|
| CompensateFailedStep | Executes compensation for a failed step |

Compensation strategies by action type:
- **ValidateCharacterState**: Terminal failure, no compensation; emits FAILED event
- **RegisterPartyQuest**: Terminal failure; removes saga from cache and emits FAILED event with party quest error code
- **WarpPartyQuestMembersToMap**: Terminal failure on member resolution error; removes saga from cache and emits FAILED event with party quest error code
- **EnterPartyQuestBonus**: Terminal failure; removes saga from cache and emits FAILED event with party quest error code
- **EquipAsset**: Reverses with UnequipAsset command
- **UnequipAsset**: Reverses with EquipAsset command
- **CreateCharacter**: No rollback available; acknowledges failure
- **CreateAndEquipAsset**: Destroys created asset if auto-equip step exists
- **ChangeHair/ChangeFace/ChangeSkin**: No rollback available; cosmetic already applied
- **AwardMesos, AcceptToStorage, AcceptToCharacter, ReleaseFromStorage, ReleaseFromCharacter**: Terminal storage failures; emits error event with context-appropriate error code
- **SelectGachaponReward**: Re-awards destroyed ticket items, then emits failure event
- **Default**: Marks failed step as pending (removes failed status)

### Cache

In-memory storage for active sagas, tenant-scoped.

| Method | Description |
|--------|-------------|
| GetAll | Returns all sagas for a tenant |
| GetById | Returns a saga by ID for a tenant |
| Put | Stores a saga for a tenant |
| Remove | Removes a saga from a tenant |

### ErrorMapper

Determines context-appropriate error codes for saga failures.

| Error Code | Condition |
|------------|-----------|
| NOT_ENOUGH_MESOS | AwardMesos failure in storage operation |
| INVENTORY_FULL | AcceptToCharacter failure in storage operation |
| STORAGE_FULL | AcceptToStorage failure in storage operation |
| UNKNOWN | All other failures |

### CharacterExtractor

Extracts the character ID from any step payload type. Returns 0 for unknown payload types.

---

# Reactor Drop

## Responsibility

Spawns item and meso drops from reactor activation with quest-aware filtering and rate multipliers.

## Core Models

### Model

| Field | Type | Description |
|-------|------|-------------|
| reactorId | uint32 | Reactor identifier |
| itemId | uint32 | Item identifier |
| questId | uint32 | Associated quest identifier (0 for non-quest items) |
| chance | uint32 | Drop chance (higher = rarer) |

### DropType Constants

| Value | Name | Description |
|-------|------|-------------|
| 1 | DropTypeSpray | Spray drop animation |
| 2 | DropTypeImmediate | Immediate drop |

## Processors

### SpawnReactorDrops

Spawns drops from reactor activation.

- Fetches character rate multipliers
- Retrieves reactor drop configuration from drop information service
- Filters quest-specific drops based on character's started quests
- Rolls chances to determine which items drop
- Calculates meso padding if minimum items not met
- Calculates drop positions using foothold data
- Produces spawn drop commands via Kafka

### filterByQuestState

Filters drops based on character's quest state.

- Returns all drops unchanged if no quest-specific drops exist
- Fetches started quest IDs for the character from quest service
- Includes drops with questId == 0 (non-quest items)
- Includes drops with questId matching a started quest
- Excludes drops with questId not matching any started quest
- On quest service error, excludes all quest-specific drops

---

# Reactor (Client)

## Responsibility

Resolves reactors by name via REST call to atlas-reactors and produces Kafka HIT commands to trigger reactor actions.

## Core Models

### ReactorRestModel

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Reactor identifier |
| name | string | Reactor name for lookup |

## Processors

| Method | Description |
|--------|-------------|
| HitReactorByName | Looks up reactors by name in a field via REST, then produces a HIT command for each matching reactor |

---

# Validation

## Responsibility

Validates character state conditions before saga step execution. Supports condition checks for job, meso, map, fame, and item possession.

## Core Models

### ConditionType

| Value | Description |
|-------|-------------|
| jobId | Job condition check |
| meso | Meso amount condition check |
| mapId | Map location condition check |
| fame | Fame condition check |
| item | Item possession condition check |

### Operator

| Value | Description |
|-------|-------------|
| = | Equals |
| > | Greater than |
| < | Less than |
| >= | Greater than or equal |
| <= | Less than or equal |

### ConditionInput

| Field | Type | Description |
|-------|------|-------------|
| type | string | Condition type (jobId, meso, item, etc.) |
| operator | string | Comparison operator |
| value | int | Value or quantity to check |
| itemId | uint32 | Item ID (only for item conditions) |

### ValidationResult

| Field | Type | Description |
|-------|------|-------------|
| passed | bool | Whether all conditions passed |
| details | []string | Human-readable descriptions |
| results | []ConditionResult | Structured condition results |
| characterId | uint32 | Character that was validated |

---

# Quest State

## Responsibility

Represents quest state information retrieved from external service for quest-aware drop filtering.

## Core Models

### State

Quest state enumeration.

| Value | Name | Description |
|-------|------|-------------|
| 0 | StateNotStarted | Quest not started |
| 1 | StateStarted | Quest in progress |
| 2 | StateCompleted | Quest completed |

### Model

| Field | Type | Description |
|-------|------|-------------|
| characterId | uint32 | Character identifier |
| questId | uint32 | Quest identifier |
| state | State | Quest state |

## Processors

### GetStartedQuestIds

Retrieves a set of started quest IDs for a character.

---

# Rates

## Responsibility

Represents character rate multipliers retrieved from external service.

## Core Models

### Model

| Field | Type | Description |
|-------|------|-------------|
| expRate | float64 | Experience rate multiplier |
| mesoRate | float64 | Meso rate multiplier |
| itemDropRate | float64 | Item drop rate multiplier |
| questExpRate | float64 | Quest experience rate multiplier |

## Processors

### GetForCharacter

Retrieves computed rates for a character. Returns default rates (all 1.0) if the rate service is unavailable.

---

# Gachapon

## Responsibility

Interfaces with the gachapon service to select random rewards and retrieve gachapon metadata for reward tier evaluation.

## Core Models

### RewardRestModel

| Field | Type | Description |
|-------|------|-------------|
| itemId | uint32 | Reward item ID |
| quantity | uint32 | Reward quantity |
| tier | string | Reward tier (common, uncommon, rare) |
| gachaponId | string | Source gachapon machine ID |

### GachaponRestModel

| Field | Type | Description |
|-------|------|-------------|
| name | string | Gachapon display name |
| npcIds | []uint32 | NPC IDs associated with this gachapon |
| commonWeight | uint32 | Common tier weight |
| uncommonWeight | uint32 | Uncommon tier weight |
| rareWeight | uint32 | Rare tier weight |

## Processors

### SelectReward

Selects a random reward from a gachapon machine via REST call.

### GetGachapon

Retrieves gachapon metadata (name, weights) via REST call.

---

# Compartment (Client)

## Responsibility

Produces Kafka commands to the inventory compartment service for asset operations within sagas.

## Core Models

### AssetRestModel

A flat model representing an asset from the character inventory service. Contains all properties for any item type (equipable stats, stackable quantities, cash item data, pet data) in a single structure.

| Field | Type | Description |
|-------|------|-------------|
| id | string | Asset identifier |
| slot | int16 | Inventory slot position |
| templateId | uint32 | Item template ID |
| expiration | time.Time | Expiration time |
| createdAt | time.Time | Creation timestamp |
| quantity | uint32 | Item quantity |
| ownerId | uint32 | Owner identifier |
| flag | uint16 | Item flags |
| rechargeable | uint64 | Recharge amount |
| strength | uint16 | Strength stat bonus |
| dexterity | uint16 | Dexterity stat bonus |
| intelligence | uint16 | Intelligence stat bonus |
| luck | uint16 | Luck stat bonus |
| hp | uint16 | HP bonus |
| mp | uint16 | MP bonus |
| weaponAttack | uint16 | Weapon attack bonus |
| magicAttack | uint16 | Magic attack bonus |
| weaponDefense | uint16 | Weapon defense bonus |
| magicDefense | uint16 | Magic defense bonus |
| accuracy | uint16 | Accuracy bonus |
| avoidability | uint16 | Avoidability bonus |
| hands | uint16 | Hands bonus |
| speed | uint16 | Speed bonus |
| jump | uint16 | Jump bonus |
| slots | uint16 | Available upgrade slots |
| locked | bool | Whether the item is locked |
| spikes | bool | Whether spikes are applied |
| karmaUsed | bool | Whether karma scissors were used |
| cold | bool | Cold protection flag |
| canBeTraded | bool | Whether the item can be traded |
| levelType | byte | Level type for leveling items |
| level | byte | Item level |
| experience | uint32 | Item experience |
| hammersApplied | uint32 | Number of hammers applied |
| equippedSince | *time.Time | When the item was equipped |
| cashId | int64 | Cash serial number |
| commodityId | uint32 | Commodity identifier |
| purchaseBy | uint32 | Purchaser character ID |
| petId | uint32 | Pet identifier |

### CompartmentRestModel

| Field | Type | Description |
|-------|------|-------------|
| id | string | Compartment identifier |
| inventoryType | byte | Inventory type |
| capacity | uint32 | Compartment capacity |
| assets | []AssetRestModel | Assets in the compartment |

## Processors

| Method | Description |
|--------|-------------|
| RequestCreateItem | Produces CREATE_ASSET command |
| RequestDestroyItem | Looks up slot by template ID, produces DESTROY command |
| RequestDestroyItemFromSlot | Produces DESTROY command for a specific slot |
| RequestEquipAsset | Produces EQUIP command |
| RequestUnequipAsset | Produces UNEQUIP command |
| RequestCreateAndEquipAsset | Delegates to RequestCreateItem (equip handled by asset consumer) |
| RequestAcceptAsset | Produces ACCEPT command with flat AssetData |
| RequestReleaseAsset | Produces RELEASE command |

---

# Storage (Client)

## Responsibility

Produces Kafka commands to the storage service for deposit, withdraw, accept, release, and mesos operations within sagas.

## Core Models

### AssetRestModel

A flat model representing an asset from the storage service.

| Field | Type | Description |
|-------|------|-------------|
| id | string | Asset identifier |
| slot | int16 | Storage slot position |
| templateId | uint32 | Item template ID |
| expiration | time.Time | Expiration time |
| quantity | uint32 | Item quantity |
| ownerId | uint32 | Owner identifier |
| flag | uint16 | Item flags |
| rechargeable | uint64 | Recharge amount |
| strength | uint16 | Strength stat bonus |
| dexterity | uint16 | Dexterity stat bonus |
| intelligence | uint16 | Intelligence stat bonus |
| luck | uint16 | Luck stat bonus |
| hp | uint16 | HP bonus |
| mp | uint16 | MP bonus |
| weaponAttack | uint16 | Weapon attack bonus |
| magicAttack | uint16 | Magic attack bonus |
| weaponDefense | uint16 | Weapon defense bonus |
| magicDefense | uint16 | Magic defense bonus |
| accuracy | uint16 | Accuracy bonus |
| avoidability | uint16 | Avoidability bonus |
| hands | uint16 | Hands bonus |
| speed | uint16 | Speed bonus |
| jump | uint16 | Jump bonus |
| slots | uint16 | Available upgrade slots |
| locked | bool | Whether the item is locked |
| spikes | bool | Whether spikes are applied |
| karmaUsed | bool | Whether karma scissors were used |
| cold | bool | Cold protection flag |
| canBeTraded | bool | Whether the item can be traded |
| levelType | byte | Level type for leveling items |
| level | byte | Item level |
| experience | uint32 | Item experience |
| hammersApplied | uint32 | Number of hammers applied |
| equippedSince | *time.Time | When the item was equipped |
| cashId | int64 | Cash serial number |
| commodityId | uint32 | Commodity identifier |
| purchaseBy | uint32 | Purchaser character ID |
| petId | uint32 | Pet identifier |

### ProjectionAssetRestModel

A flat model representing an asset from a storage projection, used during withdraw expansion.

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Asset identifier |
| slot | int16 | Storage slot position |
| templateId | uint32 | Item template ID |
| expiration | time.Time | Expiration time |
| quantity | uint32 | Item quantity |
| ownerId | uint32 | Owner identifier |
| flag | uint16 | Item flags |
| rechargeable | uint64 | Recharge amount |
| (same equipable/cash/pet fields as AssetRestModel) | | |

## Processors

| Method | Description |
|--------|-------------|
| DepositAndEmit | Produces DEPOSIT command |
| WithdrawAndEmit | Produces WITHDRAW command |
| UpdateMesosAndEmit | Produces UPDATE_MESOS command |
| DepositRollbackAndEmit | Produces DEPOSIT_ROLLBACK command |
| ShowStorageAndEmit | Produces SHOW_STORAGE command |
| AcceptAndEmit | Produces storage compartment ACCEPT command with flat AssetData |
| ReleaseAndEmit | Produces storage compartment RELEASE command |

---

# Cash Shop (Client)

## Responsibility

Produces Kafka commands to the cash shop service for currency, accept, and release operations within sagas.

## Core Models

### AssetRestModel

A flat model representing a cash shop inventory asset.

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Asset identifier |
| compartmentId | string | Compartment identifier |
| cashId | int64 | Cash serial number |
| templateId | uint32 | Item template ID |
| commodityId | uint32 | Commodity identifier |
| quantity | uint32 | Item quantity |
| flag | uint16 | Item flags |
| purchasedBy | uint32 | Purchaser character ID |
| expiration | time.Time | Expiration time |
| createdAt | time.Time | Creation timestamp |

### CompartmentRestModel

| Field | Type | Description |
|-------|------|-------------|
| id | uuid.UUID | Compartment identifier |
| accountId | uint32 | Account identifier |
| type | byte | Compartment type |
| capacity | uint32 | Compartment capacity |
| assets | []AssetRestModel | Assets in the compartment |

## Processors

| Method | Description |
|--------|-------------|
| AwardCurrencyAndEmit | Produces wallet ADJUST_CURRENCY command |
| AcceptAndEmit | Produces cash shop compartment ACCEPT command |
| ReleaseAndEmit | Produces cash shop compartment RELEASE command |

---

# Party Quest (Client)

## Responsibility

Validates party quest registration requirements and produces Kafka commands to the atlas-party-quests service for registration, leaving, custom data updates, and message broadcasting.

## Core Models

### PartyRestModel

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Party identifier |
| leaderId | uint32 | Party leader character ID |

### MemberRestModel

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Member character ID |
| name | string | Character name |
| level | byte | Character level |
| jobId | uint16 | Character job ID |
| worldId | world.Id | World identifier |
| channelId | channel.Id | Channel identifier |
| mapId | uint32 | Current map ID |
| online | bool | Online status |

### DefinitionRestModel

| Field | Type | Description |
|-------|------|-------------|
| id | string | Party quest definition ID |
| startRequirements | []ConditionRestModel | Start requirement conditions |

### ConditionRestModel

| Field | Type | Description |
|-------|------|-------------|
| type | string | Condition type (party_size, level_min, level_max) |
| operator | string | Comparison operator |
| value | uint32 | Condition value |
| referenceId | uint32 | Reference identifier |

### InstanceRestModel

| Field | Type | Description |
|-------|------|-------------|
| id | uuid.UUID | Party quest instance identifier |

### PartyQuestError

Custom error type with an error code field.

| Error Code | Description |
|------------|-------------|
| PQ_NOT_IN_PARTY | Character is not in a party |
| PQ_NOT_LEADER | Character is not the party leader |
| PQ_PARTY_SIZE | Party does not meet size requirements |
| PQ_LEVEL_MIN | A member does not meet minimum level |
| PQ_LEVEL_MAX | A member exceeds maximum level |
| PQ_DEFINITION_NOT_FOUND | Party quest definition not found |
| PQ_BONUS_NOT_AVAILABLE | Party quest bonus stage not available |
| PQ_UNKNOWN | Unknown error |

## Processors

| Method | Description |
|--------|-------------|
| RegisterPartyQuest | Validates party state and requirements, then produces REGISTER command |
| GetPartyMembers | Resolves character's party and returns all members |
| LeavePartyQuest | Produces LEAVE command to remove character from active party quest |
| UpdateCustomData | Produces UPDATE_CUSTOM_DATA command with updates and increment counters |
| BroadcastMessage | Produces BROADCAST_MESSAGE command to all party quest participants |
| StageClearAttempt | Produces STAGE_CLEAR_ATTEMPT command by instance ID |
| StageClearAttemptByCharacter | Looks up PQ instance by character, then produces STAGE_CLEAR_ATTEMPT command |
| EnterBonusByCharacter | Looks up PQ instance by character, then produces ENTER_BONUS command |

---

# System Message (Client)

## Responsibility

Produces Kafka commands to the system message service for UI effects, hints, and messages within sagas.

## Processors

| Method | Description |
|--------|-------------|
| SendMessage | Produces SEND_MESSAGE command |
| PlayPortalSound | Produces PLAY_PORTAL_SOUND command |
| ShowInfo | Produces SHOW_INFO command |
| ShowInfoText | Produces SHOW_INFO_TEXT command |
| UpdateAreaInfo | Produces UPDATE_AREA_INFO command |
| ShowHint | Produces SHOW_HINT command |
| ShowGuideHint | Produces SHOW_GUIDE_HINT command |
| ShowIntro | Produces SHOW_INTRO command |
| FieldEffect | Produces FIELD_EFFECT command |
| UiLock | Produces UI_LOCK command |
| UiDisable | Produces UI_DISABLE command |

---

# Monster (Client)

## Responsibility

Spawns monsters via REST call to the monster service.

## Core Models

### SpawnInputRestModel

| Field | Type | Description |
|-------|------|-------------|
| monsterId | uint32 | Monster template ID |
| x | int16 | X coordinate |
| y | int16 | Y coordinate |
| fh | int16 | Foothold ID |
| team | int8 | Team assignment |

### SpawnResponseRestModel

| Field | Type | Description |
|-------|------|-------------|
| uniqueId | uint32 | Spawned monster unique ID |
| worldId | world.Id | World identifier |
| channelId | channel.Id | Channel identifier |
| mapId | _map.Id | Map identifier |
| monsterId | uint32 | Monster template ID |
| controlCharacterId | uint32 | Controlling character ID |
| x | int16 | X coordinate |
| y | int16 | Y coordinate |
| fh | int16 | Foothold ID |
| stance | uint16 | Monster stance |
| team | int8 | Team assignment |
| maxHp | uint32 | Maximum HP |
| hp | uint32 | Current HP |
| maxMp | uint32 | Maximum MP |
| mp | uint32 | Current MP |

## Processors

| Method | Description |
|--------|-------------|
| SpawnMonster | Spawns a monster at the specified location via REST POST |

---

# Transport (Client)

## Responsibility

Starts instance-based transports via REST call to the transport service.

## Core Models

### TransportError

| Field | Type | Description |
|-------|------|-------------|
| code | string | Error code |
| message | string | Error message |

Error codes: ErrorCodeCapacityFull, ErrorCodeAlreadyInTransit, ErrorCodeRouteNotFound, ErrorCodeServiceError.

## Processors

| Method | Description |
|--------|-------------|
| StartTransport | Starts an instance transport for a character via REST POST |

---

# Saved Location (Client)

## Responsibility

Saves and retrieves character locations via REST calls to the character service.

## Core Models

### RestModel

| Field | Type | Description |
|-------|------|-------------|
| characterId | uint32 | Character identifier |
| locationType | string | Location type identifier |
| mapId | _map.Id | Map identifier |
| portalId | uint32 | Portal identifier |

## Processors

| Method | Description |
|--------|-------------|
| Put | Saves a location via REST PUT |
| Get | Retrieves a saved location via REST GET |
| Delete | Deletes a saved location via REST DELETE |

---

# Portal (Client)

## Responsibility

Produces Kafka commands to block and unblock portals for characters.

## Processors

| Method | Description |
|--------|-------------|
| BlockAndEmit | Produces BLOCK command and emits |
| Block | Adds BLOCK command to message buffer |
| UnblockAndEmit | Produces UNBLOCK command and emits |
| Unblock | Adds UNBLOCK command to message buffer |

---

# Buff (Client)

## Responsibility

Produces Kafka commands to cancel character buffs.

## Processors

| Method | Description |
|--------|-------------|
| CancelAllAndEmit | Produces CANCEL_ALL command and emits |
| CancelAll | Adds CANCEL_ALL command to message buffer |

---

# Map Command (Client)

## Responsibility

Produces Kafka commands to the atlas-maps service for field-level operations within sagas.

## Processors

| Method | Description |
|--------|-------------|
| FieldEffectWeather | Produces WEATHER_START command to COMMAND_TOPIC_MAP |
