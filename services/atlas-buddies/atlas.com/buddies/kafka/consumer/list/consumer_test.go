package list

import (
	list2 "atlas-buddies/kafka/message/list"
	"testing"

	"github.com/sirupsen/logrus"
)

// TestHandleIncreaseCapacityCommandTypeGuard tests that the handler correctly guards against wrong command types
func TestHandleIncreaseCapacityCommandTypeGuard(t *testing.T) {
	// Create a mock logger that captures log output
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Test that handler function can be created and type guards work correctly
	handler := handleIncreaseCapacityCommand(nil) // nil db is ok for type guard test

	// Test commands with wrong types - these should be ignored (early return)
	wrongTypeCommands := []string{
		list2.CommandTypeCreate,
		list2.CommandTypeRequestAdd,
		list2.CommandTypeRequestDelete,
		"UNKNOWN_TYPE",
		"",
	}

	for _, commandType := range wrongTypeCommands {
		t.Run("Command type: "+commandType, func(t *testing.T) {
			command := list2.Command[list2.IncreaseCapacityCommandBody]{
				WorldId:     1,
				CharacterId: 12345,
				Type:        commandType,
				Body: list2.IncreaseCapacityCommandBody{
					NewCapacity: 30,
				},
			}

			// This should not panic and should return early for wrong types
			// Note: handler doesn't return errors, it just logs them
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("Handler panicked for command type %s: %v", commandType, r)
					}
				}()
				handler(logger, nil, command) // nil context is ok for type guard test
			}()
		})
	}
}

// TestHandleIncreaseCapacityCommandCreation tests that the handler function can be created
func TestHandleIncreaseCapacityCommandCreation(t *testing.T) {
	// Test that we can create the handler function without errors
	handler := handleIncreaseCapacityCommand(nil)
	if handler == nil {
		t.Error("Expected handler function to be created, got nil")
	}
}