package effectivestats

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/sirupsen/logrus"
)

// statsBody is a faithful JSON:API document for the atlas-effective-stats
// character stats endpoint (worlds/{w}/channels/{c}/characters/{id}/stats). The
// resource type is "effective-stats" and the id is the character id as a string
// (services/atlas-effective-stats/.../stat/rest.go). Only the attributes the
// summon damage ceiling consumes are asserted; extra upstream attributes are
// included to prove they are ignored without breaking the decode.
const statsBody = `{
  "data": {
    "type": "effective-stats",
    "id": "12345",
    "attributes": {
      "strength": 120,
      "dexterity": 45,
      "luck": 30,
      "intelligence": 15,
      "maxHP": 4000,
      "maxMP": 2000,
      "weaponAttack": 95,
      "weaponDefense": 200,
      "magicAttack": 60,
      "magicDefense": 180,
      "accuracy": 150,
      "avoidability": 40,
      "speed": 130,
      "jump": 120
    }
  }
}`

// TestGetByCharacter_DecodesStats spins up an httptest server returning the stats
// document above, points the EFFECTIVE_STATS client at it via
// EFFECTIVE_STATS_SERVICE_URL, and asserts every combat stat the summon damage
// ceiling reads decodes to the expected value.
func TestGetByCharacter_DecodesStats(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(statsBody))
	}))
	defer srv.Close()

	// RootUrl("EFFECTIVE_STATS") reads EFFECTIVE_STATS_SERVICE_URL per call.
	t.Setenv("EFFECTIVE_STATS_SERVICE_URL", srv.URL+"/")

	p := NewProcessor(logrus.New(), context.Background())

	const worldId world.Id = 0
	const channelId channel.Id = 1
	const characterId uint32 = 12345

	m, err := p.GetByCharacter(worldId, channelId, characterId)
	if err != nil {
		t.Fatalf("GetByCharacter() error = %v", err)
	}

	if m.Strength() != 120 {
		t.Errorf("Strength() = %d, want 120", m.Strength())
	}
	if m.Dexterity() != 45 {
		t.Errorf("Dexterity() = %d, want 45", m.Dexterity())
	}
	if m.Luck() != 30 {
		t.Errorf("Luck() = %d, want 30", m.Luck())
	}
	if m.Intelligence() != 15 {
		t.Errorf("Intelligence() = %d, want 15", m.Intelligence())
	}
	if m.WeaponAttack() != 95 {
		t.Errorf("WeaponAttack() = %d, want 95", m.WeaponAttack())
	}
	if m.MagicAttack() != 60 {
		t.Errorf("MagicAttack() = %d, want 60", m.MagicAttack())
	}

	wantPath := "/worlds/0/channels/1/characters/12345/stats"
	if gotPath != wantPath {
		t.Errorf("request path = %q, want %q", gotPath, wantPath)
	}
}
