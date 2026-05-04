package inventory

import (
	"testing"

	"github.com/jtumidanski/api2go/jsonapi"
)

// TestCompartmentRestModel_HydratesIncludedAssets feeds a JSON:API document
// with one compartment whose single asset is denormalised in `included`,
// and asserts that every numeric attribute on the asset is populated after
// jsonapi.Unmarshal. This is the regression net for the bug where stub
// assets had Slot==0/Hp==0/Mp==0 and were silently skipped by IsEquipped().
func TestCompartmentRestModel_HydratesIncludedAssets(t *testing.T) {
	doc := []byte(`{
      "data": {
        "type": "compartments",
        "id": "00000000-0000-0000-0000-000000000001",
        "attributes": {"type": 1, "capacity": 24},
        "relationships": {
          "assets": { "data": [{"type": "assets", "id": "42"}] }
        }
      },
      "included": [{
        "type": "assets",
        "id": "42",
        "attributes": {
          "slot": -49,
          "templateId": 1142107,
          "expiration": "0001-01-01T00:00:00Z",
          "ownerId": 0,
          "strength": 1,
          "dexterity": 2,
          "intelligence": 3,
          "luck": 4,
          "hp": 47,
          "mp": 50,
          "weaponAttack": 5,
          "magicAttack": 6,
          "weaponDefense": 7,
          "magicDefense": 8,
          "accuracy": 9,
          "avoidability": 10,
          "hands": 11,
          "speed": 12,
          "jump": 13
        }
      }]
    }`)

	var c CompartmentRestModel
	if err := jsonapi.Unmarshal(doc, &c); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if len(c.Assets) != 1 {
		t.Fatalf("len(Assets) = %d, want 1", len(c.Assets))
	}
	a := c.Assets[0]
	if a.Id != 42 {
		t.Errorf("Id = %d, want 42", a.Id)
	}
	if a.Slot != -49 {
		t.Errorf("Slot = %d, want -49", a.Slot)
	}
	if a.TemplateId != 1142107 {
		t.Errorf("TemplateId = %d, want 1142107", a.TemplateId)
	}
	if a.Hp != 47 {
		t.Errorf("Hp = %d, want 47", a.Hp)
	}
	if a.Mp != 50 {
		t.Errorf("Mp = %d, want 50", a.Mp)
	}
	if a.Strength != 1 {
		t.Errorf("Strength = %d, want 1", a.Strength)
	}
	if a.Dexterity != 2 {
		t.Errorf("Dexterity = %d, want 2", a.Dexterity)
	}
	if a.Intelligence != 3 {
		t.Errorf("Intelligence = %d, want 3", a.Intelligence)
	}
	if a.Luck != 4 {
		t.Errorf("Luck = %d, want 4", a.Luck)
	}
	if a.WeaponAttack != 5 {
		t.Errorf("WeaponAttack = %d, want 5", a.WeaponAttack)
	}
	if a.MagicAttack != 6 {
		t.Errorf("MagicAttack = %d, want 6", a.MagicAttack)
	}
	if a.WeaponDefense != 7 {
		t.Errorf("WeaponDefense = %d, want 7", a.WeaponDefense)
	}
	if a.MagicDefense != 8 {
		t.Errorf("MagicDefense = %d, want 8", a.MagicDefense)
	}
	if a.Accuracy != 9 {
		t.Errorf("Accuracy = %d, want 9", a.Accuracy)
	}
	if a.Avoidability != 10 {
		t.Errorf("Avoidability = %d, want 10", a.Avoidability)
	}
	if a.Hands != 11 {
		t.Errorf("Hands = %d, want 11", a.Hands)
	}
	if a.Speed != 12 {
		t.Errorf("Speed = %d, want 12", a.Speed)
	}
	if a.Jump != 13 {
		t.Errorf("Jump = %d, want 13", a.Jump)
	}

	// Sanity: IsEquipped() must now succeed, confirming the downstream gate
	// in fetchEquipmentBonuses will no longer starve.
	if !a.IsEquipped() {
		t.Errorf("IsEquipped() = false, want true (Slot=%d)", a.Slot)
	}
}
