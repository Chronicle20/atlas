package transport_test

import (
	"atlas-query-aggregator/transport"
	"atlas-query-aggregator/transport/mock"
	"errors"
	"testing"

	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/google/uuid"
)

// createTestRoute is a helper to create transport models for testing
func createTestRoute(state string) transport.Model {
	rm := transport.RestModel{
		Id:    uuid.New().String(),
		Name:  "Test Route",
		State: state,
	}
	route, _ := transport.Extract(rm)
	return route
}

func TestProcessorMock_GetRouteByStartMap_Success(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetRouteByStartMapFunc: func(mapId _map.Id) (transport.Model, error) {
			return createTestRoute("open_entry"), nil
		},
	}

	route, err := mockProcessor.GetRouteByStartMap(101000300)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if route.State() != "open_entry" {
		t.Errorf("Expected State=open_entry, got %s", route.State())
	}

	if !route.IsOpenEntry() {
		t.Error("Expected IsOpenEntry()=true")
	}
}

func TestProcessorMock_GetRouteByStartMap_ClosedEntry(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetRouteByStartMapFunc: func(mapId _map.Id) (transport.Model, error) {
			return createTestRoute("closed"), nil
		},
	}

	route, err := mockProcessor.GetRouteByStartMap(101000300)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if route.IsOpenEntry() {
		t.Error("Expected IsOpenEntry()=false for closed route")
	}
}

func TestProcessorMock_GetRouteByStartMap_NotFound(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetRouteByStartMapFunc: func(mapId _map.Id) (transport.Model, error) {
			return transport.Model{}, errors.New("no routes found")
		},
	}

	_, err := mockProcessor.GetRouteByStartMap(999999)
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestProcessorMock_GetRouteByStartMap_ServiceError(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{
		GetRouteByStartMapFunc: func(mapId _map.Id) (transport.Model, error) {
			return transport.Model{}, errors.New("transport service unavailable")
		},
	}

	_, err := mockProcessor.GetRouteByStartMap(101000300)
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if err.Error() != "transport service unavailable" {
		t.Errorf("Expected error message 'transport service unavailable', got '%s'", err.Error())
	}
}

func TestProcessorMock_DefaultBehavior(t *testing.T) {
	mockProcessor := &mock.ProcessorImpl{}

	// Test default GetRouteByStartMap returns error (no routes)
	_, err := mockProcessor.GetRouteByStartMap(100000000)
	if err == nil {
		t.Error("Expected error from default GetRouteByStartMap")
	}
}

func TestModel_IsOpenEntry(t *testing.T) {
	tests := []struct {
		name     string
		state    string
		expected bool
	}{
		{"open_entry state is open", "open_entry", true},
		{"closed state is not open", "closed", false},
		{"departing state is not open", "departing", false},
		{"arrived state is not open", "arrived", false},
		{"empty state is not open", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := createTestRoute(tt.state)
			if route.IsOpenEntry() != tt.expected {
				t.Errorf("Expected IsOpenEntry()=%v for state=%s, got %v", tt.expected, tt.state, route.IsOpenEntry())
			}
		})
	}
}
