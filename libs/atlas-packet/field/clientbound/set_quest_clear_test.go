package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldSetQuestClear version=gms_v83 ida=0x5378ba
// packet-audit:verify packet=field/clientbound/FieldSetQuestClear version=gms_v84 ida=0x543bb8
// packet-audit:verify packet=field/clientbound/FieldSetQuestClear version=gms_v87 ida=0x55f22f
// packet-audit:verify packet=field/clientbound/FieldSetQuestClear version=gms_v95 ida=0x52c870
// packet-audit:verify packet=field/clientbound/FieldSetQuestClear version=jms_v185 ida=0x574af3
func TestSetQuestClearGolden(t *testing.T) {
	input := NewSetQuestClear()
	ctx := test.CreateContext("GMS", 83, 1)
	actual := test.Encode(t, ctx, input.Encode, nil)
	if len(actual) != 0 {
		t.Errorf("golden mismatch: got %v want empty", actual)
	}
}

// TestSetQuestClearByteOutputV48 pins the gms_v48 SET_QUEST_CLEAR clientbound wire.
// IDA: sub_4CBC9A @0x4cbc9a (GMS_v48_1_DEVM.exe) reads NOTHING — returns
// sub_4CC659(global+184), structurally identical to v61 CField::OnSetQuestClear
// @0x4ef90b (empty, QuestMan helper). NOTE: the v48 registry mislabeled SET_QUEST_CLEAR
// onto op 93 (sub_4CBB78, an 8-byte-FILETIME per-quest-timer packet); body-verification
// corrected it to op 94 (sub_4CBC9A). Empty wire matches the version-invariant golden.
// packet-audit:verify packet=field/clientbound/FieldSetQuestClear version=gms_v48 ida=0x4cbc9a
func TestSetQuestClearByteOutputV48(t *testing.T) {
	input := NewSetQuestClear()
	ctx := test.CreateContext("GMS", 48, 1)
	actual := test.Encode(t, ctx, input.Encode, nil)
	if len(actual) != 0 {
		t.Errorf("v48 golden mismatch: got %v want empty", actual)
	}
}

func TestSetQuestClearRoundTrip(t *testing.T) {
	input := NewSetQuestClear()
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
