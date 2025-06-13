package compartment

const (
	EnvEventTopicStatus    = "EVENT_TOPIC_CASH_COMPARTMENT_STATUS"
	StatusEventTypeCreated = "CREATED"
	StatusEventTypeUpdated = "UPDATED"
	StatusEventTypeDeleted = "DELETED"
)

// StatusEvent represents a cash compartment status event
// According to the requirements, it should always contain the compartmentId and type
type StatusEvent[E any] struct {
	CompartmentId   string `json:"compartmentId"`
	Type            string `json:"type"`
	CompartmentType string `json:"compartmentType"`
	Body            E      `json:"body"`
}

// StatusEventCreatedBody contains information for compartment creation events
// According to the requirements, it should include the capacity
type StatusEventCreatedBody struct {
	Capacity uint32 `json:"capacity"`
}

// StatusEventUpdatedBody contains information for compartment update events
type StatusEventUpdatedBody struct {
	Capacity uint32 `json:"capacity"`
}

// StatusEventDeletedBody is an empty body for compartment deletion events
type StatusEventDeletedBody struct {
	// Empty body as no additional information is needed for deletion
}
