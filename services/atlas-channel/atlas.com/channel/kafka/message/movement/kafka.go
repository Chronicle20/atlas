package movement

const (
	EnvCommandMonsterMovement   = "COMMAND_TOPIC_MONSTER_MOVEMENT"
	EnvCommandCharacterMovement = "COMMAND_TOPIC_CHARACTER_MOVEMENT"
	EnvCommandPetMovement       = "COMMAND_TOPIC_PET_MOVEMENT"
)

type Command[E any] struct {
	WorldId    byte   `json:"worldId"`
	ChannelId  byte   `json:"channelId"`
	MapId      uint32 `json:"mapId"`
	ObjectId   uint64 `json:"objectId"`
	ObserverId uint32 `json:"observerId"`
	X          int16  `json:"x"`
	Y          int16  `json:"y"`
	Stance     byte   `json:"stance"`
}
