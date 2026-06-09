package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func sampleCreateCharacter() CreateCharacter {
	return CreateCharacter{
		name:             "TestChar",
		jobIndex:         1,
		subJobIndex:      7,
		face:             20000,
		hair:             30000,
		hairColor:        0,
		skinColor:        0,
		topTemplateId:    1040002,
		bottomTemplateId: 1060002,
		shoesTemplateId:  1072001,
		weaponTemplateId: 1302000,
		gender:           0,
		strength:         13,
		dexterity:        4,
		intelligence:     4,
		luck:             4,
	}
}

// TestCreateCharacterVersionBoundary pins the corrected >83 -> >=87 boundary
// (delta §3.1.5): v84..86 must encode byte-identically to v83 (no subJobIndex
// short). v87/v95 stay on the v87+ path.
func TestCreateCharacterVersionBoundary(t *testing.T) {
	m := sampleCreateCharacter()
	encode := func(major uint16) []byte {
		ctx := pt.CreateContext("GMS", major, 1)
		return pt.Encode(t, ctx, m.Encode, nil)
	}
	v83 := encode(83)
	for _, major := range []uint16{84, 85, 86} {
		if got := encode(major); !bytes.Equal(got, v83) {
			t.Errorf("CreateCharacter v%d encode differs from v83 (len %d vs %d); v84..86 must match v83", major, len(got), len(v83))
		}
	}
	if v87 := encode(87); bytes.Equal(v87, v83) {
		t.Errorf("CreateCharacter v87 must stay on the v87+ path, not equal v83")
	}
}

// TestMoveVersionBoundary pins the corrected >83 -> >=87 boundary for the
// self-move dr*/dwKey/crc32 header (delta §3.1.9, MED): v84..86 encode
// byte-identically to v83 (no dr* block). v87/v95 stay on the v87+ path.
func TestMoveVersionBoundary(t *testing.T) {
	m := Move{dr0: 100, dr1: 200, fieldKey: 42, dr2: 300, dr3: 400, crc: 500, dwKey: 600, crc32: 700}
	encode := func(major uint16) []byte {
		ctx := pt.CreateContext("GMS", major, 1)
		return pt.Encode(t, ctx, m.Encode, nil)
	}
	v83 := encode(83)
	for _, major := range []uint16{84, 85, 86} {
		if got := encode(major); !bytes.Equal(got, v83) {
			t.Errorf("Move v%d encode differs from v83 (len %d vs %d); v84..86 must match v83", major, len(got), len(v83))
		}
	}
	if v87 := encode(87); bytes.Equal(v87, v83) {
		t.Errorf("Move v87 must stay on the v87+ path, not equal v83")
	}
	if v95 := encode(95); bytes.Equal(v95, v83) {
		t.Errorf("Move v95 must stay on the v87+ path, not equal v83")
	}
}
