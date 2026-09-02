package condition

import (
	"reflect"
	"testing"
)

func TestBuilderSetValues(t *testing.T) {
	m, err := NewBuilder().SetType("jobId").SetOperator("in").SetValues([]string{"1000", "1100", "1110"}).Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(m.Values(), []string{"1000", "1100", "1110"}) {
		t.Fatalf("expected Values() to be [1000 1100 1110], got %v", m.Values())
	}
	if m.Value() != "" {
		t.Fatalf("expected Value() to be empty, got %q", m.Value())
	}
}

func TestBuilderAddValue(t *testing.T) {
	m, err := NewBuilder().SetType("jobId").SetOperator("in").AddValue("1000").AddValue("1100").Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(m.Values(), []string{"1000", "1100"}) {
		t.Fatalf("expected Values() to be [1000 1100], got %v", m.Values())
	}
}

func TestBuilderValuesOmittedDefaultsNil(t *testing.T) {
	m, err := NewBuilder().SetType("level").SetOperator(">=").SetValue("10").Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Values() != nil {
		t.Fatalf("expected Values() to be nil, got %v", m.Values())
	}
	if m.Value() != "10" {
		t.Fatalf("expected Value() to be \"10\", got %q", m.Value())
	}
}

func TestBuildRequiresValueOrValues(t *testing.T) {
	_, err := NewBuilder().SetType("level").SetOperator(">=").Build()
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if err.Error() != "value or values is required" {
		t.Fatalf("expected error %q, got %q", "value or values is required", err.Error())
	}
}

func TestBuildAcceptsValuesWithoutValue(t *testing.T) {
	_, err := NewBuilder().SetType("jobId").SetOperator("in").SetValues([]string{"1000"}).Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildStillRequiresType(t *testing.T) {
	_, err := NewBuilder().SetOperator("=").SetValue("1").Build()
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if err.Error() != "condition type is required" {
		t.Fatalf("expected error %q, got %q", "condition type is required", err.Error())
	}
}

func TestBuildStillRequiresOperator(t *testing.T) {
	_, err := NewBuilder().SetType("level").SetValue("1").Build()
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if err.Error() != "operator is required" {
		t.Fatalf("expected error %q, got %q", "operator is required", err.Error())
	}
}
