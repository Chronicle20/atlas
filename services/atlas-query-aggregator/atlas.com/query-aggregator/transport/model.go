package transport

import (
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/google/uuid"
)

// Model is the domain model for a transport route
type Model struct {
	id         uuid.UUID
	name       string
	state      string
	startMapId _map.Id
}

// Id returns the route ID
func (m Model) Id() uuid.UUID {
	return m.id
}

// Name returns the route name
func (m Model) Name() string {
	return m.name
}

// State returns the current route state
func (m Model) State() string {
	return m.state
}

// StartMapId returns the starting map ID
func (m Model) StartMapId() _map.Id {
	return m.startMapId
}

// IsOpenEntry returns true if the route is accepting passengers
func (m Model) IsOpenEntry() bool {
	return m.state == "open_entry"
}
