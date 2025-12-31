package conversation

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRestModelJSONSerialization_StringReferenceId validates JSON serialization/deserialization with string ReferenceId
func TestRestModelJSONSerialization_StringReferenceId(t *testing.T) {
	tests := []struct {
		name           string
		restModel      RestModel
		expectedReferenceId string
		shouldOmitId   bool
	}{
		{
			name: "Numeric string ReferenceId",
			restModel: RestModel{
				Id:         uuid.New(),
				NpcId:      1001,
				StartState: "start",
				States: []RestStateModel{
					{
						Id:        "start",
						StateType: "genericAction",
						GenericAction: &RestGenericActionModel{
							Operations: []RestOperationModel{
								{
									OperationType: "check_item",
									Params: map[string]string{
										"itemCheck": "validate",
									},
								},
							},
							Outcomes: []RestOutcomeModel{
								{
									Conditions: []RestConditionModel{
										{
											Type:     "item",
											Operator: ">=",
											Value:    "1",
											ReferenceId:   "4001126", // Numeric string
										},
									},
									NextState: "has_item",
								},
							},
						},
					},
				},
			},
			expectedReferenceId: "4001126",
			shouldOmitId:   false,
		},
		{
			name: "Non-numeric string ReferenceId",
			restModel: RestModel{
				Id:         uuid.New(),
				NpcId:      1002,
				StartState: "start",
				States: []RestStateModel{
					{
						Id:        "start",
						StateType: "genericAction",
						GenericAction: &RestGenericActionModel{
							Operations: []RestOperationModel{
								{
									OperationType: "check_quest_item",
									Params: map[string]string{
										"questKey": "check",
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
											ReferenceId:   "LEGENDARY_SWORD_KEY", // Non-numeric string
										},
									},
									NextState: "quest_complete",
								},
							},
						},
					},
				},
			},
			expectedReferenceId: "LEGENDARY_SWORD_KEY",
			shouldOmitId:   false,
		},
		{
			name: "Empty ReferenceId (omitempty)",
			restModel: RestModel{
				Id:         uuid.New(),
				NpcId:      1003,
				StartState: "start",
				States: []RestStateModel{
					{
						Id:        "start",
						StateType: "genericAction",
						GenericAction: &RestGenericActionModel{
							Operations: []RestOperationModel{
								{
									OperationType: "check_level",
									Params: map[string]string{
										"minLevel": "30",
									},
								},
							},
							Outcomes: []RestOutcomeModel{
								{
									Conditions: []RestConditionModel{
										{
											Type:     "level",
											Operator: ">=",
											Value:    "30",
											ReferenceId:   "", // Empty string should be omitted
										},
									},
									NextState: "level_ok",
								},
							},
						},
					},
				},
			},
			expectedReferenceId: "",
			shouldOmitId:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON marshaling
			jsonData, err := json.Marshal(tt.restModel)
			require.NoError(t, err, "Failed to marshal RestModel")

			// Test JSON unmarshaling
			var unmarshaledModel RestModel
			err = json.Unmarshal(jsonData, &unmarshaledModel)
			require.NoError(t, err, "Failed to unmarshal RestModel")

			// Verify ReferenceId is preserved correctly
			state := unmarshaledModel.States[0]
			condition := state.GenericAction.Outcomes[0].Conditions[0]
			
			if tt.shouldOmitId {
				// Verify empty ReferenceId is omitted from JSON
				assert.NotContains(t, string(jsonData), `"referenceId"`, "Empty ReferenceId should be omitted from JSON")
				assert.Equal(t, "", condition.ReferenceId, "Empty ReferenceId should remain empty")
			} else {
				// Verify non-empty ReferenceId is preserved
				assert.Contains(t, string(jsonData), `"referenceId"`, "Non-empty ReferenceId should be included in JSON")
				assert.Equal(t, tt.expectedReferenceId, condition.ReferenceId, "ReferenceId mismatch after JSON round-trip")
			}

			// Verify outcome structure (no success/failure states)
			outcome := state.GenericAction.Outcomes[0]
			assert.Contains(t, string(jsonData), `"nextState"`, "Should have nextState field")
			assert.NotContains(t, string(jsonData), `"successState"`, "Should not have successState field")
			assert.NotContains(t, string(jsonData), `"failureState"`, "Should not have failureState field")
			assert.NotEmpty(t, outcome.NextState, "NextState should not be empty")
		})
	}
}

// TestExtractTransformRoundTrip_NumericReferenceId validates Extract and Transform functions preserve numeric ReferenceId
func TestExtractTransformRoundTrip_NumericReferenceId(t *testing.T) {
	tests := []struct {
		name                    string
		itemIdStr               string
		expectedUint32          uint32
		nextState               string
		expectedRoundTripString string // After round-trip, what string do we expect?
	}{
		{
			name:                    "Numeric string ReferenceId",
			itemIdStr:               "4001126",
			expectedUint32:          4001126,
			nextState:               "numeric_success",
			expectedRoundTripString: "4001126",
		},
		{
			name:                    "Zero ReferenceId converts to empty",
			itemIdStr:               "0",
			expectedUint32:          0,
			nextState:               "zero_success",
			expectedRoundTripString: "", // "0" becomes empty after round-trip
		},
		{
			name:                    "Empty ReferenceId",
			itemIdStr:               "",
			expectedUint32:          0,
			nextState:               "empty_success",
			expectedRoundTripString: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create RestModel with string ReferenceId
			restModel := RestModel{
				Id:         uuid.New(),
				NpcId:      1001,
				StartState: "start",
				States: []RestStateModel{
					{
						Id:        "start",
						StateType: "genericAction",
						GenericAction: &RestGenericActionModel{
							Operations: []RestOperationModel{
								{
									OperationType: "test_operation",
									Params: map[string]string{
										"test": "value",
									},
								},
							},
							Outcomes: []RestOutcomeModel{
								{
									Conditions: []RestConditionModel{
										{
											Type:        "item",
											Operator:    ">=",
											Value:       "1",
											ReferenceId: tt.itemIdStr,
										},
									},
									NextState: tt.nextState,
								},
							},
						},
					},
				},
			}

			// Extract to domain model
			domainModel, err := Extract(restModel)
			require.NoError(t, err, "Failed to extract domain model")

			// Verify domain model has correct uint32 ReferenceId
			state, err := domainModel.FindState("start")
			require.NoError(t, err, "Failed to find state")
			require.NotNil(t, state.GenericAction(), "GenericAction should not be nil")

			outcomes := state.GenericAction().Outcomes()
			require.Len(t, outcomes, 1, "Should have one outcome")

			conditions := outcomes[0].Conditions()
			require.Len(t, conditions, 1, "Should have one condition")

			assert.Equal(t, tt.expectedUint32, conditions[0].ReferenceId(), "Domain model ReferenceId mismatch")
			assert.Equal(t, tt.nextState, outcomes[0].NextState(), "Domain model NextState mismatch")

			// Transform back to REST model
			transformedRest, err := Transform(domainModel)
			require.NoError(t, err, "Failed to transform back to REST model")

			// Verify round-trip preservation
			transformedState := transformedRest.States[0]
			transformedCondition := transformedState.GenericAction.Outcomes[0].Conditions[0]
			transformedOutcome := transformedState.GenericAction.Outcomes[0]

			assert.Equal(t, tt.expectedRoundTripString, transformedCondition.ReferenceId, "Round-trip ReferenceId mismatch")
			assert.Equal(t, tt.nextState, transformedOutcome.NextState, "Round-trip NextState mismatch")
		})
	}
}

// TestOutcomeModelJSONValidation validates OutcomeModel JSON structure
func TestOutcomeModelJSONValidation(t *testing.T) {
	tests := []struct {
		name           string
		jsonInput      string
		expectedValid  bool
		expectedNext   string
		shouldHaveNext bool
	}{
		{
			name: "Valid outcome with nextState only",
			jsonInput: `{
				"conditions": [],
				"nextState": "success"
			}`,
			expectedValid:  true,
			expectedNext:   "success",
			shouldHaveNext: true,
		},
		{
			name: "Valid outcome with conditions and nextState",
			jsonInput: `{
				"conditions": [
					{
						"type": "item",
						"operator": ">=",
						"value": "1",
						"referenceId": "QUEST_TOKEN"
					}
				],
				"nextState": "has_item"
			}`,
			expectedValid:  true,
			expectedNext:   "has_item",
			shouldHaveNext: true,
		},
		{
			name: "Legacy JSON with successState/failureState should ignore them",
			jsonInput: `{
				"conditions": [],
				"nextState": "next",
				"successState": "old_success",
				"failureState": "old_failure"
			}`,
			expectedValid:  true,
			expectedNext:   "next",
			shouldHaveNext: true,
		},
		{
			name: "Empty outcome",
			jsonInput: `{
				"conditions": []
			}`,
			expectedValid:  true,
			expectedNext:   "",
			shouldHaveNext: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var outcome RestOutcomeModel
			err := json.Unmarshal([]byte(tt.jsonInput), &outcome)

			if tt.expectedValid {
				require.NoError(t, err, "Should unmarshal successfully")
				
				if tt.shouldHaveNext {
					assert.Equal(t, tt.expectedNext, outcome.NextState, "NextState mismatch")
				}

				// Test marshaling back excludes legacy fields
				marshaled, err := json.Marshal(outcome)
				require.NoError(t, err, "Should marshal successfully")
				marshaledStr := string(marshaled)
				
				assert.NotContains(t, marshaledStr, "successState", "Should not contain successState")
				assert.NotContains(t, marshaledStr, "failureState", "Should not contain failureState")
				
				if tt.shouldHaveNext {
					assert.Contains(t, marshaledStr, "nextState", "Should contain nextState")
				}
			} else {
				assert.Error(t, err, "Should fail to unmarshal")
			}
		})
	}
}

// TestConditionModelNumericReferenceIdTypes validates different numeric ReferenceId types
func TestConditionModelNumericReferenceIdTypes(t *testing.T) {
	testCases := []struct {
		name           string
		itemIdStr      string
		expectedUint32 uint32
		shouldBuild    bool
	}{
		{"Numeric string", "4001126", 4001126, true},
		{"Zero string", "0", 0, true},
		{"Empty string", "", 0, true},
		{"Large numeric", "2147483647", 2147483647, true},
		{"Max uint32", "4294967295", 4294967295, true},
		{"Non-numeric string returns 0", "SPECIAL_KEY", 0, true},
		{"String with hyphens returns 0", "item-123", 0, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test REST model
			condition := RestConditionModel{
				Type:        "item",
				Operator:    ">=",
				Value:       "1",
				ReferenceId: tc.itemIdStr,
			}

			// Test JSON serialization - REST model always preserves string
			jsonData, err := json.Marshal(condition)
			require.NoError(t, err, "Should marshal successfully")

			var unmarshaled RestConditionModel
			err = json.Unmarshal(jsonData, &unmarshaled)
			require.NoError(t, err, "Should unmarshal successfully")

			assert.Equal(t, tc.itemIdStr, unmarshaled.ReferenceId, "REST ReferenceId should be preserved as string")

			// Test domain model creation
			domainCondition, err := NewConditionBuilder().
				SetType(condition.Type).
				SetOperator(condition.Operator).
				SetValue(condition.Value).
				SetReferenceId(condition.ReferenceId).
				Build()

			if tc.shouldBuild {
				require.NoError(t, err, "Should create domain model successfully")
				// Domain model converts string to uint32
				assert.Equal(t, tc.expectedUint32, domainCondition.ReferenceId(), "Domain ReferenceId should be converted to uint32")
			} else {
				assert.Error(t, err, "Should fail to create domain model")
			}
		})
	}
}

// TestAPIResponseFormat_StringReferenceId validates API response format compliance
func TestAPIResponseFormat_StringReferenceId(t *testing.T) {
	// Create a sample conversation model for transformation
	conversation := createTestConversationModel()
	
	// Transform to REST model
	restModel, err := Transform(conversation)
	require.NoError(t, err, "Should transform successfully")

	// Create mock HTTP response
	responseBody, err := json.Marshal(map[string]interface{}{
		"data": map[string]interface{}{
			"type":       "conversations",
			"id":         restModel.Id.String(),
			"attributes": restModel,
		},
	})
	require.NoError(t, err, "Should marshal response successfully")

	// Create mock HTTP response recorder
	rr := httptest.NewRecorder()
	rr.Header().Set("Content-Type", "application/json")
	rr.WriteHeader(http.StatusOK)
	_, err = rr.Write(responseBody)
	require.NoError(t, err, "Should write response successfully")

	// Validate response format
	assert.Equal(t, http.StatusOK, rr.Code, "Should return 200 OK")
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"), "Should have JSON content type")

	// Parse and validate response structure
	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err, "Should parse response JSON successfully")

	// Validate JSON:API structure
	assert.Contains(t, response, "data", "Should have data field")
	data := response["data"].(map[string]interface{})
	assert.Contains(t, data, "type", "Should have type field")
	assert.Contains(t, data, "id", "Should have id field")
	assert.Contains(t, data, "attributes", "Should have attributes field")

	// Validate string ReferenceId in nested structure
	attributes := data["attributes"].(map[string]interface{})
	states := attributes["states"].([]interface{})
	require.Greater(t, len(states), 0, "Should have at least one state")

	state := states[0].(map[string]interface{})
	if genericAction, ok := state["genericAction"]; ok && genericAction != nil {
		ga := genericAction.(map[string]interface{})
		if outcomes, ok := ga["outcomes"]; ok {
			outcomesSlice := outcomes.([]interface{})
			if len(outcomesSlice) > 0 {
				outcome := outcomesSlice[0].(map[string]interface{})
				if conditions, ok := outcome["conditions"]; ok {
					conditionsSlice := conditions.([]interface{})
					if len(conditionsSlice) > 0 {
						condition := conditionsSlice[0].(map[string]interface{})
						if itemId, ok := condition["referenceId"]; ok {
							// Validate that ReferenceId is a string
							assert.IsType(t, "", itemId, "ReferenceId should be string type")
						}
					}
				}
				// Validate outcome structure
				assert.Contains(t, outcome, "nextState", "Outcome should have nextState")
				assert.NotContains(t, outcome, "successState", "Outcome should not have successState")
				assert.NotContains(t, outcome, "failureState", "Outcome should not have failureState")
			}
		}
	}
}

// Helper function to create a test conversation model
func createTestConversationModel() Model {
	condition, _ := NewConditionBuilder().
		SetType("item").
		SetOperator(">=").
		SetValue("1").
		SetReferenceId("TEST_ITEM_KEY").
		Build()

	outcome, _ := NewOutcomeBuilder().
		AddCondition(condition).
		SetNextState("test_success").
		Build()

	operation := OperationModel{
		operationType: "test_operation",
		params: map[string]string{
			"param1": "value1",
		},
	}

	genericAction, _ := NewGenericActionBuilder().
		AddOperation(operation).
		AddOutcome(outcome).
		Build()

	state := StateModel{
		id:            "test_state",
		stateType:     GenericActionType,
		genericAction: genericAction,
	}

	return Model{
		id:         uuid.New(),
		npcId:      1001,
		startState: "test_state",
		states:     []StateModel{state},
	}
}