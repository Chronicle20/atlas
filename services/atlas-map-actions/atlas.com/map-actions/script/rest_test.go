package script

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-script-core/condition"
)

func fullRestConditionModel() RestConditionModel {
	return RestConditionModel{
		Type:            "questProgress",
		Operator:        "=",
		Value:           "0",
		Values:          []string{"1000", "1100"},
		ReferenceId:     "21747",
		Step:            "9300351",
		WorldId:         "0",
		ChannelId:       "1",
		IncludeEquipped: true,
		ValueString:     "miss=o",
	}
}

// TestConditionCarriesEveryField verifies that every field on a
// RestConditionModel and a seed-JSON condition survives their respective
// paths into a condition.Model.
func TestConditionCarriesEveryField(t *testing.T) {
	tests := []struct {
		name  string
		build func() (condition.Model, error)
	}{
		{
			name: "extractCondition",
			build: func() (condition.Model, error) {
				return extractCondition(fullRestConditionModel())
			},
		},
		{
			name: "convertJsonCondition",
			build: func() (condition.Model, error) {
				raw := `{"type":"questProgress","operator":"=","value":"0","values":["1000","1100"],"referenceId":"21747","step":"9300351","worldId":"0","channelId":"1","includeEquipped":true,"valueString":"miss=o"}`

				var jc jsonCondition
				if err := json.Unmarshal([]byte(raw), &jc); err != nil {
					return condition.Model{}, err
				}
				return convertJsonCondition(jc)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := tt.build()
			if err != nil {
				t.Fatalf("build: %v", err)
			}

			if c.Type() != "questProgress" {
				t.Errorf("Type() = %q, want %q", c.Type(), "questProgress")
			}
			if c.Operator() != "=" {
				t.Errorf("Operator() = %q, want %q", c.Operator(), "=")
			}
			if c.Value() != "0" {
				t.Errorf("Value() = %q, want %q", c.Value(), "0")
			}
			if !reflect.DeepEqual(c.Values(), []string{"1000", "1100"}) {
				t.Errorf("Values() = %v, want %v", c.Values(), []string{"1000", "1100"})
			}
			if c.ReferenceIdRaw() != "21747" {
				t.Errorf("ReferenceIdRaw() = %q, want %q", c.ReferenceIdRaw(), "21747")
			}
			if c.Step() != "9300351" {
				t.Errorf("Step() = %q, want %q", c.Step(), "9300351")
			}
			if c.WorldId() != "0" {
				t.Errorf("WorldId() = %q, want %q", c.WorldId(), "0")
			}
			if c.ChannelId() != "1" {
				t.Errorf("ChannelId() = %q, want %q", c.ChannelId(), "1")
			}
			if !c.IncludeEquipped() {
				t.Errorf("IncludeEquipped() = %v, want %v", c.IncludeEquipped(), true)
			}
			if c.ValueString() != "miss=o" {
				t.Errorf("ValueString() = %q, want %q", c.ValueString(), "miss=o")
			}
		})
	}
}

// TestTransformRuleRoundTripsEveryConditionField verifies that a condition.Model
// built with every field populated round-trips through transformRule back into
// an identical RestConditionModel.
func TestTransformRuleRoundTripsEveryConditionField(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "round-trips every field"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := fullRestConditionModel()

			c, err := condition.NewBuilder().
				SetType("questProgress").
				SetOperator("=").
				SetValue("0").
				SetValues([]string{"1000", "1100"}).
				SetReferenceId("21747").
				SetStep("9300351").
				SetWorldId("0").
				SetChannelId("1").
				SetIncludeEquipped(true).
				SetValueString("miss=o").
				Build()
			if err != nil {
				t.Fatalf("condition.NewBuilder().Build(): %v", err)
			}

			rule := NewRuleBuilder().SetId("r1").AddCondition(c).Build()

			restRule := transformRule(rule)
			if len(restRule.Conditions) != 1 {
				t.Fatalf("len(restRule.Conditions) = %d, want 1", len(restRule.Conditions))
			}

			got := restRule.Conditions[0]
			if !reflect.DeepEqual(got, want) {
				t.Errorf("transformRule condition = %+v, want %+v", got, want)
			}
		})
	}
}
