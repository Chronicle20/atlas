package saga

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUnmarshalGenericPayload(t *testing.T) {
	tests := []struct {
		name          string
		rawPayload    interface{}
		expectError   bool
		errorContains string
	}{
		{
			name: "Valid AwardInventory payload",
			rawPayload: map[string]interface{}{
				"characterId": float64(12345),
				"item": map[string]interface{}{
					"templateId": float64(2000),
					"quantity":   float64(5),
				},
			},
			expectError: false,
		},
		{
			name: "Invalid payload structure",
			rawPayload: map[string]interface{}{
				"characterId": "not-a-number",
			},
			expectError:   true,
			errorContains: "json: cannot unmarshal string into Go struct field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test unmarshalAwardInventoryPayload
			result, err := unmarshalAwardInventoryPayload(tt.rawPayload)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)

				// Verify the result is of the correct type
				payload, ok := result.(AwardItemActionPayload)
				assert.True(t, ok)

				// For the valid test case, verify the values
				if tt.name == "Valid AwardInventory payload" {
					assert.Equal(t, uint32(12345), payload.CharacterId)
					assert.Equal(t, uint32(2000), payload.Item.TemplateId)
					assert.Equal(t, uint32(5), payload.Item.Quantity)
				}
			}
		})
	}
}

func TestUnmarshalPayload(t *testing.T) {
	tests := []struct {
		name          string
		action        Action
		rawPayload    interface{}
		expectError   bool
		errorContains string
	}{
		{
			name:   "AwardInventory action",
			action: AwardInventory,
			rawPayload: map[string]interface{}{
				"characterId": float64(12345),
				"item": map[string]interface{}{
					"templateId": float64(2000),
					"quantity":   float64(5),
				},
			},
			expectError: false,
		},
		{
			name:   "AwardExperience action",
			action: AwardExperience,
			rawPayload: map[string]interface{}{
				"characterId": float64(12345),
				"worldId":     float64(0),
				"channelId":   float64(0),
				"distributions": []interface{}{
					map[string]interface{}{
						"experienceType": "WHITE",
						"amount":         float64(1000),
						"attr1":          float64(0),
					},
				},
			},
			expectError: false,
		},
		{
			name:   "AwardLevel action",
			action: AwardLevel,
			rawPayload: map[string]interface{}{
				"characterId": float64(12345),
				"worldId":     float64(0),
				"channelId":   float64(0),
				"amount":      float64(2),
			},
			expectError: false,
		},
		{
			name:   "WarpToRandomPortal action",
			action: WarpToRandomPortal,
			rawPayload: map[string]interface{}{
				"characterId": float64(12345),
				"fieldId":     "0:1:0:00000000-0000-0000-0000-000000000000",
			},
			expectError: false,
		},
		{
			name:   "WarpToPortal action",
			action: WarpToPortal,
			rawPayload: map[string]interface{}{
				"characterId": float64(12345),
				"fieldId":     "0:1:0:00000000-0000-0000-0000-000000000000",
				"portalId":    float64(1),
			},
			expectError: false,
		},
		{
			name:   "DestroyAsset action",
			action: DestroyAsset,
			rawPayload: map[string]interface{}{
				"characterId": float64(12345),
				"templateId":  float64(2000),
				"quantity":    float64(5),
			},
			expectError: false,
		},
		{
			name:          "Unknown action",
			action:        "unknown_action",
			rawPayload:    map[string]interface{}{},
			expectError:   false,
			errorContains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := unmarshalPayload(tt.action, tt.rawPayload)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)

				// For unknown action, the raw payload should be returned as is
				if tt.action == "unknown_action" {
					assert.Equal(t, tt.rawPayload, result)
				} else {
					assert.NotNil(t, result)
				}
			}
		})
	}
}

func TestParseTime(t *testing.T) {
	tests := []struct {
		name    string
		timeStr string
		isValid bool
	}{
		{
			name:    "Valid RFC3339 time",
			timeStr: "2023-01-01T00:00:00Z",
			isValid: true,
		},
		{
			name:    "Invalid time format",
			timeStr: "not-a-time",
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTime(tt.timeStr)

			// Always returns a time.Time, even for invalid input
			assert.NotNil(t, result)

			if tt.isValid {
				// For valid input, the result should be the parsed time
				assert.Equal(t, "2023-01-01 00:00:00 +0000 UTC", result.String())
			} else {
				// For invalid input, the result should be the current time
				// We can't assert the exact value, but we can check it's not zero
				assert.False(t, result.IsZero())
			}
		})
	}
}
