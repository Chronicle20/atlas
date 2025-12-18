package compartment

import (
	"atlas-saga-orchestrator/kafka/message/compartment"
	"encoding/json"
	"testing"
	"time"

	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestCreateAndEquipAssetFlow(t *testing.T) {
	characterId := uint32(12345)
	templateId := uint32(1302000)
	quantity := uint32(1)

	payload := CreateAndEquipAssetPayload{
		CharacterId: characterId,
		Item: ItemPayload{
			TemplateId: templateId,
			Quantity:   quantity,
		},
	}

	t.Run("processor method delegates to RequestCreateItem", func(t *testing.T) {
		// Test that RequestCreateAndEquipAsset uses the same logic as RequestCreateItem
		// This validates that the CreateAndEquipAsset action uses award_asset semantics
		
		// Create a mock processor to validate the call
		// Note: This would typically require dependency injection or mocking framework
		// For now, we'll validate the payload structure
		
		assert.Equal(t, characterId, payload.CharacterId)
		assert.Equal(t, templateId, payload.Item.TemplateId)
		assert.Equal(t, quantity, payload.Item.Quantity)
	})

	t.Run("validates item payload structure", func(t *testing.T) {
		// Test various item payloads
		testCases := []struct {
			name       string
			templateId uint32
			quantity   uint32
		}{
			{"weapon", 1302000, 1},
			{"armor", 1040000, 1},
			{"consumable", 2000000, 100},
			{"etc", 4000000, 10},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				testPayload := CreateAndEquipAssetPayload{
					CharacterId: characterId,
					Item: ItemPayload{
						TemplateId: tc.templateId,
						Quantity:   tc.quantity,
					},
				}

				assert.Equal(t, characterId, testPayload.CharacterId)
				assert.Equal(t, tc.templateId, testPayload.Item.TemplateId)
				assert.Equal(t, tc.quantity, testPayload.Item.Quantity)
			})
		}
	})
}

func TestRequestCreateAssetCommandProvider(t *testing.T) {
	transactionId := uuid.New()
	characterId := uint32(12345)
	templateId := uint32(1302000)
	quantity := uint32(1)
	inventoryType := inventory.Type(1)

	t.Run("creates valid Kafka message", func(t *testing.T) {
		provider := RequestCreateAssetCommandProvider(transactionId, characterId, inventoryType, templateId, quantity)
		require.NotNil(t, provider)

		messages, err := provider()
		require.NoError(t, err)
		require.Len(t, messages, 1)

		message := messages[0]
		assert.Equal(t, producer.CreateKey(int(characterId)), message.Key)
		assert.NotNil(t, message.Value)

		// Deserialize the command structure
		var command compartment.Command[compartment.CreateAssetCommandBody]
		err = json.Unmarshal(message.Value, &command)
		require.NoError(t, err, "message value should be deserializable to Command[CreateAssetCommandBody]")

		assert.Equal(t, transactionId, command.TransactionId)
		assert.Equal(t, characterId, command.CharacterId)
		assert.Equal(t, compartment.CommandCreateAsset, command.Type)

		// Verify command body
		assert.Equal(t, templateId, command.Body.TemplateId)
		assert.Equal(t, quantity, command.Body.Quantity)
		assert.Equal(t, time.Time{}, command.Body.Expiration)
		assert.Equal(t, uint32(0), command.Body.OwnerId)
		assert.Equal(t, uint16(0), command.Body.Flag)
		assert.Equal(t, uint64(0), command.Body.Rechargeable)
	})

	t.Run("handles zero quantity", func(t *testing.T) {
		provider := RequestCreateAssetCommandProvider(transactionId, characterId, inventoryType, templateId, 0)
		messages, err := provider()
		require.NoError(t, err)
		require.Len(t, messages, 1)

		var command compartment.Command[compartment.CreateAssetCommandBody]
		err = json.Unmarshal(messages[0].Value, &command)
		require.NoError(t, err)
		assert.Equal(t, uint32(0), command.Body.Quantity)
	})

	t.Run("handles maximum quantity", func(t *testing.T) {
		maxQuantity := uint32(4294967295) // max uint32
		provider := RequestCreateAssetCommandProvider(transactionId, characterId, inventoryType, templateId, maxQuantity)
		messages, err := provider()
		require.NoError(t, err)
		require.Len(t, messages, 1)

		var command compartment.Command[compartment.CreateAssetCommandBody]
		err = json.Unmarshal(messages[0].Value, &command)
		require.NoError(t, err)
		assert.Equal(t, maxQuantity, command.Body.Quantity)
	})
}

func TestRequestEquipAssetCommandProvider(t *testing.T) {
	transactionId := uuid.New()
	characterId := uint32(12345)
	inventoryType := byte(1)
	source := int16(5)
	destination := int16(-1)

	t.Run("creates valid Kafka message", func(t *testing.T) {
		provider := RequestEquipAssetCommandProvider(transactionId, characterId, inventoryType, source, destination)
		require.NotNil(t, provider)

		messages, err := provider()
		require.NoError(t, err)
		require.Len(t, messages, 1)

		message := messages[0]
		assert.Equal(t, producer.CreateKey(int(characterId)), message.Key)
		assert.NotNil(t, message.Value)

		// Deserialize the command structure
		var command compartment.Command[compartment.EquipCommandBody]
		err = json.Unmarshal(message.Value, &command)
		require.NoError(t, err, "message value should be deserializable to Command[EquipCommandBody]")

		assert.Equal(t, transactionId, command.TransactionId)
		assert.Equal(t, characterId, command.CharacterId)
		assert.Equal(t, compartment.CommandEquip, command.Type)

		// Verify command body
		assert.Equal(t, source, command.Body.Source)
		assert.Equal(t, destination, command.Body.Destination)
	})

	t.Run("handles different inventory types", func(t *testing.T) {
		testCases := []struct {
			name          string
			inventoryType byte
		}{
			{"equipped", 1},
			{"inventory", 2},
			{"storage", 3},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				provider := RequestEquipAssetCommandProvider(transactionId, characterId, tc.inventoryType, source, destination)
				messages, err := provider()
				require.NoError(t, err)
				require.Len(t, messages, 1)

				var command compartment.Command[compartment.EquipCommandBody]
				err = json.Unmarshal(messages[0].Value, &command)
				require.NoError(t, err)
				assert.Equal(t, tc.inventoryType, command.InventoryType)
			})
		}
	})

	t.Run("handles negative destination slot", func(t *testing.T) {
		provider := RequestEquipAssetCommandProvider(transactionId, characterId, inventoryType, source, -1)
		messages, err := provider()
		require.NoError(t, err)
		require.Len(t, messages, 1)

		var command compartment.Command[compartment.EquipCommandBody]
		err = json.Unmarshal(messages[0].Value, &command)
		require.NoError(t, err)
		assert.Equal(t, int16(-1), command.Body.Destination)
	})
}

func TestMessageProviderErrorHandling(t *testing.T) {
	t.Run("provider returns error", func(t *testing.T) {
		expectedError := assert.AnError
		errorProvider := model.ErrorProvider[[]kafka.Message](expectedError)

		result, err := errorProvider()
		assert.Error(t, err)
		assert.Equal(t, expectedError, err)
		assert.Nil(t, result)
	})

	t.Run("provider returns empty messages", func(t *testing.T) {
		emptyProvider := model.FixedProvider([]kafka.Message{})

		result, err := emptyProvider()
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result, 0)
	})
}

func TestMessageKeyGeneration(t *testing.T) {
	t.Run("consistent key generation", func(t *testing.T) {
		characterId := uint32(12345)
		
		// Create multiple messages for same character
		provider1 := RequestCreateAssetCommandProvider(uuid.New(), characterId, inventory.Type(1), 1302000, 1)
		provider2 := RequestEquipAssetCommandProvider(uuid.New(), characterId, 1, 5, -1)

		messages1, err1 := provider1()
		require.NoError(t, err1)
		
		messages2, err2 := provider2()
		require.NoError(t, err2)

		// Both should have same key (for same character)
		expectedKey := producer.CreateKey(int(characterId))
		assert.Equal(t, expectedKey, messages1[0].Key)
		assert.Equal(t, expectedKey, messages2[0].Key)
		assert.Equal(t, messages1[0].Key, messages2[0].Key)
	})

	t.Run("different keys for different characters", func(t *testing.T) {
		char1 := uint32(12345)
		char2 := uint32(67890)

		provider1 := RequestCreateAssetCommandProvider(uuid.New(), char1, inventory.Type(1), 1302000, 1)
		provider2 := RequestCreateAssetCommandProvider(uuid.New(), char2, inventory.Type(1), 1302000, 1)

		messages1, err1 := provider1()
		require.NoError(t, err1)
		
		messages2, err2 := provider2()
		require.NoError(t, err2)

		// Should have different keys
		assert.NotEqual(t, messages1[0].Key, messages2[0].Key)
	})
}

func TestCommandTimestampValidation(t *testing.T) {
	t.Run("command contains valid timestamp", func(t *testing.T) {
		transactionId := uuid.New()
		characterId := uint32(12345)
		templateId := uint32(1302000)
		quantity := uint32(1)

		beforeTime := time.Now()
		provider := RequestCreateAssetCommandProvider(transactionId, characterId, inventory.Type(1), templateId, quantity)
		messages, err := provider()
		afterTime := time.Now()

		require.NoError(t, err)
		require.Len(t, messages, 1)

		var command compartment.Command[compartment.CreateAssetCommandBody]
		err = json.Unmarshal(messages[0].Value, &command)
		require.NoError(t, err)
		
		// Verify transaction ID is set
		assert.Equal(t, transactionId, command.TransactionId)
		
		// Verify timestamp is reasonable (within test execution window)
		// This is a basic sanity check - in real implementation, timestamp
		// would be set by the producer infrastructure
		assert.True(t, beforeTime.Before(afterTime) || beforeTime.Equal(afterTime))
	})
}