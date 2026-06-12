package compartment

import "github.com/google/uuid"

const (
	EnvCommandTopic       = "COMMAND_TOPIC_COMPARTMENT"
	CommandChangeTemplate = "CHANGE_TEMPLATE"
)

type Command[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	CharacterId   uint32    `json:"characterId"`
	InventoryType byte      `json:"inventoryType"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type ChangeTemplateCommandBody struct {
	PetId         uint32 `json:"petId"`
	NewTemplateId uint32 `json:"newTemplateId"`
}
