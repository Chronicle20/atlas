package character_test

import (
	"atlas-character/character"
	"atlas-character/kafka/message"
	character2 "atlas-character/kafka/message/character"
	"context"
	"encoding/json"
	"testing"

	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
)

// TestProducerFunctionsViaUpdate tests the new producer functions by verifying they are called
// through the Update processor method and produce the expected Kafka messages
func TestNameChangedEventViaUpdate(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := testDatabase(t)

	// Create a character first
	input := character.NewModelBuilder().
		SetAccountId(1000).
		SetWorldId(0).
		SetName("OriginalName").
		SetLevel(10).
		Build()
	processor := character.NewProcessor(testLogger(), tctx, db)
	created, err := processor.Create(message.NewBuffer())(uuid.New(), input)
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}

	// Update the name and capture the message buffer
	updateInput := character.RestModel{
		Name: "NewName",
	}

	transactionId := uuid.New()
	mb := message.NewBuffer()
	err = processor.Update(mb)(transactionId, created.Id(), updateInput)
	if err != nil {
		t.Fatalf("Failed to update character name: %v", err)
	}

	// Verify the message was added to the buffer
	messages := mb.GetAll()
	if len(messages) != 1 {
		t.Fatalf("Expected 1 topic in buffer, got %d", len(messages))
	}

	statusMessages, exists := messages[character2.EnvEventTopicCharacterStatus]
	if !exists {
		t.Fatal("Expected character status event topic in buffer")
	}

	if len(statusMessages) != 1 {
		t.Fatalf("Expected 1 message in character status topic, got %d", len(statusMessages))
	}

	// Verify the message content
	var event character2.StatusEvent[character2.StatusEventNameChangedBody]
	if err := json.Unmarshal(statusMessages[0].Value, &event); err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	if event.TransactionId != transactionId {
		t.Errorf("Expected TransactionId %s, got %s", transactionId, event.TransactionId)
	}
	if event.CharacterId != created.Id() {
		t.Errorf("Expected CharacterId %d, got %d", created.Id(), event.CharacterId)
	}
	if event.WorldId != world.Id(0) {
		t.Errorf("Expected WorldId 0, got %d", event.WorldId)
	}
	if event.Type != character2.StatusEventTypeNameChanged {
		t.Errorf("Expected Type %s, got %s", character2.StatusEventTypeNameChanged, event.Type)
	}
	if event.Body.OldName != "OriginalName" {
		t.Errorf("Expected OldName 'OriginalName', got '%s'", event.Body.OldName)
	}
	if event.Body.NewName != "NewName" {
		t.Errorf("Expected NewName 'NewName', got '%s'", event.Body.NewName)
	}
}

func TestHairChangedEventViaUpdate(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := testDatabase(t)

	// Create a character first
	input := character.NewModelBuilder().
		SetAccountId(1000).
		SetWorldId(0).
		SetName("HairTest").
		SetLevel(10).
		SetHair(30000).
		Build()
	processor := character.NewProcessor(testLogger(), tctx, db)
	created, err := processor.Create(message.NewBuffer())(uuid.New(), input)
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}

	// Update the hair and capture the message buffer
	updateInput := character.RestModel{
		Hair: 30100,
	}

	transactionId := uuid.New()
	mb := message.NewBuffer()
	err = processor.Update(mb)(transactionId, created.Id(), updateInput)
	if err != nil {
		t.Fatalf("Failed to update character hair: %v", err)
	}

	// Verify the message was added to the buffer
	messages := mb.GetAll()
	statusMessages, exists := messages[character2.EnvEventTopicCharacterStatus]
	if !exists {
		t.Fatal("Expected character status event topic in buffer")
	}

	if len(statusMessages) != 1 {
		t.Fatalf("Expected 1 message in character status topic, got %d", len(statusMessages))
	}

	// Verify the message content
	var event character2.StatusEvent[character2.StatusEventHairChangedBody]
	if err := json.Unmarshal(statusMessages[0].Value, &event); err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	if event.Type != character2.StatusEventTypeHairChanged {
		t.Errorf("Expected Type %s, got %s", character2.StatusEventTypeHairChanged, event.Type)
	}
	if event.Body.OldHair != 30000 {
		t.Errorf("Expected OldHair 30000, got %d", event.Body.OldHair)
	}
	if event.Body.NewHair != 30100 {
		t.Errorf("Expected NewHair 30100, got %d", event.Body.NewHair)
	}
}

func TestFaceChangedEventViaUpdate(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := testDatabase(t)

	// Create a character first
	input := character.NewModelBuilder().
		SetAccountId(1000).
		SetWorldId(0).
		SetName("FaceTest").
		SetLevel(10).
		SetFace(20000).
		Build()
	processor := character.NewProcessor(testLogger(), tctx, db)
	created, err := processor.Create(message.NewBuffer())(uuid.New(), input)
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}

	// Update the face and capture the message buffer
	updateInput := character.RestModel{
		Face: 20100,
	}

	transactionId := uuid.New()
	mb := message.NewBuffer()
	err = processor.Update(mb)(transactionId, created.Id(), updateInput)
	if err != nil {
		t.Fatalf("Failed to update character face: %v", err)
	}

	// Verify the message was added to the buffer
	messages := mb.GetAll()
	statusMessages, exists := messages[character2.EnvEventTopicCharacterStatus]
	if !exists {
		t.Fatal("Expected character status event topic in buffer")
	}

	if len(statusMessages) != 1 {
		t.Fatalf("Expected 1 message in character status topic, got %d", len(statusMessages))
	}

	// Verify the message content
	var event character2.StatusEvent[character2.StatusEventFaceChangedBody]
	if err := json.Unmarshal(statusMessages[0].Value, &event); err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	if event.Type != character2.StatusEventTypeFaceChanged {
		t.Errorf("Expected Type %s, got %s", character2.StatusEventTypeFaceChanged, event.Type)
	}
	if event.Body.OldFace != 20000 {
		t.Errorf("Expected OldFace 20000, got %d", event.Body.OldFace)
	}
	if event.Body.NewFace != 20100 {
		t.Errorf("Expected NewFace 20100, got %d", event.Body.NewFace)
	}
}

func TestGenderChangedEventViaUpdate(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := testDatabase(t)

	// Create a character first
	input := character.NewModelBuilder().
		SetAccountId(1000).
		SetWorldId(0).
		SetName("GenderTest").
		SetLevel(10).
		SetGender(0).
		Build()
	processor := character.NewProcessor(testLogger(), tctx, db)
	created, err := processor.Create(message.NewBuffer())(uuid.New(), input)
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}

	// Update the gender and capture the message buffer
	updateInput := character.RestModel{
		Gender: 1,
	}

	transactionId := uuid.New()
	mb := message.NewBuffer()
	err = processor.Update(mb)(transactionId, created.Id(), updateInput)
	if err != nil {
		t.Fatalf("Failed to update character gender: %v", err)
	}

	// Verify the message was added to the buffer
	messages := mb.GetAll()
	statusMessages, exists := messages[character2.EnvEventTopicCharacterStatus]
	if !exists {
		t.Fatal("Expected character status event topic in buffer")
	}

	if len(statusMessages) != 1 {
		t.Fatalf("Expected 1 message in character status topic, got %d", len(statusMessages))
	}

	// Verify the message content
	var event character2.StatusEvent[character2.StatusEventGenderChangedBody]
	if err := json.Unmarshal(statusMessages[0].Value, &event); err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	if event.Type != character2.StatusEventTypeGenderChanged {
		t.Errorf("Expected Type %s, got %s", character2.StatusEventTypeGenderChanged, event.Type)
	}
	if event.Body.OldGender != 0 {
		t.Errorf("Expected OldGender 0, got %d", event.Body.OldGender)
	}
	if event.Body.NewGender != 1 {
		t.Errorf("Expected NewGender 1, got %d", event.Body.NewGender)
	}
}

func TestSkinColorChangedEventViaUpdate(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := testDatabase(t)

	// Create a character first
	input := character.NewModelBuilder().
		SetAccountId(1000).
		SetWorldId(0).
		SetName("SkinTest").
		SetLevel(10).
		SetSkinColor(0).
		Build()
	processor := character.NewProcessor(testLogger(), tctx, db)
	created, err := processor.Create(message.NewBuffer())(uuid.New(), input)
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}

	// Update the skin color and capture the message buffer
	updateInput := character.RestModel{
		SkinColor: 5,
	}

	transactionId := uuid.New()
	mb := message.NewBuffer()
	err = processor.Update(mb)(transactionId, created.Id(), updateInput)
	if err != nil {
		t.Fatalf("Failed to update character skin color: %v", err)
	}

	// Verify the message was added to the buffer
	messages := mb.GetAll()
	statusMessages, exists := messages[character2.EnvEventTopicCharacterStatus]
	if !exists {
		t.Fatal("Expected character status event topic in buffer")
	}

	if len(statusMessages) != 1 {
		t.Fatalf("Expected 1 message in character status topic, got %d", len(statusMessages))
	}

	// Verify the message content
	var event character2.StatusEvent[character2.StatusEventSkinColorChangedBody]
	if err := json.Unmarshal(statusMessages[0].Value, &event); err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	if event.Type != character2.StatusEventTypeSkinColorChanged {
		t.Errorf("Expected Type %s, got %s", character2.StatusEventTypeSkinColorChanged, event.Type)
	}
	if event.Body.OldSkinColor != 0 {
		t.Errorf("Expected OldSkinColor 0, got %d", event.Body.OldSkinColor)
	}
	if event.Body.NewSkinColor != 5 {
		t.Errorf("Expected NewSkinColor 5, got %d", event.Body.NewSkinColor)
	}
}

func TestGmChangedEventViaUpdate(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := testDatabase(t)

	// Create a character first
	input := character.NewModelBuilder().
		SetAccountId(1000).
		SetWorldId(0).
		SetName("GmTest").
		SetLevel(10).
		SetGm(0).
		Build()
	processor := character.NewProcessor(testLogger(), tctx, db)
	created, err := processor.Create(message.NewBuffer())(uuid.New(), input)
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}

	// Update the GM status and capture the message buffer
	updateInput := character.RestModel{
		Gm: 1,
	}

	transactionId := uuid.New()
	mb := message.NewBuffer()
	err = processor.Update(mb)(transactionId, created.Id(), updateInput)
	if err != nil {
		t.Fatalf("Failed to update character GM status: %v", err)
	}

	// Verify the message was added to the buffer
	messages := mb.GetAll()
	statusMessages, exists := messages[character2.EnvEventTopicCharacterStatus]
	if !exists {
		t.Fatal("Expected character status event topic in buffer")
	}

	if len(statusMessages) != 1 {
		t.Fatalf("Expected 1 message in character status topic, got %d", len(statusMessages))
	}

	// Verify the message content
	var event character2.StatusEvent[character2.StatusEventGmChangedBody]
	if err := json.Unmarshal(statusMessages[0].Value, &event); err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	if event.Type != character2.StatusEventTypeGmChanged {
		t.Errorf("Expected Type %s, got %s", character2.StatusEventTypeGmChanged, event.Type)
	}
	if event.Body.OldGm != false {
		t.Errorf("Expected OldGm false, got %v", event.Body.OldGm)
	}
	if event.Body.NewGm != true {
		t.Errorf("Expected NewGm true, got %v", event.Body.NewGm)
	}
}

func TestMultipleFieldChangesProduceMultipleEvents(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := testDatabase(t)

	// Create a character first
	input := character.NewModelBuilder().
		SetAccountId(1000).
		SetWorldId(0).
		SetName("MultiTest").
		SetLevel(10).
		SetHair(30000).
		SetFace(20000).
		SetGender(0).
		SetSkinColor(0).
		SetGm(0).
		Build()
	processor := character.NewProcessor(testLogger(), tctx, db)
	created, err := processor.Create(message.NewBuffer())(uuid.New(), input)
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}

	// Update multiple fields and capture the message buffer
	updateInput := character.RestModel{
		Name:      "NewMultiTest",
		Hair:      30100,
		Face:      20100,
		Gender:    1,
		SkinColor: 5,
		Gm:        1,
	}

	transactionId := uuid.New()
	mb := message.NewBuffer()
	err = processor.Update(mb)(transactionId, created.Id(), updateInput)
	if err != nil {
		t.Fatalf("Failed to update character: %v", err)
	}

	// Verify multiple messages were added to the buffer
	messages := mb.GetAll()
	statusMessages, exists := messages[character2.EnvEventTopicCharacterStatus]
	if !exists {
		t.Fatal("Expected character status event topic in buffer")
	}

	// Should have 6 messages - one for each field change
	if len(statusMessages) != 6 {
		t.Fatalf("Expected 6 messages in character status topic, got %d", len(statusMessages))
	}

	// Verify we have all the expected event types
	eventTypes := make(map[string]bool)
	for _, msg := range statusMessages {
		var baseEvent map[string]interface{}
		if err := json.Unmarshal(msg.Value, &baseEvent); err != nil {
			t.Fatalf("Failed to unmarshal message: %v", err)
		}
		eventType := baseEvent["type"].(string)
		eventTypes[eventType] = true
	}

	expectedTypes := []string{
		character2.StatusEventTypeNameChanged,
		character2.StatusEventTypeHairChanged,
		character2.StatusEventTypeFaceChanged,
		character2.StatusEventTypeGenderChanged,
		character2.StatusEventTypeSkinColorChanged,
		character2.StatusEventTypeGmChanged,
	}

	for _, expectedType := range expectedTypes {
		if !eventTypes[expectedType] {
			t.Errorf("Missing expected event type: %s", expectedType)
		}
	}
}

func TestMapChangedEventViaUpdate(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := testDatabase(t)

	// Create a character first
	input := character.NewModelBuilder().
		SetAccountId(1000).
		SetWorldId(0).
		SetName("MapTest").
		SetLevel(10).
		SetMapId(100000000).
		Build()
	processor := character.NewProcessor(testLogger(), tctx, db)
	created, err := processor.Create(message.NewBuffer())(uuid.New(), input)
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}

	// Update the map and capture the message buffer
	updateInput := character.RestModel{
		MapId: _map.Id(110000000),
	}

	transactionId := uuid.New()
	mb := message.NewBuffer()
	err = processor.Update(mb)(transactionId, created.Id(), updateInput)
	if err != nil {
		t.Fatalf("Failed to update character map: %v", err)
	}

	// Verify the message was added to the buffer
	messages := mb.GetAll()
	statusMessages, exists := messages[character2.EnvEventTopicCharacterStatus]
	if !exists {
		t.Fatal("Expected character status event topic in buffer")
	}

	if len(statusMessages) != 1 {
		t.Fatalf("Expected 1 message in character status topic, got %d", len(statusMessages))
	}

	// Verify the message content - should be MAP_CHANGED event
	var event character2.StatusEvent[character2.StatusEventMapChangedBody]
	if err := json.Unmarshal(statusMessages[0].Value, &event); err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	if event.Type != character2.StatusEventTypeMapChanged {
		t.Errorf("Expected Type %s, got %s", character2.StatusEventTypeMapChanged, event.Type)
	}
	if event.Body.OldMapId != _map.Id(100000000) {
		t.Errorf("Expected OldMapId 100000000, got %d", event.Body.OldMapId)
	}
	if event.Body.TargetMapId != _map.Id(110000000) {
		t.Errorf("Expected TargetMapId 110000000, got %d", event.Body.TargetMapId)
	}
}

func TestProducerFunctionsHandleEmptyStringValues(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := testDatabase(t)

	// Create a character first
	input := character.NewModelBuilder().
		SetAccountId(1000).
		SetWorldId(0).
		SetName("OriginalName").
		SetLevel(10).
		Build()
	processor := character.NewProcessor(testLogger(), tctx, db)
	created, err := processor.Create(message.NewBuffer())(uuid.New(), input)
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}

	// Update with empty name (should not trigger event)
	updateInput := character.RestModel{
		Name: "",
	}

	transactionId := uuid.New()
	mb := message.NewBuffer()
	err = processor.Update(mb)(transactionId, created.Id(), updateInput)
	if err != nil {
		t.Fatalf("Failed to update character: %v", err)
	}

	// Verify no messages were added to the buffer
	messages := mb.GetAll()
	if len(messages) != 0 {
		t.Fatalf("Expected 0 topics in buffer for empty name, got %d", len(messages))
	}
}
