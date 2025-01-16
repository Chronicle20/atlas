package buddy

const (
	EnvCommandTopic          = "COMMAND_TOPIC_BUDDY_LIST"
	CommandTypeRequestAdd    = "REQUEST_ADD"
	CommandTypeRequestDelete = "REQUEST_DELETE"
)

type command[E any] struct {
	WorldId     byte   `json:"worldId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type requestAddBuddyCommandBody struct {
	CharacterId   uint32 `json:"characterId"`
	CharacterName string `json:"characterName"`
	Group         string `json:"group"`
}

type requestDeleteBuddyCommandBody struct {
	CharacterId uint32 `json:"characterId"`
}
