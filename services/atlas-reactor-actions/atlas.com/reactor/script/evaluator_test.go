package script

import (
	"testing"

	"github.com/Chronicle20/atlas-script-core/condition"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

func TestEvaluateReactorStateCondition(t *testing.T) {
	tests := []struct {
		name         string
		reactorState int8
		operator     string
		value        string
		want         bool
		wantErr      bool
	}{
		// Equality tests
		{
			name:         "equal - match",
			reactorState: 0,
			operator:     "=",
			value:        "0",
			want:         true,
			wantErr:      false,
		},
		{
			name:         "equal - no match",
			reactorState: 1,
			operator:     "=",
			value:        "0",
			want:         false,
			wantErr:      false,
		},
		// Not equal tests
		{
			name:         "not equal - match",
			reactorState: 1,
			operator:     "!=",
			value:        "0",
			want:         true,
			wantErr:      false,
		},
		{
			name:         "not equal - no match",
			reactorState: 0,
			operator:     "!=",
			value:        "0",
			want:         false,
			wantErr:      false,
		},
		// Greater than tests
		{
			name:         "greater than - match",
			reactorState: 2,
			operator:     ">",
			value:        "1",
			want:         true,
			wantErr:      false,
		},
		{
			name:         "greater than - no match equal",
			reactorState: 1,
			operator:     ">",
			value:        "1",
			want:         false,
			wantErr:      false,
		},
		{
			name:         "greater than - no match less",
			reactorState: 0,
			operator:     ">",
			value:        "1",
			want:         false,
			wantErr:      false,
		},
		// Less than tests
		{
			name:         "less than - match",
			reactorState: 0,
			operator:     "<",
			value:        "1",
			want:         true,
			wantErr:      false,
		},
		{
			name:         "less than - no match equal",
			reactorState: 1,
			operator:     "<",
			value:        "1",
			want:         false,
			wantErr:      false,
		},
		{
			name:         "less than - no match greater",
			reactorState: 2,
			operator:     "<",
			value:        "1",
			want:         false,
			wantErr:      false,
		},
		// Greater than or equal tests
		{
			name:         "greater or equal - match greater",
			reactorState: 2,
			operator:     ">=",
			value:        "1",
			want:         true,
			wantErr:      false,
		},
		{
			name:         "greater or equal - match equal",
			reactorState: 1,
			operator:     ">=",
			value:        "1",
			want:         true,
			wantErr:      false,
		},
		{
			name:         "greater or equal - no match",
			reactorState: 0,
			operator:     ">=",
			value:        "1",
			want:         false,
			wantErr:      false,
		},
		// Less than or equal tests
		{
			name:         "less or equal - match less",
			reactorState: 0,
			operator:     "<=",
			value:        "1",
			want:         true,
			wantErr:      false,
		},
		{
			name:         "less or equal - match equal",
			reactorState: 1,
			operator:     "<=",
			value:        "1",
			want:         true,
			wantErr:      false,
		},
		{
			name:         "less or equal - no match",
			reactorState: 2,
			operator:     "<=",
			value:        "1",
			want:         false,
			wantErr:      false,
		},
		// Error cases
		{
			name:         "invalid value",
			reactorState: 0,
			operator:     "=",
			value:        "invalid",
			want:         false,
			wantErr:      true,
		},
		{
			name:         "unknown operator",
			reactorState: 0,
			operator:     "??",
			value:        "0",
			want:         false,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EvaluateReactorStateCondition(tt.reactorState, tt.operator, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("EvaluateReactorStateCondition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("EvaluateReactorStateCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConditionEvaluator_EvaluateCondition(t *testing.T) {
	logger, _ := test.NewNullLogger()

	tests := []struct {
		name         string
		reactorState int8
		condType     string
		operator     string
		value        string
		want         bool
		wantErr      bool
	}{
		{
			name:         "reactor_state condition - match",
			reactorState: 0,
			condType:     "reactor_state",
			operator:     "=",
			value:        "0",
			want:         true,
			wantErr:      false,
		},
		{
			name:         "reactor_state condition - no match",
			reactorState: 1,
			condType:     "reactor_state",
			operator:     "=",
			value:        "0",
			want:         false,
			wantErr:      false,
		},
		{
			name:         "unknown condition type",
			reactorState: 0,
			condType:     "unknown_type",
			operator:     "=",
			value:        "0",
			want:         false,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluator := NewConditionEvaluator(logger)

			cond, _ := condition.NewBuilder().
				SetType(tt.condType).
				SetOperator(tt.operator).
				SetValue(tt.value).
				Build()

			got, err := evaluator.EvaluateCondition(tt.reactorState, cond)
			if (err != nil) != tt.wantErr {
				t.Errorf("EvaluateCondition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("EvaluateCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConditionEvaluator_EvaluateRule(t *testing.T) {
	logger, _ := test.NewNullLogger()

	tests := []struct {
		name         string
		reactorState int8
		conditions   []struct {
			condType string
			operator string
			value    string
		}
		want    bool
		wantErr bool
	}{
		{
			name:         "empty conditions - always match",
			reactorState: 0,
			conditions:   []struct{ condType, operator, value string }{},
			want:         true,
			wantErr:      false,
		},
		{
			name:         "single condition - match",
			reactorState: 0,
			conditions: []struct{ condType, operator, value string }{
				{"reactor_state", "=", "0"},
			},
			want:    true,
			wantErr: false,
		},
		{
			name:         "single condition - no match",
			reactorState: 1,
			conditions: []struct{ condType, operator, value string }{
				{"reactor_state", "=", "0"},
			},
			want:    false,
			wantErr: false,
		},
		{
			name:         "multiple conditions - all match",
			reactorState: 2,
			conditions: []struct{ condType, operator, value string }{
				{"reactor_state", ">=", "1"},
				{"reactor_state", "<=", "3"},
			},
			want:    true,
			wantErr: false,
		},
		{
			name:         "multiple conditions - one fails",
			reactorState: 5,
			conditions: []struct{ condType, operator, value string }{
				{"reactor_state", ">=", "1"},
				{"reactor_state", "<=", "3"},
			},
			want:    false,
			wantErr: false,
		},
		{
			name:         "condition with error",
			reactorState: 0,
			conditions: []struct{ condType, operator, value string }{
				{"unknown_type", "=", "0"},
			},
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluator := NewConditionEvaluator(logger)

			rb := NewRuleBuilder().SetId("test_rule")
			for _, c := range tt.conditions {
				cond, _ := condition.NewBuilder().
					SetType(c.condType).
					SetOperator(c.operator).
					SetValue(c.value).
					Build()
				rb.AddCondition(cond)
			}
			rule := rb.Build()

			got, err := evaluator.EvaluateRule(tt.reactorState, rule)
			if (err != nil) != tt.wantErr {
				t.Errorf("EvaluateRule() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("EvaluateRule() = %v, want %v", got, tt.want)
			}
		})
	}
}

func newTestLogger() logrus.FieldLogger {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	return logger
}
