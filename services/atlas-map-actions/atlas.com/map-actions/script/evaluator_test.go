package script

import (
	"atlas-map-actions/validation"
	"atlas-map-actions/validation/mock"
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-script-core/condition"
)

func newTestConditionEvaluator() *ConditionEvaluator {
	l, _ := test.NewNullLogger()
	return NewConditionEvaluator(l, context.Background())
}

func TestEvaluateMapIdOperators(t *testing.T) {
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).Build()

	tests := []struct {
		name     string
		operator string
		value    string
		want     bool
		wantErr  string
	}{
		{"eq match", "=", "100000000", true, ""},
		{"eq no match", "=", "100000001", false, ""},
		{"eq eq match", "==", "100000000", true, ""},
		{"ne match", "!=", "100000001", true, ""},
		{"ne no match", "!=", "100000000", false, ""},
		{"gt true", ">", "99999999", true, ""},
		{"gt false equal", ">", "100000000", false, ""},
		{"gt false above", ">", "100000001", false, ""},
		{"lt true", "<", "100000001", true, ""},
		{"lt false equal", "<", "100000000", false, ""},
		{"gte true equal", ">=", "100000000", true, ""},
		{"gte true below", ">=", "99999999", true, ""},
		{"gte false", ">=", "100000001", false, ""},
		{"lte true equal", "<=", "100000000", true, ""},
		{"lte true above", "<=", "100000001", true, ""},
		{"lte false", "<=", "99999999", false, ""},
		{"unsupported operator", "~=", "100000000", false, "unsupported operator [~=] for map_id condition"},
	}

	e := newTestConditionEvaluator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cond, err := condition.NewBuilder().SetType("map_id").SetOperator(tt.operator).SetValue(tt.value).Build()
			if err != nil {
				t.Fatalf("condition.NewBuilder().Build(): %v", err)
			}

			got, err := e.EvaluateCondition(f, 1, cond)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("EvaluateCondition() error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("EvaluateCondition() unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("EvaluateCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvaluateViaQueryAggregatorCarriesEveryField(t *testing.T) {
	var captured []validation.ConditionInput
	e := newTestConditionEvaluator()
	e.validationP = &mock.ProcessorMock{
		ValidateCharacterStateFunc: func(characterId uint32, conditions []validation.ConditionInput) (validation.ValidationResult, error) {
			captured = conditions
			return validation.NewValidationResult(characterId, true), nil
		},
	}

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(910510000)).Build()
	cond, err := condition.NewBuilder().
		SetType("questProgress").
		SetOperator("=").
		SetValue("0").
		SetReferenceId("21747").
		SetStep("9300351").
		SetIncludeEquipped(true).
		Build()
	if err != nil {
		t.Fatalf("condition.NewBuilder().Build(): %v", err)
	}

	if _, err := e.EvaluateCondition(f, 1, cond); err != nil {
		t.Fatalf("EvaluateCondition() unexpected error: %v", err)
	}

	if len(captured) != 1 {
		t.Fatalf("len(captured) = %d, want 1", len(captured))
	}

	want := validation.ConditionInput{
		Type:            "questProgress",
		Operator:        "=",
		Value:           0,
		Values:          nil,
		ReferenceId:     21747,
		Step:            "9300351",
		WorldId:         world.Id(0),
		ChannelId:       channel.Id(1),
		IncludeEquipped: true,
	}
	if !reflect.DeepEqual(captured[0], want) {
		t.Errorf("captured[0] = %+v, want %+v", captured[0], want)
	}
}

func TestEvaluateViaQueryAggregatorInOperatorUsesValues(t *testing.T) {
	var captured []validation.ConditionInput
	e := newTestConditionEvaluator()
	e.validationP = &mock.ProcessorMock{
		ValidateCharacterStateFunc: func(characterId uint32, conditions []validation.ConditionInput) (validation.ValidationResult, error) {
			captured = conditions
			return validation.NewValidationResult(characterId, true), nil
		},
	}

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(910510000)).Build()
	cond, err := condition.NewBuilder().
		SetType("jobId").
		SetOperator("in").
		SetValues([]string{"1000", "1100", "1110", "1200", "1210", "1300", "1310", "1400", "1410", "1500", "1510"}).
		Build()
	if err != nil {
		t.Fatalf("condition.NewBuilder().Build(): %v", err)
	}

	if _, err := e.EvaluateCondition(f, 1, cond); err != nil {
		t.Fatalf("EvaluateCondition() unexpected error: %v", err)
	}

	if len(captured) != 1 {
		t.Fatalf("len(captured) = %d, want 1", len(captured))
	}

	wantValues := []int{1000, 1100, 1110, 1200, 1210, 1300, 1310, 1400, 1410, 1500, 1510}
	if captured[0].Value != 0 {
		t.Errorf("captured[0].Value = %d, want 0", captured[0].Value)
	}
	if !reflect.DeepEqual(captured[0].Values, wantValues) {
		t.Errorf("captured[0].Values = %v, want %v", captured[0].Values, wantValues)
	}
}

func TestEvaluateViaQueryAggregatorRejectsNonIntegerScalarValue(t *testing.T) {
	called := false
	e := newTestConditionEvaluator()
	e.validationP = &mock.ProcessorMock{
		ValidateCharacterStateFunc: func(characterId uint32, conditions []validation.ConditionInput) (validation.ValidationResult, error) {
			called = true
			return validation.NewValidationResult(characterId, true), nil
		},
	}

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(910510000)).Build()
	cond, err := condition.NewBuilder().SetType("level").SetOperator(">=").SetValue("ten").Build()
	if err != nil {
		t.Fatalf("condition.NewBuilder().Build(): %v", err)
	}

	_, err = e.EvaluateCondition(f, 1, cond)
	if err == nil || !strings.Contains(err.Error(), "invalid condition value [ten]") {
		t.Fatalf("EvaluateCondition() error = %v, want containing %q", err, "invalid condition value [ten]")
	}
	if called {
		t.Errorf("aggregator was called, want it never called")
	}
}

func TestEvaluateViaQueryAggregatorRejectsNonIntegerValuesEntry(t *testing.T) {
	e := newTestConditionEvaluator()

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(910510000)).Build()
	cond, err := condition.NewBuilder().SetType("jobId").SetOperator("in").SetValues([]string{"1000", "abc"}).Build()
	if err != nil {
		t.Fatalf("condition.NewBuilder().Build(): %v", err)
	}

	_, err = e.EvaluateCondition(f, 1, cond)
	if err == nil || !strings.Contains(err.Error(), "invalid condition values entry [abc]") {
		t.Fatalf("EvaluateCondition() error = %v, want containing %q", err, "invalid condition values entry [abc]")
	}
}
