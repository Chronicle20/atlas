package thread

const (
	EnvCommandTopic = "COMMAND_TOPIC_GUILD_THREAD"

	CommandTypeCreate      = "CREATE"
	CommandTypeUpdate      = "UPDATE"
	CommandTypeDelete      = "DELETE"
	CommandTypeAddReply    = "ADD_REPLY"
	CommandTypeDeleteReply = "DELETE_REPLY"
)

type command[E any] struct {
	GuildId     uint32 `json:"guildId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type createCommandBody struct {
	Notice     bool   `json:"notice"`
	Title      string `json:"title"`
	Message    string `json:"message"`
	EmoticonId uint32 `json:"emoticonId"`
}

type updateCommandBody struct {
	ThreadId   uint32 `json:"threadId"`
	Notice     bool   `json:"notice"`
	Title      string `json:"title"`
	Message    string `json:"message"`
	EmoticonId uint32 `json:"emoticonId"`
}

type deleteCommandBody struct {
	ThreadId uint32 `json:"threadId"`
}

type addReplyCommandBody struct {
	ThreadId uint32 `json:"threadId"`
	Message  string `json:"message"`
}

type deleteReplyCommandBody struct {
	ThreadId uint32 `json:"threadId"`
	ReplyId  uint32 `json:"replyId"`
}
