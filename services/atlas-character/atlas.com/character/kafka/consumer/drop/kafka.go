package drop

const (
	EnvEventTopicDropStatus = "EVENT_TOPIC_DROP_STATUS"
	StatusEventTypeReserved = "RESERVED"
)

type statusEvent[E any] struct {
	WorldId   byte   `json:"worldId"`
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
	DropId    uint32 `json:"dropId"`
	Type      string `json:"type"`
	Body      E      `json:"body"`
}

type reservedStatusEventBody struct {
	CharacterId uint32 `json:"characterId"`
	ItemId      uint32 `json:"itemId"`
	EquipmentId uint32 `json:"equipmentId"`
	Quantity    uint32 `json:"quantity"`
	Meso        uint32 `json:"meso"`
}
