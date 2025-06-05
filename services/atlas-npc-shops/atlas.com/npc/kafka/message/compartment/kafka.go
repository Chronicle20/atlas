package compartment

import "time"

const (
	EnvCommandTopic    = "COMMAND_TOPIC_COMPARTMENT"
	CommandDestroy     = "DESTROY"
	CommandCreateAsset = "CREATE_ASSET"
	CommandRecharge    = "RECHARGE"
)

type Command[E any] struct {
	CharacterId   uint32 `json:"characterId"`
	InventoryType byte   `json:"inventoryType"`
	Type          string `json:"type"`
	Body          E      `json:"body"`
}

type DestroyCommandBody struct {
	Slot     int16  `json:"slot"`
	Quantity uint32 `json:"quantity"`
}

type CreateAssetCommandBody struct {
	TemplateId   uint32    `json:"templateId"`
	Quantity     uint32    `json:"quantity"`
	Expiration   time.Time `json:"expiration"`
	OwnerId      uint32    `json:"ownerId"`
	Flag         uint16    `json:"flag"`
	Rechargeable uint64    `json:"rechargeable"`
}

type RechargeCommandBody struct {
	Slot     int16  `json:"slot"`
	Quantity uint32 `json:"quantity"`
}
