package transport

import (
	"encoding/json"
	"github.com/sirupsen/logrus"
	"time"
)

// RouteStateEvent represents a Kafka event for route state transitions
type RouteStateEvent struct {
	RouteID   string    `json:"routeId"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

// NewRouteStateEvent creates a new route state event
func NewRouteStateEvent(routeID string, status RouteState, timestamp time.Time) RouteStateEvent {
	return RouteStateEvent{
		RouteID:   routeID,
		Status:    string(status),
		Timestamp: timestamp,
	}
}

// EmitRouteStateEvent emits a route state event to Kafka
func EmitRouteStateEvent(l logrus.FieldLogger, routeID string, status RouteState) {
	// Create the event
	event := NewRouteStateEvent(routeID, status, time.Now())

	// Marshal the event to JSON for logging
	eventJSON, _ := json.Marshal(event)
	l.Infof("Route state transition: %s", string(eventJSON))

	// TODO: Implement Kafka integration
	// The Kafka broker configuration should be externalized (e.g., via environment variables)
	// Topic name should be configurable (default: route.state.transitions)
	// Kafka production errors should be logged but must not crash the service
	// Use segmentio/kafka-go as the Kafka client
}
