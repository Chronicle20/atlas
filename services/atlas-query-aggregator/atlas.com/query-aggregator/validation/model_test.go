package validation

import (
	"atlas-query-aggregator/character"
	"testing"
)

func TestNewCondition(t *testing.T) {
	tests := []struct {
		name        string
		expression  string
		wantType    ConditionType
		wantOp      Operator
		wantValue   int
		shouldError bool
	}{
		{
			name:        "Valid job equals condition",
			expression:  "jobId=100",
			wantType:    JobCondition,
			wantOp:      Equals,
			wantValue:   100,
			shouldError: false,
		},
		{
			name:        "Valid meso greater than condition",
			expression:  "meso>10000",
			wantType:    MesoCondition,
			wantOp:      GreaterThan,
			wantValue:   10000,
			shouldError: false,
		},
		{
			name:        "Valid map less than condition",
			expression:  "mapId<2000",
			wantType:    MapCondition,
			wantOp:      LessThan,
			wantValue:   2000,
			shouldError: false,
		},
		{
			name:        "Valid fame greater than or equal condition",
			expression:  "fame>=50",
			wantType:    FameCondition,
			wantOp:      GreaterEqual,
			wantValue:   50,
			shouldError: false,
		},
		{
			name:        "Valid meso less than or equal condition",
			expression:  "meso<=5000",
			wantType:    MesoCondition,
			wantOp:      LessEqual,
			wantValue:   5000,
			shouldError: false,
		},
		{
			name:        "Invalid condition format",
			expression:  "jobId100",
			shouldError: true,
		},
		{
			name:        "Invalid condition type",
			expression:  "level=10",
			shouldError: true,
		},
		{
			name:        "Invalid value",
			expression:  "jobId=abc",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewCondition(tt.expression)
			
			if tt.shouldError {
				if err == nil {
					t.Errorf("NewCondition() error = nil, want error")
				}
				return
			}
			
			if err != nil {
				t.Errorf("NewCondition() error = %v, want nil", err)
				return
			}
			
			if got.conditionType != tt.wantType {
				t.Errorf("NewCondition() conditionType = %v, want %v", got.conditionType, tt.wantType)
			}
			
			if got.operator != tt.wantOp {
				t.Errorf("NewCondition() operator = %v, want %v", got.operator, tt.wantOp)
			}
			
			if got.value != tt.wantValue {
				t.Errorf("NewCondition() value = %v, want %v", got.value, tt.wantValue)
			}
		})
	}
}

func TestCondition_Evaluate(t *testing.T) {
	// Create a test character
	character := character.NewModelBuilder().
		SetJobId(100).
		SetMeso(10000).
		SetMapId(2000).
		SetFame(50).
		Build()

	tests := []struct {
		name        string
		condition   Condition
		wantPassed  bool
		wantContains string
	}{
		{
			name: "Job equals - pass",
			condition: Condition{
				conditionType: JobCondition,
				operator:      Equals,
				value:         100,
			},
			wantPassed:  true,
			wantContains: "Job ID = 100",
		},
		{
			name: "Job equals - fail",
			condition: Condition{
				conditionType: JobCondition,
				operator:      Equals,
				value:         200,
			},
			wantPassed:  false,
			wantContains: "Job ID = 200",
		},
		{
			name: "Meso greater than - pass",
			condition: Condition{
				conditionType: MesoCondition,
				operator:      GreaterThan,
				value:         9000,
			},
			wantPassed:  true,
			wantContains: "Meso > 9000",
		},
		{
			name: "Meso greater than - fail",
			condition: Condition{
				conditionType: MesoCondition,
				operator:      GreaterThan,
				value:         11000,
			},
			wantPassed:  false,
			wantContains: "Meso > 11000",
		},
		{
			name: "Map less than - pass",
			condition: Condition{
				conditionType: MapCondition,
				operator:      LessThan,
				value:         3000,
			},
			wantPassed:  true,
			wantContains: "Map ID < 3000",
		},
		{
			name: "Map less than - fail",
			condition: Condition{
				conditionType: MapCondition,
				operator:      LessThan,
				value:         1000,
			},
			wantPassed:  false,
			wantContains: "Map ID < 1000",
		},
		{
			name: "Fame greater than or equal - pass (equal)",
			condition: Condition{
				conditionType: FameCondition,
				operator:      GreaterEqual,
				value:         50,
			},
			wantPassed:  true,
			wantContains: "Fame >= 50",
		},
		{
			name: "Fame greater than or equal - pass (greater)",
			condition: Condition{
				conditionType: FameCondition,
				operator:      GreaterEqual,
				value:         40,
			},
			wantPassed:  true,
			wantContains: "Fame >= 40",
		},
		{
			name: "Fame greater than or equal - fail",
			condition: Condition{
				conditionType: FameCondition,
				operator:      GreaterEqual,
				value:         60,
			},
			wantPassed:  false,
			wantContains: "Fame >= 60",
		},
		{
			name: "Meso less than or equal - pass (equal)",
			condition: Condition{
				conditionType: MesoCondition,
				operator:      LessEqual,
				value:         10000,
			},
			wantPassed:  true,
			wantContains: "Meso <= 10000",
		},
		{
			name: "Meso less than or equal - pass (less)",
			condition: Condition{
				conditionType: MesoCondition,
				operator:      LessEqual,
				value:         11000,
			},
			wantPassed:  true,
			wantContains: "Meso <= 11000",
		},
		{
			name: "Meso less than or equal - fail",
			condition: Condition{
				conditionType: MesoCondition,
				operator:      LessEqual,
				value:         9000,
			},
			wantPassed:  false,
			wantContains: "Meso <= 9000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPassed, gotDescription := tt.condition.Evaluate(character)
			
			if gotPassed != tt.wantPassed {
				t.Errorf("Condition.Evaluate() passed = %v, want %v", gotPassed, tt.wantPassed)
			}
			
			if gotDescription != tt.wantContains {
				t.Errorf("Condition.Evaluate() description = %v, want to contain %v", gotDescription, tt.wantContains)
			}
		})
	}
}

func TestValidationResult(t *testing.T) {
	t.Run("New validation result", func(t *testing.T) {
		result := NewValidationResult(123)
		
		if !result.Passed() {
			t.Errorf("NewValidationResult() passed = %v, want true", result.Passed())
		}
		
		if len(result.Details()) != 0 {
			t.Errorf("NewValidationResult() details length = %v, want 0", len(result.Details()))
		}
		
		if result.CharacterId() != 123 {
			t.Errorf("NewValidationResult() characterId = %v, want 123", result.CharacterId())
		}
	})
	
	t.Run("Add passing result", func(t *testing.T) {
		result := NewValidationResult(123)
		result.AddResult(true, "Test condition")
		
		if !result.Passed() {
			t.Errorf("After AddResult(true) passed = %v, want true", result.Passed())
		}
		
		if len(result.Details()) != 1 {
			t.Errorf("After AddResult() details length = %v, want 1", len(result.Details()))
		}
		
		if result.Details()[0] != "Passed: Test condition" {
			t.Errorf("After AddResult() detail = %v, want 'Passed: Test condition'", result.Details()[0])
		}
	})
	
	t.Run("Add failing result", func(t *testing.T) {
		result := NewValidationResult(123)
		result.AddResult(false, "Test condition")
		
		if result.Passed() {
			t.Errorf("After AddResult(false) passed = %v, want false", result.Passed())
		}
		
		if len(result.Details()) != 1 {
			t.Errorf("After AddResult() details length = %v, want 1", len(result.Details()))
		}
		
		if result.Details()[0] != "Failed: Test condition" {
			t.Errorf("After AddResult() detail = %v, want 'Failed: Test condition'", result.Details()[0])
		}
	})
	
	t.Run("Multiple results", func(t *testing.T) {
		result := NewValidationResult(123)
		result.AddResult(true, "Condition 1")
		result.AddResult(true, "Condition 2")
		result.AddResult(false, "Condition 3")
		
		if result.Passed() {
			t.Errorf("After mixed AddResult calls passed = %v, want false", result.Passed())
		}
		
		if len(result.Details()) != 3 {
			t.Errorf("After multiple AddResult calls details length = %v, want 3", len(result.Details()))
		}
	})
}