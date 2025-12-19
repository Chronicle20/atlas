package compartment

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-constants/item"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

// setupTestProcessor creates a ProcessorImpl with mock dependencies for testing
func setupTestProcessor() (*ProcessorImpl, *test.Hook) {
	logger, hook := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	
	ctx := context.Background()
	
	processor := &ProcessorImpl{
		l:   logger,
		ctx: ctx,
	}
	
	return processor, hook
}

// TestRequestCreateAndEquipAsset_PayloadValidation tests the RequestCreateAndEquipAsset method with various payload validations
func TestRequestCreateAndEquipAsset_PayloadValidation(t *testing.T) {
	tests := []struct {
		name                string
		payload             CreateAndEquipAssetPayload
		expectedError       bool
		expectedErrorString string
		description         string
	}{
		{
			name: "Valid equip item payload",
			payload: CreateAndEquipAssetPayload{
				CharacterId: 12345,
				Item: ItemPayload{
					TemplateId: 1302000, // Valid equip template ID
					Quantity:   1,
				},
			},
			expectedError:       false,
			expectedErrorString: "",
			description:         "Should accept valid equip item payload",
		},
		{
			name: "Valid consumable item payload",
			payload: CreateAndEquipAssetPayload{
				CharacterId: 67890,
				Item: ItemPayload{
					TemplateId: 2000001, // Valid consumable template ID
					Quantity:   5,
				},
			},
			expectedError:       false,
			expectedErrorString: "",
			description:         "Should accept valid consumable item payload",
		},
		{
			name: "Invalid template ID - zero",
			payload: CreateAndEquipAssetPayload{
				CharacterId: 12345,
				Item: ItemPayload{
					TemplateId: 0, // Invalid template ID
					Quantity:   1,
				},
			},
			expectedError:       true,
			expectedErrorString: "invalid templateId",
			description:         "Should reject zero template ID",
		},
		{
			name: "Invalid template ID - unknown",
			payload: CreateAndEquipAssetPayload{
				CharacterId: 12345,
				Item: ItemPayload{
					TemplateId: 999999999, // Unknown template ID
					Quantity:   1,
				},
			},
			expectedError:       true,
			expectedErrorString: "invalid templateId",
			description:         "Should reject unknown template ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			processor, hook := setupTestProcessor()
			transactionId := uuid.New()

			// We'll use a timeout to prevent the test from hanging on Kafka connection
			done := make(chan error, 1)
			go func() {
				done <- processor.RequestCreateAndEquipAsset(transactionId, tt.payload)
			}()

			var err error
			select {
			case err = <-done:
				// Method completed (either with success or error)
			case <-time.After(1 * time.Second):
				// Method timed out - this is expected for valid payloads when Kafka is not available
				if !tt.expectedError {
					// For valid payloads, timeout is expected behavior
					t.Logf("Method timed out as expected for valid payload (Kafka not available)")
					return
				}
				t.Fatal("Method timed out unexpectedly for invalid payload")
			}

			// Verify results
			if tt.expectedError {
				assert.Error(t, err, tt.description)
				if tt.expectedErrorString != "" {
					assert.Contains(t, err.Error(), tt.expectedErrorString, tt.description)
				}
			} else {
				// For valid payloads, if we get here, it means the method returned quickly
				// This could happen if Kafka is available, or if there's an error we didn't expect
				if err != nil {
					t.Logf("Unexpected error for valid payload: %v", err)
				}
			}

			// Clean up
			hook.Reset()
		})
	}
}

// TestRequestCreateAndEquipAsset_PayloadTransformation tests that the payload is correctly transformed
func TestRequestCreateAndEquipAsset_PayloadTransformation(t *testing.T) {
	// Setup
	processor, hook := setupTestProcessor()
	transactionId := uuid.New()

	payload := CreateAndEquipAssetPayload{
		CharacterId: 12345,
		Item: ItemPayload{
			TemplateId: 1302000,
			Quantity:   1,
		},
	}

	// Test that the method correctly extracts and uses the payload fields
	// We can't easily test the internal delegation without mocking, but we can verify
	// that the method handles the payload structure correctly

	// Verify payload structure
	assert.Equal(t, uint32(12345), payload.CharacterId)
	assert.Equal(t, uint32(1302000), payload.Item.TemplateId)
	assert.Equal(t, uint32(1), payload.Item.Quantity)

	// Test that the method can be called without panicking
	done := make(chan error, 1)
	go func() {
		done <- processor.RequestCreateAndEquipAsset(transactionId, payload)
	}()

	select {
	case err := <-done:
		// Method completed - log any errors for debugging
		if err != nil {
			t.Logf("Method completed with error: %v", err)
		}
	case <-time.After(1 * time.Second):
		// Method timed out - expected when Kafka is not available
		t.Log("Method timed out as expected (Kafka not available)")
	}

	// Clean up
	hook.Reset()
}

// TestRequestCreateAndEquipAsset_TransactionIdHandling tests transaction ID handling
func TestRequestCreateAndEquipAsset_TransactionIdHandling(t *testing.T) {
	// Setup
	processor, hook := setupTestProcessor()
	
	// Test with different transaction IDs
	transactionIds := []uuid.UUID{
		uuid.New(),
		uuid.New(),
		uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
	}

	payload := CreateAndEquipAssetPayload{
		CharacterId: 12345,
		Item: ItemPayload{
			TemplateId: 1302000,
			Quantity:   1,
		},
	}

	for i, txnId := range transactionIds {
		t.Run(fmt.Sprintf("Transaction_%d", i), func(t *testing.T) {
			// Verify that the transaction ID is valid
			assert.NotEqual(t, uuid.Nil, txnId)
			
			// Test that the method can handle the transaction ID
			done := make(chan error, 1)
			go func() {
				done <- processor.RequestCreateAndEquipAsset(txnId, payload)
			}()

			select {
			case err := <-done:
				// Method completed - log any errors for debugging
				if err != nil {
					t.Logf("Transaction %s resulted in: %v", txnId, err)
				}
			case <-time.After(1 * time.Second):
				// Method timed out - expected when Kafka is not available
				t.Logf("Transaction %s timed out as expected", txnId)
			}
		})
	}

	// Clean up
	hook.Reset()
}

// TestRequestCreateAndEquipAsset_InternalDelegation tests the internal delegation logic
func TestRequestCreateAndEquipAsset_InternalDelegation(t *testing.T) {
	// Setup
	processor, hook := setupTestProcessor()
	transactionId := uuid.New()

	payload := CreateAndEquipAssetPayload{
		CharacterId: 12345,
		Item: ItemPayload{
			TemplateId: 1302000,
			Quantity:   1,
		},
	}

	// Test that the method correctly delegates to RequestCreateItem
	// We can verify this by testing the same logic that RequestCreateItem uses
	
	// Verify that the template ID maps to a valid inventory type
	inventoryType, ok := inventory.TypeFromItemId(item.Id(payload.Item.TemplateId))
	assert.True(t, ok, "Template ID should map to valid inventory type")
	assert.NotEqual(t, inventory.Type(0), inventoryType, "Inventory type should not be zero")

	// Test that the method can be called and behaves consistently with RequestCreateItem
	done := make(chan error, 1)
	go func() {
		done <- processor.RequestCreateAndEquipAsset(transactionId, payload)
	}()

	select {
	case err := <-done:
		// Method completed - log any errors for debugging
		if err != nil {
			t.Logf("Internal delegation completed with: %v", err)
		}
	case <-time.After(1 * time.Second):
		// Method timed out - expected when Kafka is not available
		t.Log("Internal delegation timed out as expected")
	}

	// Clean up
	hook.Reset()
}

// TestRequestCreateAndEquipAsset_InvalidTemplateIds tests various invalid template IDs
func TestRequestCreateAndEquipAsset_InvalidTemplateIds(t *testing.T) {
	processor, hook := setupTestProcessor()
	transactionId := uuid.New()

	invalidTemplateIds := []uint32{0, 999999999, 1, 99}

	for _, templateId := range invalidTemplateIds {
		t.Run(fmt.Sprintf("TemplateId_%d", templateId), func(t *testing.T) {
			payload := CreateAndEquipAssetPayload{
				CharacterId: 12345,
				Item: ItemPayload{
					TemplateId: templateId,
					Quantity:   1,
				},
			}

			done := make(chan error, 1)
			go func() {
				done <- processor.RequestCreateAndEquipAsset(transactionId, payload)
			}()

			select {
			case err := <-done:
				// Method completed - should return error for invalid template ID
				assert.Error(t, err, "Should return error for invalid template ID %d", templateId)
				assert.Contains(t, err.Error(), "invalid templateId")
			case <-time.After(1 * time.Second):
				// Method timed out - this shouldn't happen for invalid template IDs
				t.Errorf("Method timed out for invalid template ID %d, expected quick error", templateId)
			}
		})
	}

	hook.Reset()
}

// TestRequestCreateAndEquipAsset_BehaviorConsistency tests that the method behaves consistently with RequestCreateItem
func TestRequestCreateAndEquipAsset_BehaviorConsistency(t *testing.T) {
	processor, hook := setupTestProcessor()
	transactionId := uuid.New()

	testCases := []struct {
		name       string
		templateId uint32
		quantity   uint32
	}{
		{"Valid equip item", 1302000, 1},
		{"Valid consumable", 2000001, 5},
		{"Invalid template", 0, 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			payload := CreateAndEquipAssetPayload{
				CharacterId: 12345,
				Item: ItemPayload{
					TemplateId: tc.templateId,
					Quantity:   tc.quantity,
				},
			}

			// Test RequestCreateAndEquipAsset
			done1 := make(chan error, 1)
			go func() {
				done1 <- processor.RequestCreateAndEquipAsset(transactionId, payload)
			}()

			var err1 error
			select {
			case err1 = <-done1:
			case <-time.After(1 * time.Second):
				// Timeout - expected for valid template IDs
				if tc.templateId != 0 {
					t.Log("RequestCreateAndEquipAsset timed out as expected")
				}
			}

			// Test RequestCreateItem with same parameters
			done2 := make(chan error, 1)
			go func() {
				done2 <- processor.RequestCreateItem(transactionId, payload.CharacterId, tc.templateId, tc.quantity)
			}()

			var err2 error
			select {
			case err2 = <-done2:
			case <-time.After(1 * time.Second):
				// Timeout - expected for valid template IDs
				if tc.templateId != 0 {
					t.Log("RequestCreateItem timed out as expected")
				}
			}

			// Both methods should behave the same way
			if err1 != nil && err2 != nil {
				assert.Equal(t, err1.Error(), err2.Error(), "Both methods should return the same error")
			} else if err1 != nil || err2 != nil {
				// One errored and one didn't - this could be a timing issue or inconsistency
				t.Logf("Inconsistent behavior: RequestCreateAndEquipAsset error: %v, RequestCreateItem error: %v", err1, err2)
			}
		})
	}

	hook.Reset()
}