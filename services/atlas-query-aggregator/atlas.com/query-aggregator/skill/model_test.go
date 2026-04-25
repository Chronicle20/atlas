package skill_test

import (
	"testing"

	"atlas-query-aggregator/skill"

	"github.com/jtumidanski/api2go/jsonapi"
)

// TestRestModel_UnmarshalSingle guards against the regression where RestModel
// lacked SetID/GetID, causing api2go's jsonapi.Unmarshal to fail with
// "target must implement UnmarshalIdentifier interface". That failure made
// the GetSkillLevel HTTP path always log "skill not found, returning level 0",
// which silently broke the quest selectedSkillID gate (task-023).
func TestRestModel_UnmarshalSingle(t *testing.T) {
	body := []byte(`{
		"data": {
			"type": "skills",
			"id": "4001344",
			"attributes": {
				"level": 5,
				"masterLevel": 0,
				"expiration": "0001-01-01T00:00:00Z",
				"cooldownExpiresAt": "0001-01-01T00:00:00Z"
			}
		}
	}`)

	var rm skill.RestModel
	if err := jsonapi.Unmarshal(body, &rm); err != nil {
		t.Fatalf("jsonapi.Unmarshal failed: %v", err)
	}

	if rm.Id != 4001344 {
		t.Fatalf("expected Id 4001344, got %d", rm.Id)
	}
	if rm.Level != 5 {
		t.Fatalf("expected Level 5, got %d", rm.Level)
	}

	m, err := skill.Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}
	if m.Id() != 4001344 {
		t.Fatalf("expected model Id 4001344, got %d", m.Id())
	}
	if m.Level() != 5 {
		t.Fatalf("expected model Level 5, got %d", m.Level())
	}
}

func TestRestModel_UnmarshalArray(t *testing.T) {
	body := []byte(`{
		"data": [
			{"type":"skills","id":"4001344","attributes":{"level":5,"masterLevel":0,"expiration":"0001-01-01T00:00:00Z","cooldownExpiresAt":"0001-01-01T00:00:00Z"}},
			{"type":"skills","id":"4000001","attributes":{"level":8,"masterLevel":0,"expiration":"0001-01-01T00:00:00Z","cooldownExpiresAt":"0001-01-01T00:00:00Z"}}
		]
	}`)

	var rms []skill.RestModel
	if err := jsonapi.Unmarshal(body, &rms); err != nil {
		t.Fatalf("jsonapi.Unmarshal failed: %v", err)
	}

	if len(rms) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(rms))
	}

	byId := map[uint32]skill.RestModel{}
	for _, r := range rms {
		byId[r.Id] = r
	}
	if byId[4001344].Level != 5 {
		t.Fatalf("expected skill 4001344 level 5, got %d", byId[4001344].Level)
	}
	if byId[4000001].Level != 8 {
		t.Fatalf("expected skill 4000001 level 8, got %d", byId[4000001].Level)
	}
}

func TestRestModel_GetID_RoundTrip(t *testing.T) {
	r := skill.RestModel{Id: 4001344, Level: 5}
	if got := r.GetID(); got != "4001344" {
		t.Fatalf("GetID() = %q, want %q", got, "4001344")
	}

	var dst skill.RestModel
	if err := dst.SetID("4001344"); err != nil {
		t.Fatalf("SetID failed: %v", err)
	}
	if dst.Id != 4001344 {
		t.Fatalf("after SetID, Id = %d, want 4001344", dst.Id)
	}

	if err := dst.SetID("not-a-number"); err == nil {
		t.Fatalf("SetID(\"not-a-number\") should have errored")
	}
}
