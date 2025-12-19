package thread

const (
	EnvCommandTopic = "COMMAND_TOPIC_GUILD_THREAD"

	CommandTypeCreate      = "CREATE"
	CommandTypeUpdate      = "UPDATE"
	CommandTypeDelete      = "DELETE"
	CommandTypeAddReply    = "ADD_REPLY"
	CommandTypeDeleteReply = "DELETE_REPLY"
)

type Command[E any] struct {
	GuildId     uint32 `json:"guildId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type CreateCommandBody struct {
	Notice     bool   `json:"notice"`
	Title      string `json:"title"`
	Message    string `json:"message"`
	EmoticonId uint32 `json:"emoticonId"`
}

type UpdateCommandBody struct {
	ThreadId   uint32 `json:"threadId"`
	Notice     bool   `json:"notice"`
	Title      string `json:"title"`
	Message    string `json:"message"`
	EmoticonId uint32 `json:"emoticonId"`
}

type DeleteCommandBody struct {
	ThreadId uint32 `json:"threadId"`
}

type AddReplyCommandBody struct {
	ThreadId uint32 `json:"threadId"`
	Message  string `json:"message"`
}

type DeleteReplyCommandBody struct {
	ThreadId uint32 `json:"threadId"`
	ReplyId  uint32 `json:"replyId"`
}

const (
	EnvStatusEventTopic = "EVENT_TOPIC_GUILD_THREAD_STATUS"

	StatusEventTypeCreated      = "CREATED"
	StatusEventTypeUpdated      = "UPDATED"
	StatusEventTypeDeleted      = "DELETED"
	StatusEventTypeReplyAdded   = "REPLY_ADDED"
	StatusEventTypeReplyDeleted = "REPLY_DELETED"
)

type StatusEvent[E any] struct {
	WorldId  byte   `json:"worldId"`
	GuildId  uint32 `json:"guildId"`
	ThreadId uint32 `json:"threadId"`
	ActorId  uint32 `json:"actorId"`
	Type     string `json:"type"`
	Body     E      `json:"body"`
}

type CreatedStatusEventBody struct {
}

type UpdatedStatusEventBody struct {
}

type DeletedStatusEventBody struct {
}

type ReplyAddedStatusEventBody struct {
	ReplyId uint32 `json:"replyId"`
}

type ReplyDeletedStatusEventBody struct {
	ReplyId uint32 `json:"replyId"`
}
