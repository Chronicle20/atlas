package condition

import (
	"reflect"
	"testing"
)

func TestBuilderBuild(t *testing.T) {
	tests := []struct {
		name    string
		build   func() (Model, error)
		wantErr string
		check   func(t *testing.T, m Model)
	}{
		{
			name: "SetValues",
			build: func() (Model, error) {
				return NewBuilder().SetType("jobId").SetOperator("in").SetValues([]string{"1000", "1100", "1110"}).Build()
			},
			check: func(t *testing.T, m Model) {
				if !reflect.DeepEqual(m.Values(), []string{"1000", "1100", "1110"}) {
					t.Fatalf("expected Values() to be [1000 1100 1110], got %v", m.Values())
				}
				if m.Value() != "" {
					t.Fatalf("expected Value() to be empty, got %q", m.Value())
				}
			},
		},
		{
			name: "AddValue",
			build: func() (Model, error) {
				return NewBuilder().SetType("jobId").SetOperator("in").AddValue("1000").AddValue("1100").Build()
			},
			check: func(t *testing.T, m Model) {
				if !reflect.DeepEqual(m.Values(), []string{"1000", "1100"}) {
					t.Fatalf("expected Values() to be [1000 1100], got %v", m.Values())
				}
			},
		},
		{
			name: "ValuesOmittedDefaultsNil",
			build: func() (Model, error) {
				return NewBuilder().SetType("level").SetOperator(">=").SetValue("10").Build()
			},
			check: func(t *testing.T, m Model) {
				if m.Values() != nil {
					t.Fatalf("expected Values() to be nil, got %v", m.Values())
				}
				if m.Value() != "10" {
					t.Fatalf("expected Value() to be \"10\", got %q", m.Value())
				}
			},
		},
		{
			name: "RequiresValueOrValues",
			build: func() (Model, error) {
				return NewBuilder().SetType("level").SetOperator(">=").Build()
			},
			wantErr: "value or values is required",
		},
		{
			name: "AcceptsValuesWithoutValue",
			build: func() (Model, error) {
				return NewBuilder().SetType("jobId").SetOperator("in").SetValues([]string{"1000"}).Build()
			},
		},
		{
			name: "StillRequiresType",
			build: func() (Model, error) {
				return NewBuilder().SetOperator("=").SetValue("1").Build()
			},
			wantErr: "condition type is required",
		},
		{
			name: "StillRequiresOperator",
			build: func() (Model, error) {
				return NewBuilder().SetType("level").SetValue("1").Build()
			},
			wantErr: "operator is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := tt.build()
			if tt.wantErr != "" {
				if err == nil {
					t.Fatal("expected an error, got nil")
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("expected error %q, got %q", tt.wantErr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.check != nil {
				tt.check(t, m)
			}
		})
	}
}
