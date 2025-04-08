package asset

const (
	EnvEventTopicStatus    = "EVENT_TOPIC_ASSET_STATUS"
	StatusEventTypeCreated = "CREATED"
	StatusEventTypeDeleted = "DELETED"
)

type StatusEvent[E any] struct {
	AssetId uint32 `json:"asset_id"`
	Type    string `json:"type"`
	Body    E      `json:"body"`
}

type CreatedStatusBody struct {
}

type DeletedStatusEventBody struct {
}
