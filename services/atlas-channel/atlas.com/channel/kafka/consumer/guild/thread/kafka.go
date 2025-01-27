package thread

const (
	EnvStatusEventTopic = "EVENT_TOPIC_GUILD_THREAD_STATUS"

	StatusEventTypeCreated      = "CREATED"
	StatusEventTypeUpdated      = "UPDATED"
	StatusEventTypeDeleted      = "DELETED"
	StatusEventTypeReplyAdded   = "REPLY_ADDED"
	StatusEventTypeReplyDeleted = "REPLY_DELETED"
)

type statusEvent[E any] struct {
	WorldId  byte   `json:"worldId"`
	GuildId  uint32 `json:"guildId"`
	ActorId  uint32 `json:"actorId"`
	ThreadId uint32 `json:"threadId"`
	Type     string `json:"type"`
	Body     E      `json:"body"`
}

type createdStatusEventBody struct {
}

type updatedStatusEventBody struct {
}

type deletedStatusEventBody struct {
}

type replyAddedStatusEventBody struct {
	ReplyId uint32 `json:"replyId"`
}

type replyDeletedStatusEventBody struct {
	ReplyId uint32 `json:"replyId"`
}
