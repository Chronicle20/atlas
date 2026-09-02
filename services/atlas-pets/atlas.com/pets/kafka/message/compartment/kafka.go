package compartment

import (
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

const (
	EnvCommandTopic topic.Token = "COMMAND_TOPIC_COMPARTMENT"
)

const (
	CommandChangeTemplate     = "CHANGE_TEMPLATE"
	CommandResetPetExpiration = "RESET_PET_EXPIRATION"
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

// ResetPetExpirationCommandBody sets a dried-up pet asset's expiration to an
// absolute instant. The asset is resolved by (CharacterId, PetId) — never by
// slot — mirroring ChangeTemplateCommandBody. SourceTemplateId names the
// consumed Water of Life so atlas-inventory can re-derive the ceiling itself;
// the sender is not a trust boundary. Absolute (not a duration) so a
// redelivered command is a no-op rather than a second grant.
//
// MIRROR: this struct is duplicated in
// services/atlas-inventory/.../kafka/message/compartment/kafka.go. The two live
// in separate Go modules, so a field name or json tag changed in one and not
// the other fails no build — it decodes into a zero-valued body at runtime.
type ResetPetExpirationCommandBody struct {
	PetId            uint32    `json:"petId"`
	Expiration       time.Time `json:"expiration"`
	SourceTemplateId uint32    `json:"sourceTemplateId"`
}
