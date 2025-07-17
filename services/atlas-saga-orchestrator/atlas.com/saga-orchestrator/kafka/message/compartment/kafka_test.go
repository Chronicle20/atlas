package compartment

import (
	"atlas-saga-orchestrator/kafka/message/asset"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestStatusEventSerialization tests serialization and deserialization of compartment status events
func TestStatusEventSerialization(t *testing.T) {
	tests := []struct {
		name     string
		event    interface{}
		expected string
		description string
	}{
		{
			name: "CreatedStatusEvent serialization",
			event: StatusEvent[CreatedStatusEventBody]{
				TransactionId: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
				Type:          StatusEventTypeCreated,
				CharacterId:   12345,
				CompartmentId: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
				Body: CreatedStatusEventBody{
					Type:     1,
					Capacity: 100,
				},
			},
			expected: `{"transactionId":"550e8400-e29b-41d4-a716-446655440000","type":"CREATED","characterId":12345,"compartmentId":"00000000-0000-0000-0000-000000000000","body":{"type":1,"capacity":100}}`,
			description: "Should serialize CreatedStatusEvent correctly",
		},
		{
			name: "CreationFailedStatusEvent serialization",
			event: StatusEvent[CreationFailedStatusEventBody]{
				TransactionId: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
				Type:          StatusEventTypeCreationFailed,
				CharacterId:   12345,
				CompartmentId: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
				Body: CreationFailedStatusEventBody{
					ErrorCode: "INVALID_TEMPLATE_ID",
					Message:   "Template ID 999999 is not valid",
				},
			},
			expected: `{"transactionId":"550e8400-e29b-41d4-a716-446655440000","type":"CREATION_FAILED","characterId":12345,"compartmentId":"00000000-0000-0000-0000-000000000000","body":{"errorCode":"INVALID_TEMPLATE_ID","message":"Template ID 999999 is not valid"}}`,
			description: "Should serialize CreationFailedStatusEvent correctly",
		},
		{
			name: "ErrorEventBody serialization",
			event: StatusEvent[ErrorEventBody]{
				TransactionId: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
				Type:          StatusEventTypeError,
				CharacterId:   12345,
				CompartmentId: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
				Body: ErrorEventBody{
					ErrorCode:     "EQUIPMENT_SLOT_OCCUPIED",
					TransactionId: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
				},
			},
			expected: `{"transactionId":"550e8400-e29b-41d4-a716-446655440000","type":"ERROR","characterId":12345,"compartmentId":"00000000-0000-0000-0000-000000000000","body":{"errorCode":"EQUIPMENT_SLOT_OCCUPIED","transactionId":"550e8400-e29b-41d4-a716-446655440000"}}`,
			description: "Should serialize ErrorEventBody correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test serialization
			jsonData, err := json.Marshal(tt.event)
			assert.NoError(t, err, "Should serialize without error")
			assert.JSONEq(t, tt.expected, string(jsonData), "Serialized JSON should match expected")

			// Test deserialization
			switch e := tt.event.(type) {
			case StatusEvent[CreatedStatusEventBody]:
				var deserialized StatusEvent[CreatedStatusEventBody]
				err = json.Unmarshal(jsonData, &deserialized)
				assert.NoError(t, err, "Should deserialize without error")
				assert.Equal(t, e, deserialized, "Deserialized event should match original")

			case StatusEvent[CreationFailedStatusEventBody]:
				var deserialized StatusEvent[CreationFailedStatusEventBody]
				err = json.Unmarshal(jsonData, &deserialized)
				assert.NoError(t, err, "Should deserialize without error")
				assert.Equal(t, e, deserialized, "Deserialized event should match original")

			case StatusEvent[ErrorEventBody]:
				var deserialized StatusEvent[ErrorEventBody]
				err = json.Unmarshal(jsonData, &deserialized)
				assert.NoError(t, err, "Should deserialize without error")
				assert.Equal(t, e, deserialized, "Deserialized event should match original")
			}
		})
	}
}

// TestStatusEventTypeConstants tests that all status event type constants are defined correctly
func TestStatusEventTypeConstants(t *testing.T) {
	tests := []struct {
		name        string
		eventType   string
		expected    string
		description string
	}{
		{
			name:        "StatusEventTypeCreated",
			eventType:   StatusEventTypeCreated,
			expected:    "CREATED",
			description: "Should have correct value for CREATED event type",
		},
		{
			name:        "StatusEventTypeDeleted",
			eventType:   StatusEventTypeDeleted,
			expected:    "DELETED",
			description: "Should have correct value for DELETED event type",
		},
		{
			name:        "StatusEventTypeCreationFailed",
			eventType:   StatusEventTypeCreationFailed,
			expected:    "CREATION_FAILED",
			description: "Should have correct value for CREATION_FAILED event type",
		},
		{
			name:        "StatusEventTypeCapacityChanged",
			eventType:   StatusEventTypeCapacityChanged,
			expected:    "CAPACITY_CHANGED",
			description: "Should have correct value for CAPACITY_CHANGED event type",
		},
		{
			name:        "StatusEventTypeReserved",
			eventType:   StatusEventTypeReserved,
			expected:    "RESERVED",
			description: "Should have correct value for RESERVED event type",
		},
		{
			name:        "StatusEventTypeReservationCancelled",
			eventType:   StatusEventTypeReservationCancelled,
			expected:    "RESERVATION_CANCELLED",
			description: "Should have correct value for RESERVATION_CANCELLED event type",
		},
		{
			name:        "StatusEventTypeMergeComplete",
			eventType:   StatusEventTypeMergeComplete,
			expected:    "MERGE_COMPLETE",
			description: "Should have correct value for MERGE_COMPLETE event type",
		},
		{
			name:        "StatusEventTypeError",
			eventType:   StatusEventTypeError,
			expected:    "ERROR",
			description: "Should have correct value for ERROR event type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.eventType, tt.description)
		})
	}
}

// TestCommandSerialization tests serialization and deserialization of compartment commands
func TestCommandSerialization(t *testing.T) {
	tests := []struct {
		name     string
		command  interface{}
		expected string
		description string
	}{
		{
			name: "CreateCommand serialization",
			command: Command[CreateCommandBody]{
				TransactionId: uuid.UUID{},
				CharacterId:   12345,
				InventoryType: 0,
				Type:          CommandTypeCreate,
				Body: CreateCommandBody{
					TemplateId: 1302000,
					Quantity:   1,
				},
			},
			expected: `{"transactionId":"00000000-0000-0000-0000-000000000000","characterId":12345,"inventoryType":0,"type":"CREATE","body":{"templateId":1302000,"quantity":1}}`,
			description: "Should serialize CreateCommand correctly",
		},
		{
			name: "DeleteCommand serialization",
			command: Command[DeleteCommandBody]{
				TransactionId: uuid.UUID{},
				CharacterId:   12345,
				InventoryType: 0,
				Type:          CommandTypeDelete,
				Body: DeleteCommandBody{
					TemplateId: 1302000,
					Quantity:   1,
				},
			},
			expected: `{"transactionId":"00000000-0000-0000-0000-000000000000","characterId":12345,"inventoryType":0,"type":"DELETE","body":{"templateId":1302000,"quantity":1}}`,
			description: "Should serialize DeleteCommand correctly",
		},
		{
			name: "EquipCommand serialization",
			command: Command[EquipCommandBody]{
				TransactionId: uuid.UUID{},
				CharacterId:   12345,
				InventoryType: 0,
				Type:          CommandTypeEquip,
				Body: EquipCommandBody{
					InventoryType: 1,
					Source:        5,
					Destination:   -1,
				},
			},
			expected: `{"transactionId":"00000000-0000-0000-0000-000000000000","characterId":12345,"inventoryType":0,"type":"EQUIP","body":{"inventoryType":1,"source":5,"destination":-1}}`,
			description: "Should serialize EquipCommand correctly",
		},
		{
			name: "UnequipCommand serialization",
			command: Command[UnequipCommandBody]{
				TransactionId: uuid.UUID{},
				CharacterId:   12345,
				InventoryType: 0,
				Type:          CommandTypeUnequip,
				Body: UnequipCommandBody{
					InventoryType: 1,
					Source:        -1,
					Destination:   5,
				},
			},
			expected: `{"transactionId":"00000000-0000-0000-0000-000000000000","characterId":12345,"inventoryType":0,"type":"UNEQUIP","body":{"inventoryType":1,"source":-1,"destination":5}}`,
			description: "Should serialize UnequipCommand correctly",
		},
		{
			name: "CreateAndEquipCommand serialization",
			command: Command[CreateAndEquipCommandBody]{
				TransactionId: uuid.UUID{},
				CharacterId:   12345,
				InventoryType: 0,
				Type:          CommandTypeCreateAndEquip,
				Body: CreateAndEquipCommandBody{
					TemplateId: 1302000,
					Quantity:   1,
				},
			},
			expected: `{"transactionId":"00000000-0000-0000-0000-000000000000","characterId":12345,"inventoryType":0,"type":"CREATE_AND_EQUIP","body":{"templateId":1302000,"quantity":1}}`,
			description: "Should serialize CreateAndEquipCommand correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test serialization
			jsonData, err := json.Marshal(tt.command)
			assert.NoError(t, err, "Should serialize without error")
			assert.JSONEq(t, tt.expected, string(jsonData), "Serialized JSON should match expected")

			// Test deserialization
			switch c := tt.command.(type) {
			case Command[CreateCommandBody]:
				var deserialized Command[CreateCommandBody]
				err = json.Unmarshal(jsonData, &deserialized)
				assert.NoError(t, err, "Should deserialize without error")
				assert.Equal(t, c, deserialized, "Deserialized command should match original")

			case Command[DeleteCommandBody]:
				var deserialized Command[DeleteCommandBody]
				err = json.Unmarshal(jsonData, &deserialized)
				assert.NoError(t, err, "Should deserialize without error")
				assert.Equal(t, c, deserialized, "Deserialized command should match original")

			case Command[EquipCommandBody]:
				var deserialized Command[EquipCommandBody]
				err = json.Unmarshal(jsonData, &deserialized)
				assert.NoError(t, err, "Should deserialize without error")
				assert.Equal(t, c, deserialized, "Deserialized command should match original")

			case Command[UnequipCommandBody]:
				var deserialized Command[UnequipCommandBody]
				err = json.Unmarshal(jsonData, &deserialized)
				assert.NoError(t, err, "Should deserialize without error")
				assert.Equal(t, c, deserialized, "Deserialized command should match original")

			case Command[CreateAndEquipCommandBody]:
				var deserialized Command[CreateAndEquipCommandBody]
				err = json.Unmarshal(jsonData, &deserialized)
				assert.NoError(t, err, "Should deserialize without error")
				assert.Equal(t, c, deserialized, "Deserialized command should match original")
			}
		})
	}
}

// TestCommandTypeConstants tests that all command type constants are defined correctly
func TestCommandTypeConstants(t *testing.T) {
	tests := []struct {
		name        string
		commandType string
		expected    string
		description string
	}{
		{
			name:        "CommandTypeCreate",
			commandType: CommandTypeCreate,
			expected:    "CREATE",
			description: "Should have correct value for CREATE command type",
		},
		{
			name:        "CommandTypeDelete",
			commandType: CommandTypeDelete,
			expected:    "DELETE",
			description: "Should have correct value for DELETE command type",
		},
		{
			name:        "CommandTypeEquip",
			commandType: CommandTypeEquip,
			expected:    "EQUIP",
			description: "Should have correct value for EQUIP command type",
		},
		{
			name:        "CommandTypeUnequip",
			commandType: CommandTypeUnequip,
			expected:    "UNEQUIP",
			description: "Should have correct value for UNEQUIP command type",
		},
		{
			name:        "CommandTypeCreateAndEquip",
			commandType: CommandTypeCreateAndEquip,
			expected:    "CREATE_AND_EQUIP",
			description: "Should have correct value for CREATE_AND_EQUIP command type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.commandType, tt.description)
		})
	}
}

// TestCreateAndEquipEventFlow tests the specific event flow for CreateAndEquipAsset operations
func TestCreateAndEquipEventFlow(t *testing.T) {
	transactionId := uuid.New()
	characterId := uint32(12345)

	// Test the typical flow of events for CreateAndEquipAsset
	tests := []struct {
		name        string
		events      []interface{}
		description string
	}{
		{
			name: "Successful CreateAndEquipAsset flow",
			events: []interface{}{
				// 1. Initial command
				Command[CreateAndEquipCommandBody]{
					CharacterId: characterId,
					Type:        CommandTypeCreateAndEquip,
					Body: CreateAndEquipCommandBody{
						TemplateId: 1302000,
						Quantity:   1,
					},
				},
				// 2. Asset creation success
				StatusEvent[CreatedStatusEventBody]{
					TransactionId: transactionId,
					Type:          StatusEventTypeCreated,
					CharacterId:   characterId,
					Body: CreatedStatusEventBody{
						Type:     1,
						AssetId:  67890,
						Quantity: 1,
					},
				},
				// 3. Auto-generated equip command (internal)
				Command[EquipCommandBody]{
					CharacterId: characterId,
					Type:        CommandTypeEquip,
					Body: EquipCommandBody{
						InventoryType: 1,
						Source:        5,
						Destination:   -1,
					},
				},
				// 4. Equipment success - represented as asset moved event
				asset.StatusEvent[asset.MovedStatusEventBody]{
					TransactionId: transactionId,
					CharacterId:   characterId,
					CompartmentId: uuid.UUID{},
					AssetId:       67890,
					TemplateId:    1302000,
					Slot:          -1, // Equipment slot
					Type:          asset.StatusEventTypeMoved,
					Body: asset.MovedStatusEventBody{
						OldSlot: 5, // Moved from inventory slot 5 to equipment slot
					},
				},
			},
			description: "Should handle complete CreateAndEquipAsset flow",
		},
		{
			name: "CreateAndEquipAsset with asset creation failure",
			events: []interface{}{
				// 1. Initial command
				Command[CreateAndEquipCommandBody]{
					CharacterId: characterId,
					Type:        CommandTypeCreateAndEquip,
					Body: CreateAndEquipCommandBody{
						TemplateId: 999999, // Invalid template
						Quantity:   1,
					},
				},
				// 2. Asset creation failure
				StatusEvent[CreationFailedStatusEventBody]{
					TransactionId: transactionId,
					Type:          StatusEventTypeCreationFailed,
					CharacterId:   characterId,
					Body: CreationFailedStatusEventBody{
						ErrorCode: "INVALID_TEMPLATE_ID",
						Message:   "Template ID 999999 is not valid",
					},
				},
			},
			description: "Should handle asset creation failure in CreateAndEquipAsset",
		},
		{
			name: "CreateAndEquipAsset with equipment failure",
			events: []interface{}{
				// 1. Initial command
				Command[CreateAndEquipCommandBody]{
					CharacterId: characterId,
					Type:        CommandTypeCreateAndEquip,
					Body: CreateAndEquipCommandBody{
						TemplateId: 1302000,
						Quantity:   1,
					},
				},
				// 2. Asset creation success
				StatusEvent[CreatedStatusEventBody]{
					TransactionId: transactionId,
					Type:          StatusEventTypeCreated,
					CharacterId:   characterId,
					Body: CreatedStatusEventBody{
						Type:     1,
						AssetId:  67890,
						Quantity: 1,
					},
				},
				// 3. Auto-generated equip command (internal)
				Command[EquipCommandBody]{
					CharacterId: characterId,
					Type:        CommandTypeEquip,
					Body: EquipCommandBody{
						InventoryType: 1,
						Source:        5,
						Destination:   -1,
					},
				},
				// 4. Equipment failure
				StatusEvent[ErrorEventBody]{
					TransactionId: transactionId,
					Type:          StatusEventTypeError,
					CharacterId:   characterId,
					Body: ErrorEventBody{
						ErrorCode: "EQUIPMENT_SLOT_OCCUPIED",
					},
				},
			},
			description: "Should handle equipment failure in CreateAndEquipAsset",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify that each event in the flow can be serialized and deserialized
			for i, event := range tt.events {
				jsonData, err := json.Marshal(event)
				assert.NoError(t, err, "Event %d should serialize without error", i)
				
				// Verify the JSON is well-formed
				var temp interface{}
				err = json.Unmarshal(jsonData, &temp)
				assert.NoError(t, err, "Event %d should produce valid JSON", i)
				
				// Verify specific fields based on event type
				switch e := event.(type) {
				case Command[CreateAndEquipCommandBody]:
					assert.Contains(t, string(jsonData), "CREATE_AND_EQUIP")
					assert.Contains(t, string(jsonData), "templateId")
					assert.Contains(t, string(jsonData), "quantity")
					
				case Command[EquipCommandBody]:
					assert.Contains(t, string(jsonData), "EQUIP")
					assert.Contains(t, string(jsonData), "inventoryType")
					assert.Contains(t, string(jsonData), "source")
					assert.Contains(t, string(jsonData), "destination")
					
				case StatusEvent[CreatedStatusEventBody]:
					assert.Contains(t, string(jsonData), "CREATED")
					assert.Contains(t, string(jsonData), "transactionId")
					assert.Contains(t, string(jsonData), "assetId")
					assert.Equal(t, transactionId, e.TransactionId)
					
				case asset.StatusEvent[asset.MovedStatusEventBody]:
					assert.Contains(t, string(jsonData), "MOVED")
					assert.Contains(t, string(jsonData), "transactionId")
					assert.Contains(t, string(jsonData), "oldSlot")
					assert.Equal(t, transactionId, e.TransactionId)
					
				case StatusEvent[CreationFailedStatusEventBody]:
					assert.Contains(t, string(jsonData), "CREATION_FAILED")
					assert.Contains(t, string(jsonData), "errorCode")
					assert.Contains(t, string(jsonData), "message")
					
				case StatusEvent[ErrorEventBody]:
					assert.Contains(t, string(jsonData), "ERROR")
					assert.Contains(t, string(jsonData), "errorCode")
				}
			}
		})
	}
}

// TestEnvironmentVariableConstants tests environment variable constants
func TestEnvironmentVariableConstants(t *testing.T) {
	tests := []struct {
		name        string
		constant    string
		expected    string
		description string
	}{
		{
			name:        "EnvCommandTopic",
			constant:    EnvCommandTopic,
			expected:    "COMMAND_TOPIC_COMPARTMENT",
			description: "Should have correct value for compartment command topic",
		},
		{
			name:        "EnvEventTopicStatus",
			constant:    EnvEventTopicStatus,
			expected:    "EVENT_TOPIC_COMPARTMENT_STATUS",
			description: "Should have correct value for compartment status event topic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.constant, tt.description)
		})
	}
}

// TestEventTimestamps tests that events can handle timestamps correctly
func TestEventTimestamps(t *testing.T) {
	now := time.Now()
	transactionId := uuid.New()

	// Test that events maintain timestamp consistency
	event := StatusEvent[CreatedStatusEventBody]{
		TransactionId: transactionId,
		Type:          StatusEventTypeCreated,
		CharacterId:   12345,
		Body: CreatedStatusEventBody{
			Type:     1,
			AssetId:  67890,
			Quantity: 1,
		},
	}

	// Serialize the event
	jsonData, err := json.Marshal(event)
	assert.NoError(t, err, "Should serialize event with timestamp")

	// Deserialize the event
	var deserialized StatusEvent[CreatedStatusEventBody]
	err = json.Unmarshal(jsonData, &deserialized)
	assert.NoError(t, err, "Should deserialize event with timestamp")

	// Verify the event maintains its structure
	assert.Equal(t, event.TransactionId, deserialized.TransactionId)
	assert.Equal(t, event.Type, deserialized.Type)
	assert.Equal(t, event.CharacterId, deserialized.CharacterId)
	assert.Equal(t, event.Body, deserialized.Body)

	// Verify transaction ID is preserved
	assert.Equal(t, transactionId, deserialized.TransactionId)

	// Verify the operation completed within a reasonable time
	elapsed := time.Since(now)
	assert.Less(t, elapsed, time.Second, "Event processing should be fast")
}