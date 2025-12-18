package guild

import (
	"atlas-query-aggregator/guild/member"
	"atlas-query-aggregator/guild/title"
	"testing"
)

func TestExtract(t *testing.T) {
	tests := []struct {
		name     string
		restModel RestModel
		expected Model
	}{
		{
			name: "valid guild model",
			restModel: RestModel{
				Id:       123,
				WorldId:  1,
				Name:     "TestGuild",
				Notice:   "Welcome to TestGuild",
				Points:   1000,
				Capacity: 100,
				Logo:     1,
				LogoColor: 1,
				LogoBackground: 1,
				LogoBackgroundColor: 1,
				LeaderId: 456,
				Members:  []member.RestModel{},
				Titles:   []title.RestModel{},
			},
			expected: Model{
				id:                  123,
				worldId:             1,
				name:                "TestGuild",
				notice:              "Welcome to TestGuild",
				points:              1000,
				capacity:            100,
				logo:                1,
				logoColor:           1,
				logoBackground:      1,
				logoBackgroundColor: 1,
				leaderId:            456,
				members:             []member.Model{},
				titles:              []title.Model{},
			},
		},
		{
			name: "empty guild model",
			restModel: RestModel{
				Id:       0,
				WorldId:  0,
				Name:     "",
				Notice:   "",
				Points:   0,
				Capacity: 0,
				Logo:     0,
				LogoColor: 0,
				LogoBackground: 0,
				LogoBackgroundColor: 0,
				LeaderId: 0,
				Members:  []member.RestModel{},
				Titles:   []title.RestModel{},
			},
			expected: Model{
				id:                  0,
				worldId:             0,
				name:                "",
				notice:              "",
				points:              0,
				capacity:            0,
				logo:                0,
				logoColor:           0,
				logoBackground:      0,
				logoBackgroundColor: 0,
				leaderId:            0,
				members:             []member.Model{},
				titles:              []title.Model{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Extract(tt.restModel)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			
			if result.Id() != tt.expected.Id() {
				t.Errorf("expected ID %d, got %d", tt.expected.Id(), result.Id())
			}
			
			if result.Name() != tt.expected.Name() {
				t.Errorf("expected name %s, got %s", tt.expected.Name(), result.Name())
			}
			
			if result.Notice() != tt.expected.Notice() {
				t.Errorf("expected notice %s, got %s", tt.expected.Notice(), result.Notice())
			}
			
			if result.Points() != tt.expected.Points() {
				t.Errorf("expected points %d, got %d", tt.expected.Points(), result.Points())
			}
			
			if result.Capacity() != tt.expected.Capacity() {
				t.Errorf("expected capacity %d, got %d", tt.expected.Capacity(), result.Capacity())
			}
			
			if result.LeaderId() != tt.expected.LeaderId() {
				t.Errorf("expected leader ID %d, got %d", tt.expected.LeaderId(), result.LeaderId())
			}
		})
	}
}

func TestMemberRank(t *testing.T) {
	// Create a test guild with members
	members := []member.Model{
		{}, // This would need to be properly constructed with actual member data
	}
	
	guild := Model{
		id:      123,
		name:    "TestGuild",
		members: members,
	}
	
	// Test MemberRank function
	rank := guild.MemberRank(456)
	if rank != 0 {
		t.Errorf("expected rank 0 for non-member, got %d", rank)
	}
}