# Saga

## Responsibility

Coordinates distributed transactions across multiple Atlas microservices using the saga pattern. Maintains transaction consistency by tracking step execution and performing compensation on failure.

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

### Saga Types

| Type | Description |
|------|-------------|
| inventory_transaction | Inventory-related transactions |
| quest_reward | Quest reward distribution |
| trade_transaction | Player-to-player trading |
| character_creation | Character creation workflows |
| storage_operation | Account storage operations |
| character_respawn | Character respawn handling |

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
| award_inventory | (Deprecated) Awards items to inventory |
| award_experience | Awards experience points |
| award_level | Awards levels |
| award_mesos | Awards mesos currency |
| award_currency | Awards cash shop currency |
| award_fame | Awards fame |
| warp_to_random_portal | Warps to a random portal in a field |
| warp_to_portal | Warps to a specific portal |
| destroy_asset | Destroys an inventory asset |
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
| apply_consumable_effect | Applies consumable item effects |
| send_message | Sends a system message |
| deposit_to_storage | Deposits to account storage |
| update_storage_mesos | Updates storage mesos |
| show_storage | Shows storage UI |
| transfer_to_storage | Transfers item to storage |
| withdraw_from_storage | Withdraws item from storage |
| accept_to_storage | Accepts item to storage (internal) |
| release_from_character | Releases item from character (internal) |
| accept_to_character | Accepts item to character (internal) |
| release_from_storage | Releases item from storage (internal) |
| transfer_to_cash_shop | Transfers item to cash shop |
| withdraw_from_cash_shop | Withdraws item from cash shop |
| accept_to_cash_shop | Accepts item to cash shop (internal) |
| release_from_cash_shop | Releases item from cash shop (internal) |
| set_hp | Sets character HP |
| deduct_experience | Deducts character experience |
| cancel_all_buffs | Cancels all character buffs |
| play_portal_sound | Plays portal sound effect |
| show_info | Shows info effect |
| show_info_text | Shows info text message |
| update_area_info | Updates area info |
| show_hint | Shows hint box |
| block_portal | Blocks a portal |
| unblock_portal | Unblocks a portal |

## Invariants

- Transaction ID must be non-nil
- Saga type must be non-empty
- Step IDs must be unique within a saga
- Step ordering must follow: completed steps before pending steps
- A failing saga has exactly one failed step
- Status transitions: pending -> completed, pending -> failed, completed -> failed, failed -> pending

## State Transitions

### Saga Lifecycle

1. Saga created with all steps in pending status
2. Steps execute sequentially from first pending step
3. On step completion: step marked completed, next step executes
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

## Processors

### Processor

Manages saga execution lifecycle.

| Method | Description |
|--------|-------------|
| GetAll | Returns all sagas for the tenant |
| GetById | Returns a saga by transaction ID |
| Put | Adds or updates a saga |
| MarkFurthestCompletedStepFailed | Marks the last completed step as failed |
| MarkEarliestPendingStep | Updates the first pending step status |
| MarkEarliestPendingStepCompleted | Marks the first pending step as completed |
| StepCompleted | Handles step completion event |
| AddStep | Adds a step after the current step |
| AddStepAfterCurrent | Adds a step after the current pending step |
| Step | Executes the next pending step |

### Handler

Executes action-specific logic for each step type.

| Method | Description |
|--------|-------------|
| GetHandler | Returns the handler function for an action type |

### Compensator

Performs compensation actions for failed steps.

| Method | Description |
|--------|-------------|
| CompensateFailedStep | Executes compensation for a failed step |

### Cache

In-memory storage for active sagas, tenant-scoped.

| Method | Description |
|--------|-------------|
| GetAll | Returns all sagas for a tenant |
| GetById | Returns a saga by ID for a tenant |
| Put | Stores a saga for a tenant |
| Remove | Removes a saga from a tenant |
