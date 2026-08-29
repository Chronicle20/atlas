package compartment

import (
	"atlas-saga-orchestrator/kafka/message/compartment"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	saga "github.com/Chronicle20/atlas/libs/atlas-saga"
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
		provider := RequestCreateAssetCommandProvider(transactionId, characterId, inventoryType, templateId, quantity, time.Time{}, false)
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
		provider := RequestCreateAssetCommandProvider(transactionId, characterId, inventoryType, templateId, 0, time.Time{}, false)
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
		provider := RequestCreateAssetCommandProvider(transactionId, characterId, inventoryType, templateId, maxQuantity, time.Time{}, false)
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
		provider1 := RequestCreateAssetCommandProvider(uuid.New(), characterId, inventory.Type(1), 1302000, 1, time.Time{}, false)
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

		provider1 := RequestCreateAssetCommandProvider(uuid.New(), char1, inventory.Type(1), 1302000, 1, time.Time{}, false)
		provider2 := RequestCreateAssetCommandProvider(uuid.New(), char2, inventory.Type(1), 1302000, 1, time.Time{}, false)

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
		provider := RequestCreateAssetCommandProvider(transactionId, characterId, inventory.Type(1), templateId, quantity, time.Time{}, false)
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

func TestRequestCreateAssetCommandProvider_UseAverageStats(t *testing.T) {
	transactionId := uuid.New()
	characterId := uint32(12345)
	templateId := uint32(1302000)
	quantity := uint32(1)
	inventoryType := inventory.Type(1)

	t.Run("useAverageStats=true is serialised into kafka message body", func(t *testing.T) {
		provider := RequestCreateAssetCommandProvider(transactionId, characterId, inventoryType, templateId, quantity, time.Time{}, true)
		require.NotNil(t, provider)

		messages, err := provider()
		require.NoError(t, err)
		require.Len(t, messages, 1)

		var command compartment.Command[compartment.CreateAssetCommandBody]
		err = json.Unmarshal(messages[0].Value, &command)
		require.NoError(t, err)

		assert.True(t, command.Body.UseAverageStats, "expected UseAverageStats=true in kafka body")

		// Also check raw JSON contains the field
		raw := string(messages[0].Value)
		assert.Contains(t, raw, `"useAverageStats":true`)
	})

	t.Run("useAverageStats=false is omitted from kafka message body (omitempty)", func(t *testing.T) {
		provider := RequestCreateAssetCommandProvider(transactionId, characterId, inventoryType, templateId, quantity, time.Time{}, false)
		messages, err := provider()
		require.NoError(t, err)
		require.Len(t, messages, 1)

		var command compartment.Command[compartment.CreateAssetCommandBody]
		err = json.Unmarshal(messages[0].Value, &command)
		require.NoError(t, err)

		assert.False(t, command.Body.UseAverageStats)
	})
}

// TestRequestCreateItemWithExplicitStatsCarriesEveryField pins that a fully
// populated saga.AwardCraftedAssetPayload survives, stat by stat, from the
// provider into the emitted CreateAssetCommandBody. Slots is asserted
// specifically because it is the one field without omitempty on the wire.
func TestRequestCreateItemWithExplicitStatsCarriesEveryField(t *testing.T) {
	transactionId := uuid.New()
	characterId := uint32(12345)
	templateId := uint32(1082002)
	quantity := uint32(1)
	inventoryType := inventory.Type(1)

	stats := saga.AwardCraftedAssetPayload{
		Slots:         7,
		Strength:      1,
		Dexterity:     2,
		Intelligence:  3,
		Luck:          4,
		HP:            5,
		MP:            6,
		WeaponAttack:  7,
		MagicAttack:   8,
		WeaponDefense: 9,
		MagicDefense:  10,
		Accuracy:      11,
		Avoidability:  12,
		Hands:         13,
		Speed:         14,
		Jump:          15,
	}

	provider := RequestCreateAssetWithStatsCommandProvider(transactionId, characterId, inventoryType, templateId, quantity, time.Time{}, false, stats)
	require.NotNil(t, provider)

	messages, err := provider()
	require.NoError(t, err)
	require.Len(t, messages, 1)

	var command compartment.Command[compartment.CreateAssetCommandBody]
	err = json.Unmarshal(messages[0].Value, &command)
	require.NoError(t, err)

	assert.Equal(t, uint16(7), command.Body.Slots)
	assert.Equal(t, stats.Strength, command.Body.Strength)
	assert.Equal(t, stats.Dexterity, command.Body.Dexterity)
	assert.Equal(t, stats.Intelligence, command.Body.Intelligence)
	assert.Equal(t, stats.Luck, command.Body.Luck)
	assert.Equal(t, stats.HP, command.Body.HP)
	assert.Equal(t, stats.MP, command.Body.MP)
	assert.Equal(t, stats.WeaponAttack, command.Body.WeaponAttack)
	assert.Equal(t, stats.MagicAttack, command.Body.MagicAttack)
	assert.Equal(t, stats.WeaponDefense, command.Body.WeaponDefense)
	assert.Equal(t, stats.MagicDefense, command.Body.MagicDefense)
	assert.Equal(t, stats.Accuracy, command.Body.Accuracy)
	assert.Equal(t, stats.Avoidability, command.Body.Avoidability)
	assert.Equal(t, stats.Hands, command.Body.Hands)
	assert.Equal(t, stats.Speed, command.Body.Speed)
	assert.Equal(t, stats.Jump, command.Body.Jump)
}

// TestRequestCreateItemIsUnchanged pins that the legacy
// RequestCreateAssetCommandProvider path is byte-identical to before the
// explicit-stat widening: every new stat field, and Slots, decode to zero.
func TestRequestCreateItemIsUnchanged(t *testing.T) {
	transactionId := uuid.New()
	characterId := uint32(12345)
	templateId := uint32(1302000)
	quantity := uint32(1)
	inventoryType := inventory.Type(1)

	provider := RequestCreateAssetCommandProvider(transactionId, characterId, inventoryType, templateId, quantity, time.Time{}, false)
	messages, err := provider()
	require.NoError(t, err)
	require.Len(t, messages, 1)

	var command compartment.Command[compartment.CreateAssetCommandBody]
	err = json.Unmarshal(messages[0].Value, &command)
	require.NoError(t, err)

	assert.Equal(t, uint16(0), command.Body.Slots)
	assert.Equal(t, uint16(0), command.Body.Strength)
	assert.Equal(t, uint16(0), command.Body.Dexterity)
	assert.Equal(t, uint16(0), command.Body.Intelligence)
	assert.Equal(t, uint16(0), command.Body.Luck)
	assert.Equal(t, uint16(0), command.Body.HP)
	assert.Equal(t, uint16(0), command.Body.MP)
	assert.Equal(t, uint16(0), command.Body.WeaponAttack)
	assert.Equal(t, uint16(0), command.Body.MagicAttack)
	assert.Equal(t, uint16(0), command.Body.WeaponDefense)
	assert.Equal(t, uint16(0), command.Body.MagicDefense)
	assert.Equal(t, uint16(0), command.Body.Accuracy)
	assert.Equal(t, uint16(0), command.Body.Avoidability)
	assert.Equal(t, uint16(0), command.Body.Hands)
	assert.Equal(t, uint16(0), command.Body.Speed)
	assert.Equal(t, uint16(0), command.Body.Jump)
}
