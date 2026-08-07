package effect

import (
	"encoding/json"
	"testing"
)

// TestCommonKeyJSONTags pins the JSON attribute name of every field added for
// a Skill.wz `common` key. The tag is the wz key verbatim (design §5.1) so
// atlas-ui and any other consumer can address it by its archive name.
func TestCommonKeyJSONTags(t *testing.T) {
	m := NewModelBuilder().
		SetRange(1).SetMastery(2).SetZ(3).SetDot(4).SetCr(5).
		SetDotInterval(6).SetDotTime(7).SetDamR(8).SetCriticaldamageMin(9).
		SetMHPRRate(10).SetV(11).SetIgnoreMobpdpR(12).SetEpad(13).SetW(14).
		SetU(15).SetEpdd(16).SetEmdd(17).SetSelfDestruction(18).SetAsrR(19).
		SetMMPRRate(20).SetT(21).SetEr(22).SetPddR(23).SetTerR(24).
		SetMadX(25).SetSubProp(26).SetEmhp(27).SetCriticaldamageMax(28).
		SetExpR(29).SetEmmp(30).SetConsumeItemId(31).SetMddR(32).
		SetSubTime(33).SetPadX(34).SetMesoR(35).
		Build()

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform error = %v", err)
	}

	b, err := json.Marshal(rm)
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	want := map[string]float64{
		"range": 1, "mastery": 2, "z": 3, "dot": 4, "cr": 5,
		"dotInterval": 6, "dotTime": 7, "damR": 8, "criticaldamageMin": 9,
		"MHPRRate": 10, "v": 11, "ignoreMobpdpR": 12, "epad": 13, "w": 14,
		"u": 15, "epdd": 16, "emdd": 17, "selfDestruction": 18, "asrR": 19,
		"MMPRRate": 20, "t": 21, "er": 22, "pddR": 23, "terR": 24,
		"madX": 25, "subProp": 26, "emhp": 27, "criticaldamageMax": 28,
		"expR": 29, "emmp": 30, "consumeItemId": 31, "mddR": 32,
		"subTime": 33, "padX": 34, "mesoR": 35,
	}
	for key, w := range want {
		v, ok := got[key]
		if !ok {
			t.Fatalf("marshalled effect has no %q attribute", key)
		}
		if v != w {
			t.Fatalf("attribute %q = %v, want %v", key, v, w)
		}
	}
}

// TestExistingItemConsumeUnchanged pins FR-6.4: the pre-existing
// `itemConsume` attribute is fed by wz `itemCon` and must NOT be repurposed
// for wz `common/itemConsume`, which lands on `consumeItemId`.
func TestExistingItemConsumeUnchanged(t *testing.T) {
	m := NewModelBuilder().SetItemConsume(2000000).SetConsumeItemId(2331000).Build()
	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform error = %v", err)
	}
	if rm.ItemConsume != 2000000 {
		t.Fatalf("ItemConsume = %d, want 2000000", rm.ItemConsume)
	}
	if rm.ConsumeItemId != 2331000 {
		t.Fatalf("ConsumeItemId = %d, want 2331000", rm.ConsumeItemId)
	}
}
