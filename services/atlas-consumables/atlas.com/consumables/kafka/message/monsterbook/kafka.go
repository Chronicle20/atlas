package monsterbook

import "github.com/google/uuid"

const (
	EnvCommandTopic         = "COMMAND_TOPIC_MONSTER_BOOK"
	CommandTypeCardPickedUp = "CARD_PICKED_UP"
)

// Sources a CARD_PICKED_UP command can originate from. Informational only —
// atlas-monster-book registers the card the same way regardless.
const (
	// SourceDropPickup is a card registered by the consume-on-pickup divert,
	// straight off a field drop. This is the normal path.
	SourceDropPickup = "drop_pickup"
	// SourceItemUse is a card registered by using it from the USE inventory.
	// Cards are not supposed to land in the inventory at all, but one that did
	// (a pickup predating the consume-on-pickup flag, a GM grant, a trade) is
	// usable from the inventory and must still register.
	SourceItemUse = "item_use"
)

type Command[B any] struct {
	TenantId    uuid.UUID `json:"tenantId"`
	CharacterId uint32    `json:"characterId"`
	EventId     uuid.UUID `json:"eventId"`
	Type        string    `json:"type"`
	Body        B         `json:"body"`
}

type CardPickedUpBody struct {
	CardId uint32 `json:"cardId"`
	Source string `json:"source"`
}
