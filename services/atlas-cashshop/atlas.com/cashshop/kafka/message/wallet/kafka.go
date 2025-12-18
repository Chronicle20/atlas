package wallet

const (
	EnvEventTopicStatus        = "EVENT_TOPIC_WALLET_STATUS"
	StatusEventTypeCreated     = "CREATED"
	StatusEventTypeUpdated     = "UPDATED"
	StatusEventTypeDeleted     = "DELETED"
)

type StatusEvent[E any] struct {
	AccountId uint32 `json:"accountId"`
	Type      string `json:"type"`
	Body      E      `json:"body"`
}

type StatusEventCreatedBody struct {
	Credit  uint32 `json:"credit"`
	Points  uint32 `json:"points"`
	Prepaid uint32 `json:"prepaid"`
}

type StatusEventUpdatedBody struct {
	Credit  uint32 `json:"credit"`
	Points  uint32 `json:"points"`
	Prepaid uint32 `json:"prepaid"`
}

type StatusEventDeletedBody struct {
	// Empty body as no additional information is needed for deletion
}