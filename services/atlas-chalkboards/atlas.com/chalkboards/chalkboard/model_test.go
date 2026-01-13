package chalkboard

import "testing"

func TestModelAccessors(t *testing.T) {
	m := Model{id: 12345, message: "Hello"}

	if m.Id() != 12345 {
		t.Errorf("Id mismatch. Expected 12345, got %d", m.Id())
	}

	if m.Message() != "Hello" {
		t.Errorf("Message mismatch. Expected 'Hello', got %q", m.Message())
	}
}

func TestModelEmptyMessage(t *testing.T) {
	m := Model{id: 1, message: ""}

	if m.Message() != "" {
		t.Errorf("Expected empty message, got %q", m.Message())
	}
}

func TestBuilder(t *testing.T) {
	m := NewBuilder(12345).SetMessage("Hello, World!").Build()

	if m.Id() != 12345 {
		t.Errorf("Id mismatch. Expected 12345, got %d", m.Id())
	}

	if m.Message() != "Hello, World!" {
		t.Errorf("Message mismatch. Expected 'Hello, World!', got %q", m.Message())
	}
}

func TestBuilderChaining(t *testing.T) {
	b := NewBuilder(1)
	b.SetMessage("first")
	b.SetMessage("second")
	m := b.Build()

	if m.Message() != "second" {
		t.Errorf("Expected last message, got %q", m.Message())
	}
}

func TestBuilderEmptyMessage(t *testing.T) {
	m := NewBuilder(1).Build()

	if m.Message() != "" {
		t.Errorf("Expected empty message by default, got %q", m.Message())
	}
}
