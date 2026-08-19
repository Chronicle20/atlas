package parcel

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// COMMAND_TOPIC_PARCEL carries commands dispatched to atlas-channel that open
// the Duey parcel dialog for a character. This mirrors the storage service's
// COMMAND_TOPIC_STORAGE / ShowStorageCommand shape (task-241 Task 19); it is a
// distinct topic from COMMAND_TOPIC_PARCEL_CUSTODY
// (kafka/message/parcel/custody/kafka.go), which carries the atomic custody
// commands dispatched to atlas-parcel.
const (
	EnvCommandTopic       = "COMMAND_TOPIC_PARCEL"
	CommandTypeShowParcel = "SHOW_PARCEL"
)

// ShowParcelCommand is sent to atlas-channel to display the Duey parcel
// dialog. Quick discriminates the two entry points: false is the NPC path
// (PARCEL[OPEN] with the mailbox), true is the Quick Delivery Ticket path
// (PARCEL[OPEN_QUICK], mode 0x1A, no list — task-241 design §5.2, §9.5).
type ShowParcelCommand struct {
	TransactionId uuid.UUID  `json:"transactionId"`
	WorldId       world.Id   `json:"worldId"`
	ChannelId     channel.Id `json:"channelId"`
	CharacterId   uint32     `json:"characterId"`
	NpcId         uint32     `json:"npcId"`
	Quick         bool       `json:"quick"`
	Type          string     `json:"type"`
}
