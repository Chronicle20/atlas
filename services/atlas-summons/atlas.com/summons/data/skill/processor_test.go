package skill

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
)

// skillBody is a faithful JSON:API document for the atlas-data skill endpoint
// (data/skills/{id}). The resource type is "skills" and `effects` is a plain
// JSON array attribute embedded in the skill's `attributes` (NOT a relationship)
// per services/atlas-data/.../data/skill/rest.go. Two effect levels are present
// so GetEffect(skillId, level) can index effects[level-1]. The level-2 effect
// carries every attribute the summon lifecycle reads (hp/duration/x/y/prop,
// weaponAttack/magicAttack, monsterStatus, and a statups entry).
const skillBody = `{
  "data": {
    "type": "skills",
    "id": "1321007",
    "attributes": {
      "name": "Beholder",
      "action": false,
      "element": "",
      "animationTime": 600,
      "maxLevel": 2,
      "effects": [
        {
          "weaponAttack": 10,
          "magicAttack": 0,
          "hp": 100,
          "duration": 60000,
          "x": 1000,
          "y": 0,
          "prop": 0.5,
          "monsterStatus": {},
          "statups": []
        },
        {
          "weaponAttack": 25,
          "magicAttack": 7,
          "hp": 300,
          "duration": 200000,
          "x": 1500,
          "y": 9,
          "prop": 0.8,
          "monsterStatus": {"STUN": 1},
          "statups": [{"type": "WEAPON_DEFENSE", "amount": 60}]
        }
      ]
    }
  }
}`

// TestGetEffect_DecodesEmbeddedEffectArray spins up an httptest server returning
// the skill document above, points the DATA client at it via DATA_SERVICE_URL,
// and asserts the level-2 effect's attributes (including the nested statups
// element) decode to the expected values.
func TestGetEffect_DecodesEmbeddedEffectArray(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(skillBody))
	}))
	defer srv.Close()

	// RootUrl("DATA") reads DATA_SERVICE_URL per call.
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	p := NewProcessor(logrus.New(), context.Background())

	const skillId uint32 = 1321007

	eff, err := p.GetEffect(skillId, 2)
	if err != nil {
		t.Fatalf("GetEffect() error = %v", err)
	}

	if eff.WeaponAttack() != 25 {
		t.Errorf("WeaponAttack() = %d, want 25", eff.WeaponAttack())
	}
	if eff.MagicAttack() != 7 {
		t.Errorf("MagicAttack() = %d, want 7", eff.MagicAttack())
	}
	if eff.Hp() != 300 {
		t.Errorf("Hp() = %d, want 300", eff.Hp())
	}
	if eff.Duration() != 200000 {
		t.Errorf("Duration() = %d, want 200000", eff.Duration())
	}
	if eff.X() != 1500 {
		t.Errorf("X() = %d, want 1500", eff.X())
	}
	if eff.Y() != 9 {
		t.Errorf("Y() = %d, want 9", eff.Y())
	}
	if eff.Prop() != 0.8 {
		t.Errorf("Prop() = %v, want 0.8", eff.Prop())
	}
	if got := eff.MonsterStatus()["STUN"]; got != 1 {
		t.Errorf("MonsterStatus[STUN] = %d, want 1", got)
	}
	statups := eff.Statups()
	if len(statups) != 1 {
		t.Fatalf("len(Statups()) = %d, want 1", len(statups))
	}
	if statups[0].Type != "WEAPON_DEFENSE" || statups[0].Amount != 60 {
		t.Errorf("Statups()[0] = %+v, want {WEAPON_DEFENSE 60}", statups[0])
	}

	wantPath := "/data/skills/1321007"
	if gotPath != wantPath {
		t.Errorf("request path = %q, want %q", gotPath, wantPath)
	}
}

// TestGetEffect_LevelZeroReturnsEmpty asserts the level-0 short-circuit (no
// effect) without hitting the network beyond the initial fetch.
func TestGetEffect_LevelZeroReturnsEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(skillBody))
	}))
	defer srv.Close()

	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	p := NewProcessor(logrus.New(), context.Background())

	eff, err := p.GetEffect(1321007, 0)
	if err != nil {
		t.Fatalf("GetEffect() error = %v", err)
	}
	if eff.Duration() != 0 || eff.Hp() != 0 {
		t.Errorf("level 0 effect = %+v, want zero value", eff)
	}
}
