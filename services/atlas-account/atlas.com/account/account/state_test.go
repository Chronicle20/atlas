package account

import "testing"

func TestIsLoggedIn(t *testing.T) {
	tests := []struct {
		name     string
		state    State
		expected bool
	}{
		{"NotLoggedIn", StateNotLoggedIn, false},
		{"LoggedIn", StateLoggedIn, true},
		{"Transition", StateTransition, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsLoggedIn(tt.state)
			if result != tt.expected {
				t.Errorf("IsLoggedIn(%v) = %v, expected %v", tt.state, result, tt.expected)
			}
		})
	}
}

func TestIsTransition(t *testing.T) {
	tests := []struct {
		name     string
		state    State
		expected bool
	}{
		{"NotLoggedIn", StateNotLoggedIn, false},
		{"LoggedIn", StateLoggedIn, false},
		{"Transition", StateTransition, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsTransition(tt.state)
			if result != tt.expected {
				t.Errorf("IsTransition(%v) = %v, expected %v", tt.state, result, tt.expected)
			}
		})
	}
}

func TestLoggedInModelFunction(t *testing.T) {
	st := sampleTenant()
	tests := []struct {
		name     string
		state    State
		expected bool
	}{
		{"NotLoggedIn", StateNotLoggedIn, false},
		{"LoggedIn", StateLoggedIn, true},
		{"Transition", StateTransition, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, _ := NewBuilder(st.Id(), "test").
				SetState(tt.state).
				Build()

			result := LoggedIn(m)
			if result != tt.expected {
				t.Errorf("LoggedIn(model with state %v) = %v, expected %v", tt.state, result, tt.expected)
			}
		})
	}
}
