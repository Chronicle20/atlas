package saga

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

// AwardItemActionPayload represents the data needed to award an item to a character.
type AwardItemActionPayload struct {
	CharacterId uint32      `json:"characterId"` // CharacterId associated with the action
	Item        ItemPayload `json:"item"`        // Item to award
	ShowEffect  bool        `json:"showEffect"`  // Render a client-visible item-gain effect/chat line when true
}

// ItemPayload represents an individual item in a transaction, such as in inventory manipulation.
type ItemPayload struct {
	TemplateId uint32    `json:"templateId"`           // TemplateId of the item
	Quantity   uint32    `json:"quantity"`             // Quantity of the item
	Period     uint32    `json:"period,omitempty"`     // Period in days for time-limited items (0 = permanent)
	Expiration time.Time `json:"expiration,omitempty"` // Expiration time for the item (zero value = no expiration)
}

// WarpToRandomPortalPayload represents the payload required to warp to a random portal within a specific field.
type WarpToRandomPortalPayload struct {
	CharacterId uint32   `json:"characterId"` // CharacterId associated with the action
	FieldId     field.Id `json:"fieldId"`     // FieldId references the unique identifier of the field
}

// WarpToPortalPayload represents the payload required to warp a character to a specific portal in a field.
type WarpToPortalPayload struct {
	CharacterId uint32     `json:"characterId"`          // CharacterId associated with the action
	WorldId     world.Id   `json:"worldId"`              // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`            // ChannelId associated with the action
	MapId       _map.Id    `json:"mapId"`                // MapId specifies the map to warp to
	Instance    uuid.UUID  `json:"instance"`             // Instance specifies the map instance UUID (uuid.Nil for non-instanced maps)
	PortalId    uint32     `json:"portalId"`             // PortalId specifies the unique identifier of the portal
	PortalName  string     `json:"portalName,omitempty"` // PortalName specifies the name of the portal (resolved to ID if provided)
}

// AwardExperiencePayload represents the payload required to award experience to a character.
type AwardExperiencePayload struct {
	CharacterId   uint32                    `json:"characterId"`   // CharacterId associated with the action
	WorldId       world.Id                  `json:"worldId"`       // WorldId associated with the action
	ChannelId     channel.Id                `json:"channelId"`     // ChannelId associated with the action
	Distributions []ExperienceDistributions `json:"distributions"` // List of experience distributions to award
	ShowEffect    bool                      `json:"showEffect"`    // Render a client-visible EXP chat line when true
}

// AwardLevelPayload represents the payload required to award levels to a character.
type AwardLevelPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId associated with the action
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	Amount      byte       `json:"amount"`      // Number of levels to award
}

// AwardMesosPayload represents the payload required to award mesos to a character.
type AwardMesosPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId associated with the action
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	ActorId     uint32     `json:"actorId"`     // ActorId identifies who is giving/taking the mesos
	ActorType   string     `json:"actorType"`   // ActorType identifies the type of actor (e.g., "SYSTEM", "NPC", "CHARACTER")
	Amount      int32      `json:"amount"`      // Amount of mesos to award (can be negative for deduction)
	ShowEffect  bool       `json:"showEffect"`  // Render the meso chat line on the client when true
}

// AwardCurrencyPayload represents the payload required to award cash shop currency to a character.
type AwardCurrencyPayload struct {
	CharacterId  uint32 `json:"characterId"`  // CharacterId associated with the action
	AccountId    uint32 `json:"accountId"`    // AccountId that owns the wallet
	CurrencyType uint32 `json:"currencyType"` // CurrencyType: 1=credit, 2=points, 3=prepaid
	Amount       int32  `json:"amount"`       // Amount of currency to award (can be negative for deduction)
}

// AwardFamePayload represents the payload required to award fame to a character.
type AwardFamePayload struct {
	CharacterId uint32     `json:"characterId"`         // CharacterId to award fame to
	WorldId     world.Id   `json:"worldId"`             // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`           // ChannelId associated with the action
	ActorId     uint32     `json:"actorId,omitempty"`   // ActorId identifies who is giving fame (e.g., quest ID)
	ActorType   string     `json:"actorType,omitempty"` // ActorType identifies the type of actor (e.g., "quest")
	Amount      int16      `json:"amount"`              // Amount of fame to award (can be negative)
}

// DestroyAssetPayload represents the payload required to destroy an asset in a compartment.
type DestroyAssetPayload struct {
	CharacterId uint32 `json:"characterId"` // CharacterId associated with the action
	TemplateId  uint32 `json:"templateId"`  // TemplateId of the item to destroy
	Quantity    uint32 `json:"quantity"`    // Quantity of the item to destroy (ignored if RemoveAll is true)
	RemoveAll   bool   `json:"removeAll"`   // If true, remove all instances of the item regardless of Quantity
	ShowEffect  bool   `json:"showEffect"`  // Render the item-loss chat line on the client when true
}

// DestroyAssetFromSlotPayload represents the payload required to destroy an asset from a specific inventory slot.
type DestroyAssetFromSlotPayload struct {
	CharacterId   uint32 `json:"characterId"`   // CharacterId associated with the action
	InventoryType byte   `json:"inventoryType"` // Type of inventory (1=equip, 2=use, 3=setup, 4=etc, 5=cash)
	Slot          int16  `json:"slot"`          // Slot to destroy from (negative for equipped slots, positive for inventory slots)
	Quantity      uint32 `json:"quantity"`      // Quantity to destroy (0 or 1 for equipment)
	ShowEffect    bool   `json:"showEffect"`    // Render the item-loss chat line on the client when true
	// TemplateId lets the compensator re-create a slot-destroyed asset
	TemplateId uint32 `json:"templateId,omitempty"`
}

// EquipAssetPayload represents the payload required to equip an asset from one inventory slot to an equipped slot.
type EquipAssetPayload struct {
	CharacterId   uint32 `json:"characterId"`   // CharacterId associated with the action
	InventoryType uint32 `json:"inventoryType"` // Type of inventory (e.g., equipment, consumables)
	Source        int16  `json:"source"`        // Source inventory slot
	Destination   int16  `json:"destination"`   // Destination equipped slot (negative values)
}

// UnequipAssetPayload represents the payload required to unequip an asset from an equipped slot.
type UnequipAssetPayload struct {
	CharacterId   uint32 `json:"characterId"`   // CharacterId associated with the action
	InventoryType uint32 `json:"inventoryType"` // Type of inventory
	Source        int16  `json:"source"`        // Source equipped slot (negative values)
	Destination   int16  `json:"destination"`   // Destination inventory slot
}

// CreateAndEquipAssetPayload represents the payload required to create and equip an asset.
type CreateAndEquipAssetPayload struct {
	CharacterId     uint32      `json:"characterId"`               // CharacterId associated with the action
	Item            ItemPayload `json:"item"`                      // Item to create and equip
	UseAverageStats bool        `json:"useAverageStats,omitempty"` // UseAverageStats indicates whether average stats should be used when creating the item
}

// ChangeJobPayload represents the payload required to change a character's job.
type ChangeJobPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId associated with the action
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	JobId       job.Id     `json:"jobId"`       // JobId to change to
}

// ChangeHairPayload represents the payload required to change a character's hair.
type ChangeHairPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId associated with the action
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	StyleId     uint32     `json:"styleId"`     // Hair style ID to change to
}

// ChangeFacePayload represents the payload required to change a character's face.
type ChangeFacePayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId associated with the action
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	StyleId     uint32     `json:"styleId"`     // Face style ID to change to
}

// ChangeSkinPayload represents the payload required to change a character's skin color.
type ChangeSkinPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId associated with the action
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	StyleId     byte       `json:"styleId"`     // Skin color ID to change to
}

// SetHPPayload represents the payload required to set a character's HP to an absolute value.
type SetHPPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId to set HP for
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	Amount      uint16     `json:"amount"`      // Absolute HP value to set
}

// DeductExperiencePayload represents the payload required to deduct experience from a character.
type DeductExperiencePayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId to deduct experience from
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	Amount      uint32     `json:"amount"`      // Amount of experience to deduct
}

// CancelAllBuffsPayload represents the payload required to cancel all active buffs on a character.
type CancelAllBuffsPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId to cancel buffs for
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	MapId       _map.Id    `json:"mapId"`       // MapId associated with the action
	Instance    uuid.UUID  `json:"instance"`    // Instance associated with the action
}

// ResetStatsPayload represents the payload required to reset a character's stats.
type ResetStatsPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId to reset stats for
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
}

// RebalanceStat identifies which primary stat a RebalanceAP target operates on.
type RebalanceStat string

const (
	RebalanceStatStrength     RebalanceStat = "strength"
	RebalanceStatDexterity    RebalanceStat = "dexterity"
	RebalanceStatIntelligence RebalanceStat = "intelligence"
	RebalanceStatLuck         RebalanceStat = "luck"
)

// RebalanceTarget pairs a primary stat with the floor value it should be raised to.
// Floor is uint16 to match the character entity stat columns (str/dex/int/luk), which are uint16.
type RebalanceTarget struct {
	Stat  RebalanceStat `json:"stat"`
	Floor uint16        `json:"floor"`
}

// RebalanceAPPayload represents the payload required to rebalance a character's
// primary stats during first-job advancement. The algorithm resets STR/DEX/INT/LUK
// to 4, raises each target stat to its floor, and returns the reclaimed surplus
// to unallocated AP. HP/MP are not touched.
type RebalanceAPPayload struct {
	CharacterId uint32            `json:"characterId"` // CharacterId associated with the action
	WorldId     world.Id          `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id        `json:"channelId"`   // ChannelId associated with the action
	Targets     []RebalanceTarget `json:"targets"`     // Target stats and floors to apply
}

// TransferAPPayload represents the payload for transfer_ap (AP Reset,
// item 5050000): move one already-spent ability point From -> To. From/To
// are validated ability enums (STRENGTH/DEXTERITY/INTELLIGENCE/LUCK/HP/MP),
// never raw client stat flags.
type TransferAPPayload struct {
	CharacterId uint32     `json:"characterId"`
	WorldId     world.Id   `json:"worldId"`
	ChannelId   channel.Id `json:"channelId"`
	From        string     `json:"from"`
	To          string     `json:"to"`
}

// TransferSPPayload represents the payload for transfer_sp (SP Reset,
// items 5050001-5050004): move one skill point FromSkillId -> ToSkillId.
// JobId, ItemTier, and TargetMaxLevel ride along for authoritative
// re-validation in atlas-skills (trusted server-side caller — atlas-channel).
type TransferSPPayload struct {
	CharacterId    uint32     `json:"characterId"`
	WorldId        world.Id   `json:"worldId"`
	ChannelId      channel.Id `json:"channelId"`
	JobId          job.Id     `json:"jobId"`
	FromSkillId    skill.Id   `json:"fromSkillId"`
	ToSkillId      skill.Id   `json:"toSkillId"`
	ItemTier       byte       `json:"itemTier"`
	TargetMaxLevel byte       `json:"targetMaxLevel"`
}

// CreateSkillPayload represents the payload required to create a skill for a character.
type CreateSkillPayload struct {
	CharacterId uint32    `json:"characterId"` // CharacterId associated with the action
	SkillId     uint32    `json:"skillId"`     // SkillId to create
	Level       byte      `json:"level"`       // Skill level
	MasterLevel byte      `json:"masterLevel"` // Skill master level
	Expiration  time.Time `json:"expiration"`  // Skill expiration time
}

// UpdateSkillPayload represents the payload required to update a skill for a character.
type UpdateSkillPayload struct {
	CharacterId uint32    `json:"characterId"` // CharacterId associated with the action
	SkillId     uint32    `json:"skillId"`     // SkillId to update
	Level       byte      `json:"level"`       // New skill level
	MasterLevel byte      `json:"masterLevel"` // New skill master level
	Expiration  time.Time `json:"expiration"`  // New skill expiration time
}

// IncreaseBuddyCapacityPayload represents the payload required to increase a character's buddy list capacity.
type IncreaseBuddyCapacityPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId associated with the action
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	Amount      byte       `json:"amount"`      // Amount to increase buddy capacity by
}

// GainClosenessPayload represents the payload required to gain closeness with a pet.
type GainClosenessPayload struct {
	PetId  uint32 `json:"petId"`  // PetId associated with the action
	Amount uint16 `json:"amount"` // Amount of closeness to gain
}

// EvolvePetPayload drives an NPC pet evolution. The outcome roll is owned by
// atlas-pets; this payload only identifies the pet.
type EvolvePetPayload struct {
	CharacterId uint32 `json:"characterId"`
	PetId       uint32 `json:"petId"`
}

// ValidateCharacterStatePayload represents the payload required to validate a character's state.
type ValidateCharacterStatePayload struct {
	CharacterId uint32                     `json:"characterId"` // CharacterId associated with the action
	Conditions  []ValidationConditionInput `json:"conditions"`  // Conditions to validate
}

// CompleteQuestPayload represents the payload required to complete a quest.
type CompleteQuestPayload struct {
	CharacterId uint32            `json:"characterId"`       // CharacterId associated with the action
	WorldId     world.Id          `json:"worldId"`           // WorldId associated with the action
	QuestId     uint32            `json:"questId"`           // QuestId to complete
	NpcId       uint32            `json:"npcId"`             // NpcId that completed the quest
	Force       bool              `json:"force"`             // If true, skip requirement checks and just mark complete
	Rewards     []QuestRewardItem `json:"rewards,omitempty"` // Item rewards granted alongside completion (for downstream display)
}

// QuestRewardItem describes an item granted as part of a quest completion,
// carried through the CompleteQuest saga step so downstream services can
// surface the rewards without reloading quest data.
type QuestRewardItem struct {
	ItemId uint32 `json:"itemId"`
	Amount int32  `json:"amount"`
}

// StartQuestPayload represents the payload required to start a quest.
type StartQuestPayload struct {
	CharacterId uint32            `json:"characterId"`       // CharacterId associated with the action
	WorldId     world.Id          `json:"worldId"`           // WorldId associated with the action
	QuestId     uint32            `json:"questId"`           // QuestId to start
	NpcId       uint32            `json:"npcId"`             // NpcId that started the quest
	Rewards     []QuestRewardItem `json:"rewards,omitempty"` // Item rewards granted alongside start (for downstream display)
}

// SetQuestProgressPayload represents the payload required to update quest progress.
type SetQuestProgressPayload struct {
	CharacterId uint32   `json:"characterId"` // CharacterId associated with the action
	WorldId     world.Id `json:"worldId"`     // WorldId associated with the action
	QuestId     uint32   `json:"questId"`     // QuestId to update progress for
	InfoNumber  uint32   `json:"infoNumber"`  // Progress info number/step to update
	Progress    string   `json:"progress"`    // Progress value to set
}

// ForfeitQuestPayload represents the payload required to forfeit a quest for a character.
type ForfeitQuestPayload struct {
	CharacterId uint32   `json:"characterId"` // CharacterId associated with the action
	WorldId     world.Id `json:"worldId"`     // WorldId associated with the action
	QuestId     uint32   `json:"questId"`     // QuestId to forfeit
	ShowEffect  bool     `json:"showEffect"`  // Render the forfeit effect on the client when true
}

// ApplyConsumableEffectPayload represents the payload required to apply consumable item effects to a character.
type ApplyConsumableEffectPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId to apply item effects to
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	ItemId      uint32     `json:"itemId"`      // Consumable item ID whose effects should be applied
}

// CancelConsumableEffectPayload represents the payload required to cancel consumable item effects on a character.
type CancelConsumableEffectPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId to cancel item effects for
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	ItemId      uint32     `json:"itemId"`      // Consumable item ID whose effects should be cancelled
}

// SendMessagePayload represents the payload required to send a system message to a character.
type SendMessagePayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId to send message to
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	MessageType string     `json:"messageType"` // Message type: "NOTICE", "POP_UP", "PINK_TEXT", "BLUE_TEXT"
	Message     string     `json:"message"`     // The message text to display
}

// FieldEffectPayload represents the payload for showing a field effect to a character.
type FieldEffectPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId to show effect to
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	Path        string     `json:"path"`        // Path to the field effect (e.g., "maplemap/enter/1020000")
}

// UiLockPayload represents the payload for locking or unlocking the UI for a character.
type UiLockPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId to lock/unlock UI for
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	Enable      bool       `json:"enable"`      // true = lock UI, false = unlock UI
}

// PlayPortalSoundPayload represents the payload for playing portal sound effect.
type PlayPortalSoundPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId to play sound for
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
}

// UpdateAreaInfoPayload represents the payload for updating a player's area info.
type UpdateAreaInfoPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId to update area info for
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	Area        uint16     `json:"area"`        // Area/info number
	Info        string     `json:"info"`        // Info string to display
}

// ShowInfoPayload represents the payload for showing an info/tutorial effect to a player.
type ShowInfoPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId to show info to
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	Path        string     `json:"path"`        // Path to the info effect
}

// ShowInfoTextPayload represents the payload for showing a text message to a player.
type ShowInfoTextPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId to show text to
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	Text        string     `json:"text"`        // Text message to display
}

// ShowIntroPayload represents the payload for showing an intro/direction effect to a player.
type ShowIntroPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId to show intro to
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	Path        string     `json:"path"`        // Path to the intro effect
}

// ShowHintPayload represents the payload for showing a hint box to a player.
type ShowHintPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId to show hint to
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	Hint        string     `json:"hint"`        // Hint text to display
	Width       uint16     `json:"width"`       // Width of the hint box (0 for auto)
	Height      uint16     `json:"height"`      // Height of the hint box (0 for auto)
}

// ShowGuideHintPayload represents the payload for showing a pre-defined guide hint by ID.
type ShowGuideHintPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId to show guide hint to
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	HintId      uint32     `json:"hintId"`      // Pre-defined hint ID
	Duration    uint32     `json:"duration"`    // Duration in milliseconds (default 7000ms if 0)
}

// BlockPortalPayload represents the payload for blocking a portal for a character.
type BlockPortalPayload struct {
	CharacterId uint32  `json:"characterId"` // CharacterId to block the portal for
	MapId       _map.Id `json:"mapId"`       // MapId where the portal is located
	PortalId    uint32  `json:"portalId"`    // PortalId to block
}

// UnblockPortalPayload represents the payload for unblocking a portal for a character.
type UnblockPortalPayload struct {
	CharacterId uint32  `json:"characterId"` // CharacterId to unblock the portal for
	MapId       _map.Id `json:"mapId"`       // MapId where the portal is located
	PortalId    uint32  `json:"portalId"`    // PortalId to unblock
}

// SpawnMonsterPayload represents the payload required to spawn monsters.
type SpawnMonsterPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId associated with the action
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	MapId       _map.Id    `json:"mapId"`       // MapId where monsters should spawn
	Instance    uuid.UUID  `json:"instance"`
	MonsterId   uint32     `json:"monsterId"` // MonsterId to spawn
	X           int16      `json:"x"`         // X coordinate for spawn
	Y           int16      `json:"y"`         // Y coordinate for spawn
	Team        int8       `json:"team"`      // Team assignment (optional, defaults to 0)
	Count       int        `json:"count"`     // Number of monsters to spawn (optional, defaults to 1)
}

// SpawnReactorDropsPayload represents the payload for spawning drops from a reactor.
type SpawnReactorDropsPayload struct {
	CharacterId    uint32     `json:"characterId"` // Character who triggered the reactor
	WorldId        world.Id   `json:"worldId"`     // WorldId for drop spawning
	ChannelId      channel.Id `json:"channelId"`   // ChannelId for drop spawning
	MapId          _map.Id    `json:"mapId"`       // MapId where drops should spawn
	Instance       uuid.UUID  `json:"instance"`
	ReactorId      uint32     `json:"reactorId"`      // ReactorId for fetching drop configuration
	Classification string     `json:"classification"` // Reactor classification string
	X              int16      `json:"x"`              // Reactor X position (drop origin)
	Y              int16      `json:"y"`              // Reactor Y position (drop origin)
	DropType       string     `json:"dropType"`       // "drop" (simultaneous) or "spray" (200ms intervals)
	Meso           bool       `json:"meso"`           // Whether meso drops are enabled
	MesoChance     uint32     `json:"mesoChance"`     // Meso drop probability (1/chance)
	MesoMin        uint32     `json:"mesoMin"`        // Minimum meso amount per drop
	MesoMax        uint32     `json:"mesoMax"`        // Maximum meso amount per drop
	MinItems       uint32     `json:"minItems"`       // Minimum guaranteed drops (padded with meso)
}

// ShowStoragePayload represents the payload required to show the storage UI to a character.
type ShowStoragePayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId to show storage to
	NpcId       uint32     `json:"npcId"`       // NpcId of the storage keeper
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	AccountId   uint32     `json:"accountId"`   // AccountId that owns the storage
}

// DepositToStoragePayload represents the payload required to deposit an item to account storage.
type DepositToStoragePayload struct {
	CharacterId   uint32    `json:"characterId"`   // CharacterId initiating the deposit
	AccountId     uint32    `json:"accountId"`     // AccountId that owns the storage
	WorldId       world.Id  `json:"worldId"`       // WorldId for the storage (storage is world-scoped)
	Slot          int16     `json:"slot"`          // Target slot in storage
	TemplateId    uint32    `json:"templateId"`    // Item template ID
	ReferenceId   uint32    `json:"referenceId"`   // Reference ID for the item data
	ReferenceType string    `json:"referenceType"` // Type of reference: "EQUIPABLE", "CONSUMABLE", "SETUP", "ETC", "CASH"
	Expiration    time.Time `json:"expiration"`    // Item expiration time
	Quantity      uint32    `json:"quantity"`      // Quantity (for stackables)
	OwnerId       uint32    `json:"ownerId"`       // Owner ID (for stackables)
	Flag          uint16    `json:"flag"`          // Item flag (for stackables)
}

// UpdateStorageMesosPayload represents the payload required to update mesos in account storage.
type UpdateStorageMesosPayload struct {
	CharacterId uint32   `json:"characterId"` // CharacterId initiating the update
	AccountId   uint32   `json:"accountId"`   // AccountId that owns the storage
	WorldId     world.Id `json:"worldId"`     // WorldId for the storage
	Operation   string   `json:"operation"`   // Operation: "SET", "ADD", "SUBTRACT"
	Mesos       uint32   `json:"mesos"`       // Mesos amount
}

// TransferToStoragePayload is the high-level payload for transferring an asset from character to storage.
// This step is expanded by saga-orchestrator into accept_to_storage + release_from_character.
type TransferToStoragePayload struct {
	TransactionId       uuid.UUID `json:"transactionId"`       // Saga transaction ID
	CharacterId         uint32    `json:"characterId"`         // Character initiating the transfer
	WorldId             world.Id  `json:"worldId"`             // World ID
	AccountId           uint32    `json:"accountId"`           // Account ID (storage owner)
	SourceSlot          int16     `json:"sourceSlot"`          // Slot in character inventory
	SourceInventoryType byte      `json:"sourceInventoryType"` // Character inventory type
	Quantity            uint32    `json:"quantity"`            // Quantity to transfer (0 = all)
}

// WithdrawFromStoragePayload is the high-level payload for withdrawing an asset from storage to character.
// This step is expanded by saga-orchestrator into accept_to_character + release_from_storage.
type WithdrawFromStoragePayload struct {
	TransactionId uuid.UUID `json:"transactionId"` // Saga transaction ID
	CharacterId   uint32    `json:"characterId"`   // Character receiving the item
	WorldId       world.Id  `json:"worldId"`       // World ID
	AccountId     uint32    `json:"accountId"`     // Account ID (storage owner)
	SourceSlot    int16     `json:"sourceSlot"`    // Slot in storage
	InventoryType byte      `json:"inventoryType"` // Target character inventory type
	Quantity      uint32    `json:"quantity"`      // Quantity to withdraw (0 = all)
}

// TransferToCashShopPayload is the high-level payload for transferring an asset from character to cash shop.
// This step is expanded by saga-orchestrator into accept_to_cash_shop + release_from_character.
type TransferToCashShopPayload struct {
	TransactionId       uuid.UUID `json:"transactionId"`       // Saga transaction ID
	CharacterId         uint32    `json:"characterId"`         // Character initiating the transfer
	AccountId           uint32    `json:"accountId"`           // Account ID (cash shop owner)
	CashId              int64     `json:"cashId"`              // Cash serial number of the item to transfer
	SourceInventoryType byte      `json:"sourceInventoryType"` // Character inventory type
	CompartmentType     byte      `json:"compartmentType"`     // Cash shop compartment type (1=Explorer, 2=Cygnus, 3=Legend)
}

// WithdrawFromCashShopPayload is the high-level payload for withdrawing an asset from cash shop to character.
// This step is expanded by saga-orchestrator into accept_to_character + release_from_cash_shop.
type WithdrawFromCashShopPayload struct {
	TransactionId   uuid.UUID `json:"transactionId"`   // Saga transaction ID
	CharacterId     uint32    `json:"characterId"`     // Character receiving the item
	AccountId       uint32    `json:"accountId"`       // Account ID (cash shop owner)
	CashId          uint64    `json:"cashId"`          // Cash serial number of the item to withdraw
	CompartmentType byte      `json:"compartmentType"` // Cash shop compartment type
	InventoryType   byte      `json:"inventoryType"`   // Target character inventory type
}

// ReleaseFromCharacterPayload represents the payload for the release_from_character action (internal step).
type ReleaseFromCharacterPayload struct {
	TransactionId uuid.UUID `json:"transactionId"` // Saga transaction ID
	CharacterId   uint32    `json:"characterId"`   // Character ID
	InventoryType byte      `json:"inventoryType"` // Inventory type (equip, use, etc.)
	AssetId       uint32    `json:"assetId"`       // Asset ID to release (populated during expansion)
	Quantity      uint32    `json:"quantity"`      // Quantity to release (0 = all)
}

// ReleaseFromStoragePayload represents the payload for the release_from_storage action (internal step).
type ReleaseFromStoragePayload struct {
	TransactionId uuid.UUID `json:"transactionId"` // Saga transaction ID
	WorldId       world.Id  `json:"worldId"`       // World ID
	AccountId     uint32    `json:"accountId"`     // Account ID
	CharacterId   uint32    `json:"characterId"`   // Character receiving the item
	AssetId       uint32    `json:"assetId"`       // Asset ID to release (populated during expansion)
	Quantity      uint32    `json:"quantity"`      // Quantity to release (0 = all)
}

// TransferToMtsPayload — expanded into release_from_character + accept_to_mts_listing.
// The seller supplies the listing/sale parameters at list time; expansion copies them
// into the accept step. The item snapshot itself is looked up from inventory during
// expansion (NOT carried here).
type TransferToMtsPayload struct {
	TransactionId       uuid.UUID  `json:"transactionId"`
	CharacterId         uint32     `json:"characterId"`
	SellerAccountId     uint32     `json:"sellerAccountId"` // Seller's cash-shop account, captured onto the listing for the settle-at-expiry seller-points credit
	WorldId             world.Id   `json:"worldId"`
	SourceInventoryType byte       `json:"sourceInventoryType"`
	AssetId             uint32     `json:"assetId"`
	Quantity            uint32     `json:"quantity"`
	ListingId           uuid.UUID  `json:"listingId"`
	SellerName          string     `json:"sellerName"`     // Seller character name for the listing
	SaleType            string     `json:"saleType"`       // Sale type (e.g. "buy_now", "auction")
	ListValue           uint32     `json:"listValue"`      // Seller's asking/starting price in NX
	BuyNowPrice         *uint32    `json:"buyNowPrice"`    // Optional buy-now price (nil = none)
	CommissionRate      float64    `json:"commissionRate"` // Commission rate applied at settlement
	Category            string     `json:"category"`       // Listing category
	SubCategory         string     `json:"subCategory"`    // Listing sub-category
	EndsAt              *time.Time `json:"endsAt"`         // Auction end time (nil = none)
	MinIncrement        uint32     `json:"minIncrement"`   // Minimum bid increment for auctions
	OfferWishSerial     uint32     `json:"offerWishSerial"`  // Want-ad serial an `offer` listing fulfills (0 for non-offers)
	OfferWishOwnerId    uint32     `json:"offerWishOwnerId"` // Want-ad poster id an `offer` listing fulfills (0 for non-offers)
}

// WithdrawFromMtsPayload — expanded into release_from_mts_holding + accept_to_character.
type WithdrawFromMtsPayload struct {
	TransactionId uuid.UUID `json:"transactionId"`
	CharacterId   uint32    `json:"characterId"`
	WorldId       world.Id  `json:"worldId"`
	HoldingId     uuid.UUID `json:"holdingId"`
	InventoryType byte      `json:"inventoryType"`
}

// AcceptToMtsListingPayload (atomic, dispatched to atlas-mts custody consumer).
// Carries everything atlas-mts needs to CREATE the listing row in `active` state:
// world/seller identity, the full item snapshot (looked up from inventory during
// expansion), and the seller's sale parameters. Mirrors AcceptToCashShopPayload
// carrying its item snapshot.
type AcceptToMtsListingPayload struct {
	TransactionId   uuid.UUID `json:"transactionId"`
	ListingId       uuid.UUID `json:"listingId"`
	WorldId         world.Id  `json:"worldId"`
	SellerId        uint32    `json:"sellerId"`
	SellerAccountId uint32    `json:"sellerAccountId"`
	SellerName      string    `json:"sellerName"`
	SaleType        string    `json:"saleType"`

	// Item snapshot
	TemplateId    uint32 `json:"templateId"`
	Quantity      uint32 `json:"quantity"`
	Strength      uint16 `json:"strength"`
	Dexterity     uint16 `json:"dexterity"`
	Intelligence  uint16 `json:"intelligence"`
	Luck          uint16 `json:"luck"`
	HP            uint16 `json:"hp"`
	MP            uint16 `json:"mp"`
	WeaponAttack  uint16 `json:"weaponAttack"`
	MagicAttack   uint16 `json:"magicAttack"`
	WeaponDefense uint16 `json:"weaponDefense"`
	MagicDefense  uint16 `json:"magicDefense"`
	Accuracy      uint16 `json:"accuracy"`
	Avoidability  uint16 `json:"avoidability"`
	Hands         uint16 `json:"hands"`
	Speed         uint16 `json:"speed"`
	Jump          uint16 `json:"jump"`
	Slots         uint16 `json:"slots"`
	Level         byte   `json:"level"`
	ItemLevel     byte   `json:"itemLevel"`
	ItemExp       uint32 `json:"itemExp"`
	RingId        uint32 `json:"ringId"`
	ViciousCount  uint32 `json:"viciousCount"`
	Flags         uint16 `json:"flags"`

	// Sale params
	ListValue      uint32     `json:"listValue"`
	BuyNowPrice    *uint32    `json:"buyNowPrice"`
	CommissionRate float64    `json:"commissionRate"`
	Category       string     `json:"category"`
	SubCategory    string     `json:"subCategory"`
	EndsAt         *time.Time `json:"endsAt"`
	MinIncrement   uint32     `json:"minIncrement"`

	// Offer link: which want-ad this `offer` listing fulfills (0 for non-offers).
	OfferWishSerial  uint32 `json:"offerWishSerial"`
	OfferWishOwnerId uint32 `json:"offerWishOwnerId"`
}

// ReleaseFromMtsHoldingPayload (atomic, dispatched to atlas-mts custody consumer).
type ReleaseFromMtsHoldingPayload struct {
	TransactionId uuid.UUID `json:"transactionId"`
	HoldingId     uuid.UUID `json:"holdingId"`
}

// MtsMoveListingToHoldingPayload (atomic custody step): moves a sold/settled
// listing's custody to the buyer's holding. The item snapshot is read from the
// listing row by atlas-mts (not carried here); the buyer/world identify the
// holding row to create.
type MtsMoveListingToHoldingPayload struct {
	TransactionId uuid.UUID `json:"transactionId"`
	ListingId     uuid.UUID `json:"listingId"`
	BuyerId       uint32    `json:"buyerId"`
	WorldId       world.Id  `json:"worldId"`
	// ResultKind carries which client result mode the sold notice should route to
	// (item / zzim / wish / auction_settle) so the channel picks the matching
	// CITC::OnNormalItemResult arm. Threaded from the buy/settle chain onto the
	// LISTING_SOLD event.
	ResultKind string `json:"resultKind"`
	// Price is the settled BASE price (list value / buy-now / winning bid) carried
	// through to the LISTING_SOLD event for the auction-settle SuccessBidInfo arm.
	Price uint32 `json:"price"`
}

// MtsSettlePurchasePayload (composite money-mover): debit buyer prepaid, credit seller points, move custody.
type MtsSettlePurchasePayload struct {
	TransactionId   uuid.UUID `json:"transactionId"`
	ListingId       uuid.UUID `json:"listingId"`
	WorldId         world.Id  `json:"worldId"` // World scoping for the buyer holding created on the final move step
	BuyerId         uint32    `json:"buyerId"`
	BuyerAccountId  uint32    `json:"buyerAccountId"`
	SellerId        uint32    `json:"sellerId"`
	SellerAccountId uint32    `json:"sellerAccountId"`
	MarkedUpPrice   int32     `json:"markedUpPrice"`
	ListValue       int32     `json:"listValue"`
	// ResultKind carries which client result mode the sold notice should route to
	// (item / zzim / wish); threaded onto the expanded move step's payload and then
	// the LISTING_SOLD event.
	ResultKind string `json:"resultKind"`
	// Price is the settled BASE price carried through to the LISTING_SOLD event.
	Price uint32 `json:"price"`
}

// MtsBidEscrowPayload (single-step wallet hold).
type MtsBidEscrowPayload struct {
	TransactionId   uuid.UUID `json:"transactionId"`
	ListingId       uuid.UUID `json:"listingId"`
	BidderId        uint32    `json:"bidderId"`
	BidderAccountId uint32    `json:"bidderAccountId"`
	Amount          int32     `json:"amount"` // negative to hold, positive to release
}

// RequestGuildNamePayload represents the payload required to request a guild name.
type RequestGuildNamePayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId associated with the action
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
}

// RequestGuildEmblemPayload represents the payload required to request a guild emblem change.
type RequestGuildEmblemPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId associated with the action
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
}

// RequestGuildDisbandPayload represents the payload required to request a guild disband.
type RequestGuildDisbandPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId associated with the action
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
}

// RequestGuildCapacityIncreasePayload represents the payload required to request a guild capacity increase.
type RequestGuildCapacityIncreasePayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId associated with the action
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
}

// CreateInvitePayload represents the payload required to create an invitation.
type CreateInvitePayload struct {
	InviteType   string   `json:"inviteType"`   // Type of invitation (e.g., "GUILD", "PARTY", "BUDDY")
	OriginatorId uint32   `json:"originatorId"` // ID of the character sending the invitation
	TargetId     uint32   `json:"targetId"`     // ID of the character receiving the invitation
	ReferenceId  uint32   `json:"referenceId"`  // ID of the entity being invited to (e.g., guild ID)
	WorldId      world.Id `json:"worldId"`      // WorldId associated with the action
}

// CharacterCreatePayload represents the payload required to create a character.
type CharacterCreatePayload struct {
	AccountId    uint32   `json:"accountId"` // AccountId associated with the action
	WorldId      world.Id `json:"worldId"`   // WorldId associated with the action
	Name         string   `json:"name"`      // Name of the character to create
	Gender       byte     `json:"gender"`
	Level        byte     `json:"level"`
	Strength     uint16   `json:"strength"`
	Dexterity    uint16   `json:"dexterity"`
	Intelligence uint16   `json:"intelligence"`
	Luck         uint16   `json:"luck"`
	JobId        job.Id   `json:"jobId"` // JobId to create the character with
	Hp           uint16   `json:"hp"`
	Mp           uint16   `json:"mp"`
	Face         uint32   `json:"face"`   // Face of the character
	Hair         uint32   `json:"hair"`   // Hair of the character
	Skin         byte     `json:"skin"`   // Skin of the character
	Top          uint32   `json:"top"`    // Top of the character
	Bottom       uint32   `json:"bottom"` // Bottom of the character
	Shoes        uint32   `json:"shoes"`  // Shoes of the character
	Weapon       uint32   `json:"weapon"` // Weapon of the character
	MapId        _map.Id  `json:"mapId"`  // Starting map ID for the character
	Gm           int      `json:"gm,omitempty"`
	Meso         uint32   `json:"meso,omitempty"`
}

// AwaitCharacterCreatedPayload represents the payload required to await character creation completion.
type AwaitCharacterCreatedPayload struct {
	CharacterName      string `json:"characterName"`                // Name of the character being created
	FollowUpSagaId     string `json:"followUpSagaId"`               // ID of the follow-up saga to trigger
	CreatedCharacterId uint32 `json:"createdCharacterId,omitempty"` // CharacterId once created (set by orchestrator)
}

// AwaitInventoryCreatedPayload represents the payload required to await
// inventory-compartment creation. The orchestrator's result-forwarding
// substitutes CharacterId=0 with the actual id emitted by handleCharacterCreatedEvent.
type AwaitInventoryCreatedPayload struct {
	CharacterId uint32 `json:"characterId"`
}

// StartInstanceTransportPayload represents the payload required to start an instance-based transport.
type StartInstanceTransportPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId to start transport for
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	RouteName   string     `json:"routeName"`   // Route name (resolved to UUID at runtime)
}

// SaveLocationPayload represents the payload required to save a character's current location.
type SaveLocationPayload struct {
	CharacterId  uint32     `json:"characterId"`  // CharacterId associated with the action
	WorldId      world.Id   `json:"worldId"`      // WorldId associated with the action
	ChannelId    channel.Id `json:"channelId"`    // ChannelId associated with the action
	LocationType string     `json:"locationType"` // Location type key (e.g., "FREE_MARKET", "EVENT")
	MapId        _map.Id    `json:"mapId"`        // MapId to save
	PortalId     uint32     `json:"portalId"`     // PortalId to save
}

// WarpToSavedLocationPayload represents the payload required to warp a character back to a saved location.
type WarpToSavedLocationPayload struct {
	CharacterId  uint32     `json:"characterId"`  // CharacterId associated with the action
	WorldId      world.Id   `json:"worldId"`      // WorldId associated with the action
	ChannelId    channel.Id `json:"channelId"`    // ChannelId associated with the action
	LocationType string     `json:"locationType"` // Location type key
}

// SelectGachaponRewardPayload represents the payload required to select a random reward from a gachapon.
type SelectGachaponRewardPayload struct {
	CharacterId uint32   `json:"characterId"` // CharacterId associated with the action
	WorldId     world.Id `json:"worldId"`     // WorldId associated with the action
	GachaponId  string   `json:"gachaponId"`  // Gachapon machine ID to select from
}

// EmitGachaponWinPayload represents the payload required to emit a gachapon win event.
type EmitGachaponWinPayload struct {
	CharacterId  uint32   `json:"characterId"`  // CharacterId who won
	WorldId      world.Id `json:"worldId"`      // WorldId for broadcasting
	ItemId       uint32   `json:"itemId"`       // Won item ID
	Quantity     uint32   `json:"quantity"`     // Won item quantity
	Tier         string   `json:"tier"`         // Reward tier (uncommon, rare)
	GachaponId   string   `json:"gachaponId"`   // Gachapon machine ID
	GachaponName string   `json:"gachaponName"` // Gachapon display name
}

// RegisterPartyQuestPayload represents the payload required to register a party for a party quest.
type RegisterPartyQuestPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId initiating the registration
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	MapId       _map.Id    `json:"mapId"`       // MapId where the registration NPC is
	QuestId     string     `json:"questId"`     // Party quest definition ID (e.g., "henesys_pq")
}

// WarpPartyQuestMembersToMapPayload represents the payload required to warp all party quest members to a map.
type WarpPartyQuestMembersToMapPayload struct {
	CharacterId uint32     `json:"characterId"` // Character initiating the warp (must be in a party)
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	MapId       _map.Id    `json:"mapId"`       // Destination map ID
	PortalId    uint32     `json:"portalId"`    // Destination portal ID
}

// LeavePartyQuestPayload represents the payload required to remove a character from their active party quest.
type LeavePartyQuestPayload struct {
	CharacterId uint32   `json:"characterId"` // CharacterId of the character leaving
	WorldId     world.Id `json:"worldId"`     // WorldId associated with the action
}

// EnterPartyQuestBonusPayload represents the payload for entering the bonus stage of a party quest.
type EnterPartyQuestBonusPayload struct {
	CharacterId uint32   `json:"characterId"` // CharacterId initiating bonus entry
	WorldId     world.Id `json:"worldId"`     // WorldId associated with the action
}

// UpdatePqCustomDataPayload represents the payload for updating party quest custom data.
type UpdatePqCustomDataPayload struct {
	InstanceId uuid.UUID         `json:"instanceId"`           // Party quest instance ID
	Updates    map[string]string `json:"updates,omitempty"`    // Key-value pairs to set
	Increments []string          `json:"increments,omitempty"` // Keys to increment
}

// HitReactorPayload represents the payload for programmatically hitting a reactor by name.
type HitReactorPayload struct {
	WorldId     world.Id   `json:"worldId"`     // WorldId of the reactor's field
	ChannelId   channel.Id `json:"channelId"`   // ChannelId of the reactor's field
	MapId       _map.Id    `json:"mapId"`       // MapId of the reactor's field
	Instance    uuid.UUID  `json:"instance"`    // Instance UUID of the reactor's field
	CharacterId uint32     `json:"characterId"` // CharacterId triggering the hit
	ReactorName string     `json:"reactorName"` // Reactor name to resolve via REST
}

// BroadcastPqMessagePayload represents the payload for broadcasting a message to PQ members.
type BroadcastPqMessagePayload struct {
	InstanceId  uuid.UUID `json:"instanceId"`  // Party quest instance ID
	MessageType string    `json:"messageType"` // Message type (e.g., "PINK_TEXT")
	Message     string    `json:"message"`     // Message text
}

// StageClearAttemptPqPayload represents the payload for attempting to clear the current PQ stage.
type StageClearAttemptPqPayload struct {
	InstanceId  uuid.UUID `json:"instanceId"`            // Party quest instance ID (used by reactor actions)
	CharacterId uint32    `json:"characterId,omitempty"` // Character ID for instance lookup (used by NPC conversations)
}

// FieldEffectWeatherPayload represents the payload for showing a weather effect to all characters in a field.
type FieldEffectWeatherPayload struct {
	WorldId   world.Id   `json:"worldId"`   // WorldId of the field
	ChannelId channel.Id `json:"channelId"` // ChannelId of the field
	MapId     _map.Id    `json:"mapId"`     // MapId of the field
	Instance  uuid.UUID  `json:"instance"`  // Instance UUID of the field
	ItemId    uint32     `json:"itemId"`    // Cash shop weather item ID
	Message   string     `json:"message"`   // Weather message text
	Duration  uint32     `json:"duration"`  // Duration in seconds
}

// SetAssetOwnerPayload represents the payload required to set the owner tag on an asset in a specific inventory slot.
type SetAssetOwnerPayload struct {
	CharacterId   uint32 `json:"characterId"`   // CharacterId associated with the action
	InventoryType byte   `json:"inventoryType"` // Type of inventory (1=equip, 2=use, 3=setup, 4=etc, 5=cash)
	Slot          int16  `json:"slot"`          // Slot of the asset to tag (negative for equipped slots, positive for inventory slots)
	Owner         string `json:"owner"`         // Owner name to set on the asset
}

// ApplyAssetLockPayload represents the payload required to apply a sealing lock (expiration) to an asset in a specific inventory slot.
type ApplyAssetLockPayload struct {
	CharacterId   uint32    `json:"characterId"`   // CharacterId associated with the action
	InventoryType byte      `json:"inventoryType"` // Type of inventory (1=equip, 2=use, 3=setup, 4=etc, 5=cash)
	Slot          int16     `json:"slot"`          // Slot of the asset to lock (negative for equipped slots, positive for inventory slots)
	Expiration    time.Time `json:"expiration"`    // Expiration time to apply to the asset
}

// IncubatorResultPayload represents the payload required to deliver the result of an incubator use to a character.
type IncubatorResultPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId associated with the action
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	ItemId      uint32     `json:"itemId"`      // ItemId of the resulting item
	Count       uint32     `json:"count"`       // Count of the resulting item
}
