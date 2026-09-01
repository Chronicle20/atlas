package playernpc

// EventType names the four design §7/§8.3 domain events a state-changing
// operation can emit.
type EventType string

const (
	EventTypeDeployed     EventType = "DEPLOYED"
	EventTypeUpdated      EventType = "UPDATED"
	EventTypeRemoved      EventType = "REMOVED"
	EventTypeRepositioned EventType = "REPOSITIONED"
)
