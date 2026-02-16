package party_quest

import (
	"errors"
	"testing"
)

func TestCompareUint32(t *testing.T) {
	tests := []struct {
		name     string
		actual   uint32
		operator string
		expected uint32
		want     bool
	}{
		{"eq true", 5, "eq", 5, true},
		{"eq false", 5, "eq", 6, false},
		{"gte equal", 5, "gte", 5, true},
		{"gte greater", 6, "gte", 5, true},
		{"gte less", 4, "gte", 5, false},
		{"lte equal", 5, "lte", 5, true},
		{"lte less", 4, "lte", 5, true},
		{"lte greater", 6, "lte", 5, false},
		{"gt true", 6, "gt", 5, true},
		{"gt equal", 5, "gt", 5, false},
		{"gt less", 4, "gt", 5, false},
		{"lt true", 4, "lt", 5, true},
		{"lt equal", 5, "lt", 5, false},
		{"lt greater", 6, "lt", 5, false},
		{"unknown operator", 5, "invalid", 5, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compareUint32(tt.actual, tt.operator, tt.expected)
			if got != tt.want {
				t.Errorf("compareUint32(%d, %q, %d) = %v, want %v", tt.actual, tt.operator, tt.expected, got, tt.want)
			}
		})
	}
}

func TestValidateStartRequirements_Empty(t *testing.T) {
	members := []MemberRestModel{{Id: 1, Level: 30}}
	err := validateStartRequirements(nil, members)
	if err != nil {
		t.Errorf("expected nil for empty requirements, got %v", err)
	}
	err = validateStartRequirements([]ConditionRestModel{}, members)
	if err != nil {
		t.Errorf("expected nil for empty slice requirements, got %v", err)
	}
}

func TestValidateStartRequirements_PartySizePass(t *testing.T) {
	reqs := []ConditionRestModel{{Type: "party_size", Operator: "gte", Value: 3}}
	members := []MemberRestModel{{Id: 1}, {Id: 2}, {Id: 3}}
	if err := validateStartRequirements(reqs, members); err != nil {
		t.Errorf("expected pass, got %v", err)
	}
}

func TestValidateStartRequirements_PartySizeFail(t *testing.T) {
	reqs := []ConditionRestModel{{Type: "party_size", Operator: "gte", Value: 3}}
	members := []MemberRestModel{{Id: 1}, {Id: 2}}
	err := validateStartRequirements(reqs, members)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var pqErr PartyQuestError
	if !errors.As(err, &pqErr) {
		t.Fatalf("expected PartyQuestError, got %T", err)
	}
	if pqErr.Code != ErrorCodePartySizeFailed {
		t.Errorf("expected code %s, got %s", ErrorCodePartySizeFailed, pqErr.Code)
	}
}

func TestValidateStartRequirements_PartySizeEq(t *testing.T) {
	reqs := []ConditionRestModel{{Type: "party_size", Operator: "eq", Value: 6}}
	pass := []MemberRestModel{{Id: 1}, {Id: 2}, {Id: 3}, {Id: 4}, {Id: 5}, {Id: 6}}
	if err := validateStartRequirements(reqs, pass); err != nil {
		t.Errorf("expected pass with 6 members, got %v", err)
	}
	fail := []MemberRestModel{{Id: 1}, {Id: 2}, {Id: 3}, {Id: 4}, {Id: 5}}
	err := validateStartRequirements(reqs, fail)
	if err == nil {
		t.Fatal("expected error with 5 members, got nil")
	}
	var pqErr PartyQuestError
	if !errors.As(err, &pqErr) || pqErr.Code != ErrorCodePartySizeFailed {
		t.Errorf("expected PQ_PARTY_SIZE, got %v", err)
	}
}

func TestValidateStartRequirements_LevelMinPass(t *testing.T) {
	reqs := []ConditionRestModel{{Type: "level_min", Operator: "gte", Value: 35}}
	members := []MemberRestModel{{Id: 1, Level: 35}, {Id: 2, Level: 50}, {Id: 3, Level: 40}}
	if err := validateStartRequirements(reqs, members); err != nil {
		t.Errorf("expected pass, got %v", err)
	}
}

func TestValidateStartRequirements_LevelMinFail(t *testing.T) {
	reqs := []ConditionRestModel{{Type: "level_min", Operator: "gte", Value: 35}}
	members := []MemberRestModel{{Id: 1, Level: 40}, {Id: 2, Level: 20}, {Id: 3, Level: 50}}
	err := validateStartRequirements(reqs, members)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var pqErr PartyQuestError
	if !errors.As(err, &pqErr) || pqErr.Code != ErrorCodeLevelMinFailed {
		t.Errorf("expected PQ_LEVEL_MIN, got %v", err)
	}
}

func TestValidateStartRequirements_LevelMaxPass(t *testing.T) {
	reqs := []ConditionRestModel{{Type: "level_max", Operator: "lte", Value: 50}}
	members := []MemberRestModel{{Id: 1, Level: 35}, {Id: 2, Level: 50}, {Id: 3, Level: 40}}
	if err := validateStartRequirements(reqs, members); err != nil {
		t.Errorf("expected pass, got %v", err)
	}
}

func TestValidateStartRequirements_LevelMaxFail(t *testing.T) {
	reqs := []ConditionRestModel{{Type: "level_max", Operator: "lte", Value: 50}}
	members := []MemberRestModel{{Id: 1, Level: 40}, {Id: 2, Level: 55}, {Id: 3, Level: 45}}
	err := validateStartRequirements(reqs, members)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var pqErr PartyQuestError
	if !errors.As(err, &pqErr) || pqErr.Code != ErrorCodeLevelMaxFailed {
		t.Errorf("expected PQ_LEVEL_MAX, got %v", err)
	}
}

func TestValidateStartRequirements_SkipsUnknownTypes(t *testing.T) {
	reqs := []ConditionRestModel{
		{Type: "item_count", Operator: "gte", Value: 10, ReferenceId: 4001101},
		{Type: "monster_kill", Operator: "gte", Value: 5, ReferenceId: 100100},
		{Type: "some_future_type", Operator: "eq", Value: 1},
	}
	members := []MemberRestModel{{Id: 1, Level: 30}}
	if err := validateStartRequirements(reqs, members); err != nil {
		t.Errorf("expected unknown types to be skipped, got %v", err)
	}
}

func TestValidateStartRequirements_MultipleConditions(t *testing.T) {
	reqs := []ConditionRestModel{
		{Type: "party_size", Operator: "gte", Value: 3},
		{Type: "level_min", Operator: "gte", Value: 10},
		{Type: "level_max", Operator: "lte", Value: 50},
	}

	t.Run("all pass", func(t *testing.T) {
		members := []MemberRestModel{
			{Id: 1, Level: 15},
			{Id: 2, Level: 30},
			{Id: 3, Level: 45},
		}
		if err := validateStartRequirements(reqs, members); err != nil {
			t.Errorf("expected pass, got %v", err)
		}
	})

	t.Run("party size fails", func(t *testing.T) {
		members := []MemberRestModel{
			{Id: 1, Level: 30},
			{Id: 2, Level: 40},
		}
		err := validateStartRequirements(reqs, members)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var pqErr PartyQuestError
		if !errors.As(err, &pqErr) || pqErr.Code != ErrorCodePartySizeFailed {
			t.Errorf("expected PQ_PARTY_SIZE, got %v", err)
		}
	})

	t.Run("level min fails", func(t *testing.T) {
		members := []MemberRestModel{
			{Id: 1, Level: 5},
			{Id: 2, Level: 30},
			{Id: 3, Level: 40},
		}
		err := validateStartRequirements(reqs, members)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var pqErr PartyQuestError
		if !errors.As(err, &pqErr) || pqErr.Code != ErrorCodeLevelMinFailed {
			t.Errorf("expected PQ_LEVEL_MIN, got %v", err)
		}
	})

	t.Run("level max fails", func(t *testing.T) {
		members := []MemberRestModel{
			{Id: 1, Level: 30},
			{Id: 2, Level: 40},
			{Id: 3, Level: 55},
		}
		err := validateStartRequirements(reqs, members)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var pqErr PartyQuestError
		if !errors.As(err, &pqErr) || pqErr.Code != ErrorCodeLevelMaxFailed {
			t.Errorf("expected PQ_LEVEL_MAX, got %v", err)
		}
	})
}

func TestGetErrorCode_PartyQuestErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"not in party", PartyQuestError{Code: ErrorCodeNotInParty, Message: "test"}, ErrorCodeNotInParty},
		{"not leader", PartyQuestError{Code: ErrorCodeNotLeader, Message: "test"}, ErrorCodeNotLeader},
		{"party size", PartyQuestError{Code: ErrorCodePartySizeFailed, Message: "test"}, ErrorCodePartySizeFailed},
		{"level min", PartyQuestError{Code: ErrorCodeLevelMinFailed, Message: "test"}, ErrorCodeLevelMinFailed},
		{"level max", PartyQuestError{Code: ErrorCodeLevelMaxFailed, Message: "test"}, ErrorCodeLevelMaxFailed},
		{"definition not found", PartyQuestError{Code: ErrorCodeDefinitionNotFound, Message: "test"}, ErrorCodeDefinitionNotFound},
		{"generic error", errors.New("something failed"), "PQ_UNKNOWN"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetErrorCode(tt.err)
			if got != tt.want {
				t.Errorf("GetErrorCode() = %q, want %q", got, tt.want)
			}
		})
	}
}
