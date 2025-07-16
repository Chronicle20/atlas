package character

import (
	"testing"
)

func TestNewModel(t *testing.T) {
	tests := []struct {
		name        string
		id          uint32
		charName    string
		level       byte
		wantId      uint32
		wantName    string
		wantLevel   byte
	}{
		{
			name:      "creates_model_with_valid_parameters",
			id:        123,
			charName:  "TestChar",
			level:     50,
			wantId:    123,
			wantName:  "TestChar",
			wantLevel: 50,
		},
		{
			name:      "creates_model_with_zero_id",
			id:        0,
			charName:  "ZeroChar",
			level:     1,
			wantId:    0,
			wantName:  "ZeroChar",
			wantLevel: 1,
		},
		{
			name:      "creates_model_with_empty_name",
			id:        456,
			charName:  "",
			level:     99,
			wantId:    456,
			wantName:  "",
			wantLevel: 99,
		},
		{
			name:      "creates_model_with_max_level",
			id:        789,
			charName:  "MaxLevel",
			level:     255,
			wantId:    789,
			wantName:  "MaxLevel",
			wantLevel: 255,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := NewModel(tt.id, tt.charName, tt.level)
			
			if model.Id() != tt.wantId {
				t.Errorf("Id() = %v, want %v", model.Id(), tt.wantId)
			}
			
			if model.Name() != tt.wantName {
				t.Errorf("Name() = %v, want %v", model.Name(), tt.wantName)
			}
			
			if model.Level() != tt.wantLevel {
				t.Errorf("Level() = %v, want %v", model.Level(), tt.wantLevel)
			}
		})
	}
}

func TestModel_Id(t *testing.T) {
	model := Model{id: 12345}
	
	expected := uint32(12345)
	if model.Id() != expected {
		t.Errorf("Id() = %v, want %v", model.Id(), expected)
	}
}

func TestModel_Name(t *testing.T) {
	model := Model{name: "TestName"}
	
	expected := "TestName"
	if model.Name() != expected {
		t.Errorf("Name() = %v, want %v", model.Name(), expected)
	}
}

func TestModel_Level(t *testing.T) {
	model := Model{level: 75}
	
	expected := byte(75)
	if model.Level() != expected {
		t.Errorf("Level() = %v, want %v", model.Level(), expected)
	}
}