package inventory

const (
	EnvEventTopicStatus    = "EVENT_TOPIC_CASH_INVENTORY_STATUS"
	StatusEventTypeCreated = "CREATED"
	StatusEventTypeUpdated = "UPDATED"
	StatusEventTypeDeleted = "DELETED"
)

// StatusEvent represents a cash inventory status event
// According to the requirements, it should always contain the accountId
// and have an empty body for created/deleted events
type StatusEvent[E any] struct {
	AccountId uint32 `json:"accountId"`
	Type      string `json:"type"`
	Body      E      `json:"body"`
}

// StatusEventCreatedBody is an empty body for inventory creation events
type StatusEventCreatedBody struct {
	// Empty body as no additional information is needed for creation
}

// StatusEventDeletedBody is an empty body for inventory deletion events
type StatusEventDeletedBody struct {
	// Empty body as no additional information is needed for deletion
}
