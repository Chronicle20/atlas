package tradeability

import (
	"reflect"
	"testing"
)

func TestTransformRoundTrip(t *testing.T) {
	tests := []struct {
		name      string
		input     Model
		transform func(Model) (Model, error)
	}{
		{
			name:  "equipment",
			input: NewModel(true, 77, false),
			transform: func(m Model) (Model, error) {
				rm, err := TransformEquipment(m)
				if err != nil {
					return Model{}, err
				}
				return extract(rm)
			},
		},
		{
			name:  "consumable",
			input: NewModel(false, 88, true),
			transform: func(m Model) (Model, error) {
				rm, err := TransformConsumable(m)
				if err != nil {
					return Model{}, err
				}
				return extract(rm)
			},
		},
		{
			name:  "setup",
			input: NewModel(true, 99, true),
			transform: func(m Model) (Model, error) {
				rm, err := TransformSetup(m)
				if err != nil {
					return Model{}, err
				}
				return extract(rm)
			},
		},
		{
			name:  "etc",
			input: NewModel(false, 111, false),
			transform: func(m Model) (Model, error) {
				rm, err := TransformEtc(m)
				if err != nil {
					return Model{}, err
				}
				return extract(rm)
			},
		},
		{
			name:  "cash",
			input: NewModel(true, 222, true),
			transform: func(m Model) (Model, error) {
				rm, err := TransformCash(m)
				if err != nil {
					return Model{}, err
				}
				return extract(rm)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.transform(tt.input)
			if err != nil {
				t.Fatalf("round trip failed: %v", err)
			}
			if !reflect.DeepEqual(got, tt.input) {
				t.Errorf("round trip mismatch. Expected %+v, got %+v", tt.input, got)
			}
		})
	}
}
