package conversation

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRestModelTransformation_StringItemId validates that RestConditionModel correctly handles string ItemId
func TestRestModelTransformation_StringItemId(t *testing.T) {
	tests := []struct {
		name        string
		itemId      string
		description string
	}{
		{
			name:        "Numeric string ItemId",
			itemId:      "4001126",
			description: "Traditional numeric item ID as string",
		},
		{
			name:        "Empty string ItemId",
			itemId:      "",
			description: "Empty ItemId (omitempty behavior)",
		},
		{
			name:        "Zero string ItemId",
			itemId:      "0",
			description: "Zero as string ItemId",
		},
		{
			name:        "Non-numeric string ItemId",
			itemId:      "SPECIAL_KEY_ITEM",
			description: "String-based item identifier",
		},
		{
			name:        "ItemId with special characters",
			itemId:      "item-123_special",
			description: "ItemId with hyphens and underscores",
		},
		{
			name:        "Unicode ItemId",
			itemId:      "アイテム_123",
			description: "ItemId with Unicode characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a REST condition model with string ItemId
			restCondition := RestConditionModel{
				Type:     "item",
				Operator: ">=",
				Value:    "1",
				ItemId:   tt.itemId,
			}

			// Test JSON marshaling
			jsonData, err := json.Marshal(restCondition)
			require.NoError(t, err, "Failed to marshal RestConditionModel")

			// Test JSON unmarshaling
			var unmarshaledCondition RestConditionModel
			err = json.Unmarshal(jsonData, &unmarshaledCondition)
			require.NoError(t, err, "Failed to unmarshal RestConditionModel")

			// Verify ItemId is preserved correctly
			assert.Equal(t, tt.itemId, unmarshaledCondition.ItemId, "ItemId mismatch after JSON round-trip")

			// If ItemId is empty, verify omitempty behavior
			if tt.itemId == "" {
				assert.NotContains(t, string(jsonData), `"itemId"`, "Empty ItemId should be omitted from JSON")
			} else {
				assert.Contains(t, string(jsonData), `"itemId"`, "Non-empty ItemId should be included in JSON")
			}
		})
	}
}

// TestRestOutcomeModel_WithoutSuccessFailureStates validates RestOutcomeModel without success/failure states
func TestRestOutcomeModel_WithoutSuccessFailureStates(t *testing.T) {
	// Create conditions with string ItemId
	conditions := []RestConditionModel{
		{
			Type:     "level",
			Operator: ">=",
			Value:    "30",
			ItemId:   "",
		},
		{
			Type:     "item",
			Operator: ">=",
			Value:    "5",
			ItemId:   "QUEST_ITEM_KEY",
		},
	}

	// Create outcome with only NextState (no success/failure states)
	restOutcome := RestOutcomeModel{
		Conditions: conditions,
		NextState:  "quest_complete",
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(restOutcome)
	require.NoError(t, err, "Failed to marshal RestOutcomeModel")

	// Verify JSON structure
	var jsonMap map[string]interface{}
	err = json.Unmarshal(jsonData, &jsonMap)
	require.NoError(t, err, "Failed to unmarshal to map")

	// Ensure no success/failure state fields exist
	assert.NotContains(t, jsonMap, "successState", "RestOutcomeModel should not have successState field")
	assert.NotContains(t, jsonMap, "failureState", "RestOutcomeModel should not have failureState field")
	assert.Contains(t, jsonMap, "nextState", "RestOutcomeModel should have nextState field")
	assert.Contains(t, jsonMap, "conditions", "RestOutcomeModel should have conditions field")

	// Test JSON unmarshaling
	var unmarshaledOutcome RestOutcomeModel
	err = json.Unmarshal(jsonData, &unmarshaledOutcome)
	require.NoError(t, err, "Failed to unmarshal RestOutcomeModel")

	// Verify outcome data
	assert.Equal(t, "quest_complete", unmarshaledOutcome.NextState)
	assert.Len(t, unmarshaledOutcome.Conditions, 2)
	assert.Equal(t, "QUEST_ITEM_KEY", unmarshaledOutcome.Conditions[1].ItemId)
}

// TestCompleteConversationModel_APIContract validates full conversation model with string ItemId
func TestCompleteConversationModel_APIContract(t *testing.T) {
	// Create a complete conversation REST model
	conversationRest := RestModel{
		Id:         uuid.New(),
		NpcId:      9001,
		StartState: "greeting",
		States: []RestStateModel{
			{
				Id:        "greeting",
				StateType: "listSelection",
				ListSelection: &RestListSelectionModel{
					Title: "Welcome! How can I help you?",
					Choices: []RestChoiceModel{
						{
							Text:      "I need quest items",
							NextState: "check_items",
						},
						{
							Text:      "Tell me about the quest",
							NextState: "quest_info",
						},
					},
				},
			},
			{
				Id:        "check_items",
				StateType: "genericAction",
				GenericAction: &RestGenericActionModel{
					Operations: []RestOperationModel{
						{
							OperationType: "check_inventory",
							Params: map[string]string{
								"checkType": "quest_items",
							},
						},
					},
					Outcomes: []RestOutcomeModel{
						{
							Conditions: []RestConditionModel{
								{
									Type:     "item",
									Operator: ">=",
									Value:    "3",
									ItemId:   "GOLDEN_MAPLE_LEAF", // String ItemId
								},
								{
									Type:     "item",
									Operator: ">=",
									Value:    "5",
									ItemId:   "4000313", // Numeric string ItemId
								},
							},
							NextState: "items_complete",
						},
						{
							Conditions: []RestConditionModel{}, // No conditions = default outcome
							NextState:  "items_incomplete",
						},
					},
				},
			},
			{
				Id:        "items_complete",
				StateType: "dialogue",
				Dialogue: &RestDialogueModel{
					DialogueType: "npc",
					Text:         "Great! You have all the required items.",
					Choices: []RestChoiceModel{
						{
							Text:      "#LLet's claim the reward!",
							NextState: "reward",
						},
					},
				},
			},
			{
				Id:        "reward",
				StateType: "genericAction",
				GenericAction: &RestGenericActionModel{
					Operations: []RestOperationModel{
						{
							OperationType: "award_item",
							Params: map[string]string{
								"itemId":   "SPECIAL_REWARD_TOKEN",
								"quantity": "1",
							},
						},
						{
							OperationType: "award_exp",
							Params: map[string]string{
								"amount": "5000",
							},
						},
					},
					Outcomes: []RestOutcomeModel{
						{
							Conditions: []RestConditionModel{},
							NextState:  "end", // Single NextState, no success/failure states
						},
					},
				},
			},
		},
	}

	// Test JSON marshaling of complete model
	jsonData, err := json.Marshal(conversationRest)
	require.NoError(t, err, "Failed to marshal complete conversation model")

	// Test JSON unmarshaling
	var unmarshaledConversation RestModel
	err = json.Unmarshal(jsonData, &unmarshaledConversation)
	require.NoError(t, err, "Failed to unmarshal complete conversation model")

	// Verify complex model structure
	assert.Equal(t, conversationRest.NpcId, unmarshaledConversation.NpcId)
	assert.Len(t, unmarshaledConversation.States, 4)

	// Verify check_items state with conditions
	checkItemsState := unmarshaledConversation.States[1]
	require.NotNil(t, checkItemsState.GenericAction)
	assert.Len(t, checkItemsState.GenericAction.Outcomes, 2)

	// Verify first outcome with string ItemId conditions
	firstOutcome := checkItemsState.GenericAction.Outcomes[0]
	assert.Len(t, firstOutcome.Conditions, 2)
	assert.Equal(t, "GOLDEN_MAPLE_LEAF", firstOutcome.Conditions[0].ItemId)
	assert.Equal(t, "4000313", firstOutcome.Conditions[1].ItemId)
	assert.Equal(t, "items_complete", firstOutcome.NextState)

	// Verify reward state outcome has only NextState
	rewardState := unmarshaledConversation.States[3]
	require.NotNil(t, rewardState.GenericAction)
	assert.Len(t, rewardState.GenericAction.Outcomes, 1)
	assert.Equal(t, "end", rewardState.GenericAction.Outcomes[0].NextState)
}

// TestDomainToRestTransformation validates Transform and Extract functions with string ItemId
func TestDomainToRestTransformation(t *testing.T) {
	// Create domain model with string ItemId
	condition1, err := NewConditionBuilder().
		SetType("item").
		SetOperator(">=").
		SetValue("10").
		SetItemId("RARE_CRYSTAL").
		Build()
	require.NoError(t, err)

	condition2, err := NewConditionBuilder().
		SetType("level").
		SetOperator(">=").
		SetValue("50").
		SetItemId(""). // Empty for non-item conditions
		Build()
	require.NoError(t, err)

	outcome, err := NewOutcomeBuilder().
		AddCondition(condition1).
		AddCondition(condition2).
		SetNextState("boss_battle").
		Build()
	require.NoError(t, err)

	genericAction, err := NewGenericActionBuilder().
		AddOperation(OperationModel{
			operationType: "check_requirements",
			params: map[string]string{
				"type": "boss_entry",
			},
		}).
		AddOutcome(outcome).
		Build()
	require.NoError(t, err)

	// Transform to REST model
	restGenericAction, err := TransformGenericAction(*genericAction)
	require.NoError(t, err)

	// Verify transformation
	assert.Len(t, restGenericAction.Outcomes, 1)
	assert.Len(t, restGenericAction.Outcomes[0].Conditions, 2)
	assert.Equal(t, "RARE_CRYSTAL", restGenericAction.Outcomes[0].Conditions[0].ItemId)
	assert.Equal(t, "", restGenericAction.Outcomes[0].Conditions[1].ItemId)
	assert.Equal(t, "boss_battle", restGenericAction.Outcomes[0].NextState)

	// Extract back to domain model
	extractedOutcome, err := ExtractOutcome(restGenericAction.Outcomes[0])
	require.NoError(t, err)

	// Verify round-trip preservation
	assert.Len(t, extractedOutcome.Conditions(), 2)
	assert.Equal(t, "RARE_CRYSTAL", extractedOutcome.Conditions()[0].ItemId())
	assert.Equal(t, "", extractedOutcome.Conditions()[1].ItemId())
	assert.Equal(t, "boss_battle", extractedOutcome.NextState())
}

// TestAPIRequestResponse_StringItemId validates API request/response handling with string ItemId
func TestAPIRequestResponse_StringItemId(t *testing.T) {
	// Create a conversation creation request with string ItemId
	createRequest := RestModel{
		NpcId:      1001,
		StartState: "start",
		States: []RestStateModel{
			{
				Id:        "start",
				StateType: "genericAction",
				GenericAction: &RestGenericActionModel{
					Operations: []RestOperationModel{
						{
							OperationType: "validate_quest",
							Params: map[string]string{
								"questId": "1234",
							},
						},
					},
					Outcomes: []RestOutcomeModel{
						{
							Conditions: []RestConditionModel{
								{
									Type:     "item",
									Operator: "==",
									Value:    "1",
									ItemId:   "LEGENDARY_SWORD", // String ItemId in request
								},
							},
							NextState: "has_sword",
						},
						{
							Conditions: []RestConditionModel{},
							NextState:  "no_sword",
						},
					},
				},
			},
		},
	}

	// Marshal to JSON (simulating API request body)
	requestBody, err := json.Marshal(createRequest)
	require.NoError(t, err)

	// Parse request body (simulating API handler)
	var parsedRequest RestModel
	err = json.NewDecoder(bytes.NewReader(requestBody)).Decode(&parsedRequest)
	require.NoError(t, err)

	// Extract domain model (what handler would do)
	domainModel, err := Extract(parsedRequest)
	require.NoError(t, err)

	// Verify domain model has correct data
	state, err := domainModel.FindState("start")
	require.NoError(t, err)
	require.NotNil(t, state.GenericAction())
	
	outcomes := state.GenericAction().Outcomes()
	require.Len(t, outcomes, 2)
	
	// Check first outcome with condition
	require.Len(t, outcomes[0].Conditions(), 1)
	assert.Equal(t, "LEGENDARY_SWORD", outcomes[0].Conditions()[0].ItemId())

	// Transform back to REST (what response would contain)
	responseModel, err := Transform(domainModel)
	require.NoError(t, err)

	// Marshal response
	responseBody, err := json.Marshal(responseModel)
	require.NoError(t, err)

	// Verify response contains string ItemId
	assert.Contains(t, string(responseBody), "LEGENDARY_SWORD")
}

// TestBackwardCompatibility_NumericStringItemId ensures numeric strings work as before
func TestBackwardCompatibility_NumericStringItemId(t *testing.T) {
	// Test that existing numeric ItemIds still work as strings
	numericItemIds := []string{
		"4001126",
		"2000000",
		"1302000",
		"5220000",
	}

	for _, itemId := range numericItemIds {
		t.Run("ItemId_"+itemId, func(t *testing.T) {
			condition := RestConditionModel{
				Type:     "item",
				Operator: ">=",
				Value:    "1",
				ItemId:   itemId,
			}

			// Marshal and unmarshal
			data, err := json.Marshal(condition)
			require.NoError(t, err)

			var unmarshaled RestConditionModel
			err = json.Unmarshal(data, &unmarshaled)
			require.NoError(t, err)

			// Verify ItemId is preserved
			assert.Equal(t, itemId, unmarshaled.ItemId)
			
			// Verify it can be used in domain model
			domainCondition, err := NewConditionBuilder().
				SetType(condition.Type).
				SetOperator(condition.Operator).
				SetValue(condition.Value).
				SetItemId(condition.ItemId).
				Build()
			require.NoError(t, err)
			assert.Equal(t, itemId, domainCondition.ItemId())
		})
	}
}