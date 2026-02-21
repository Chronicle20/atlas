package saga

import (
	"testing"

	"github.com/google/uuid"
)

// TestCreateCommandProvider tests the saga command provider
func TestCreateCommandProvider(t *testing.T) {
	transactionId := uuid.New()

	saga := Saga{
		TransactionId: transactionId,
		SagaType:      InventoryTransaction,
		InitiatedBy:   "atlas-messages",
		Steps: []Step{
			{
				StepId:  "test_step",
				Status:  Pending,
				Action:  AwardExperience,
				Payload: AwardExperiencePayload{CharacterId: 12345},
			},
		},
	}

	provider := CreateCommandProvider(saga)

	if provider == nil {
		t.Fatal("CreateCommandProvider should not return nil")
	}

	// Execute the provider
	messages, err := provider()
	if err != nil {
		t.Fatalf("Provider returned error: %v", err)
	}

	// Should return exactly one message
	if len(messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(messages))
	}

	// Key should be the transaction ID
	expectedKey := transactionId.String()
	if string(messages[0].Key) != expectedKey {
		t.Errorf("Expected key '%s', got '%s'", expectedKey, string(messages[0].Key))
	}

	// Value should be set (the serialized saga)
	if len(messages[0].Value) == 0 {
		t.Error("Expected message value to be set")
	}
}

// TestCreateCommandProvider_DifferentTransactionIds tests key uniqueness
func TestCreateCommandProvider_DifferentTransactionIds(t *testing.T) {
	id1 := uuid.New()
	id2 := uuid.New()

	saga1 := Saga{TransactionId: id1, Steps: []Step{}}
	saga2 := Saga{TransactionId: id2, Steps: []Step{}}

	provider1 := CreateCommandProvider(saga1)
	provider2 := CreateCommandProvider(saga2)

	msgs1, _ := provider1()
	msgs2, _ := provider2()

	if string(msgs1[0].Key) == string(msgs2[0].Key) {
		t.Error("Expected different keys for different transaction IDs")
	}
}

// TestCreateCommandProvider_SameTransactionId tests key consistency
func TestCreateCommandProvider_SameTransactionId(t *testing.T) {
	id := uuid.New()

	saga1 := Saga{TransactionId: id, Steps: []Step{}}
	saga2 := Saga{TransactionId: id, Steps: []Step{}}

	provider1 := CreateCommandProvider(saga1)
	provider2 := CreateCommandProvider(saga2)

	msgs1, _ := provider1()
	msgs2, _ := provider2()

	if string(msgs1[0].Key) != string(msgs2[0].Key) {
		t.Error("Expected same key for same transaction ID")
	}
}

// TestCreateCommandProvider_WithEmptySteps tests saga with no steps
func TestCreateCommandProvider_WithEmptySteps(t *testing.T) {
	saga := Saga{
		TransactionId: uuid.New(),
		SagaType:      QuestReward,
		InitiatedBy:   "test",
		Steps:         []Step{},
	}

	provider := CreateCommandProvider(saga)
	messages, err := provider()

	if err != nil {
		t.Fatalf("Provider returned error: %v", err)
	}

	if len(messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(messages))
	}
}

// TestCreateCommandProvider_WithMultipleSteps tests saga with multiple steps
func TestCreateCommandProvider_WithMultipleSteps(t *testing.T) {
	saga, buildErr := NewBuilder().
		SetTransactionId(uuid.New()).
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("step1", Pending, AwardExperience, AwardExperiencePayload{}).
		AddStep("step2", Pending, AwardMesos, AwardMesosPayload{}).
		AddStep("step3", Pending, AwardAsset, AwardItemActionPayload{}).
		Build()

	if buildErr != nil {
		t.Fatalf("unexpected build error: %v", buildErr)
	}

	provider := CreateCommandProvider(saga)
	messages, err := provider()

	if err != nil {
		t.Fatalf("Provider returned error: %v", err)
	}

	if len(messages) != 1 {
		t.Errorf("Expected 1 message even with multiple steps, got %d", len(messages))
	}

	// The entire saga (with all steps) should be in a single message
	if len(messages[0].Value) == 0 {
		t.Error("Expected message value to contain serialized saga")
	}
}

// TestCreateCommandProvider_KeyFormat tests the key format
func TestCreateCommandProvider_KeyFormat(t *testing.T) {
	transactionId := uuid.New()
	saga := Saga{TransactionId: transactionId}

	provider := CreateCommandProvider(saga)
	messages, _ := provider()

	key := string(messages[0].Key)

	// Key should be a valid UUID string
	parsedId, err := uuid.Parse(key)
	if err != nil {
		t.Errorf("Key should be a valid UUID string, got '%s': %v", key, err)
	}

	if parsedId != transactionId {
		t.Errorf("Key UUID should match transaction ID")
	}
}
