package transport

// RouteState represents the state of a transport route
type RouteState string

const (
	// OutOfService indicates that the vessel is not in service
	OutOfService RouteState = "out_of_service"

	// AwaitingReturn indicates that the vessel is not yet available
	AwaitingReturn RouteState = "awaiting_return"

	// OpenEntry indicates that players can board
	OpenEntry RouteState = "open_entry"

	// LockedEntry indicates that boarding is closed and the vessel is in pre-departure phase
	LockedEntry RouteState = "locked_entry"

	// InTransit indicates that characters are in the en-route map
	InTransit RouteState = "in_transit"
)
