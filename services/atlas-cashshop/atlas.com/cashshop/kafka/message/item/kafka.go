package item

const (
	EnvCommandTopic = "COMMAND_TOPIC_CASH_ITEM"
	EnvStatusTopic  = "STATUS_TOPIC_CASH_ITEM"

	CommandCreate = "CREATE"

	StatusCreated = "CREATED"
)

type Command[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type CreateCommandBody struct {
	TemplateId  uint32 `json:"templateId"`
	Quantity    uint32 `json:"quantity"`
	PurchasedBy uint32 `json:"purchasedBy"`
}

type StatusEvent[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type StatusEventCreatedBody struct {
	CashId      int64  `json:"cashId"`
	TemplateId  uint32 `json:"templateId"`
	Quantity    uint32 `json:"quantity"`
	PurchasedBy uint32 `json:"purchasedBy"`
	Flag        uint16 `json:"flag"`
}
