package guild

import "testing"

func TestNewModel(t *testing.T) {
	tests := []struct {
		name     string
		id       uint32
		guildName string
		rank     uint32
		expected Model
	}{
		{
			name:     "valid guild model",
			id:       123,
			guildName: "TestGuild",
			rank:     5,
			expected: Model{
				id:     123,
				name:   "TestGuild",
				rank:   5,
				member: true,
			},
		},
		{
			name:     "empty guild model with zero ID",
			id:       0,
			guildName: "",
			rank:     0,
			expected: Model{
				id:     0,
				name:   "",
				rank:   0,
				member: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewModel(tt.id, tt.guildName, tt.rank)
			
			if result.Id() != tt.expected.Id() {
				t.Errorf("expected ID %d, got %d", tt.expected.Id(), result.Id())
			}
			
			if result.Name() != tt.expected.Name() {
				t.Errorf("expected name %s, got %s", tt.expected.Name(), result.Name())
			}
			
			if result.Rank() != tt.expected.Rank() {
				t.Errorf("expected rank %d, got %d", tt.expected.Rank(), result.Rank())
			}
			
			if result.IsMember() != tt.expected.IsMember() {
				t.Errorf("expected member status %t, got %t", tt.expected.IsMember(), result.IsMember())
			}
		})
	}
}

func TestEmptyModel(t *testing.T) {
	result := EmptyModel()
	
	if result.Id() != 0 {
		t.Errorf("expected ID 0, got %d", result.Id())
	}
	
	if result.Name() != "" {
		t.Errorf("expected empty name, got %s", result.Name())
	}
	
	if result.Rank() != 0 {
		t.Errorf("expected rank 0, got %d", result.Rank())
	}
	
	if result.IsMember() {
		t.Errorf("expected member status false, got true")
	}
}