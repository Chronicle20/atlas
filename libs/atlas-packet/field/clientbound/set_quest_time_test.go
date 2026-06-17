package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldSetQuestTime version=gms_v83 ida=0x5378cd
// packet-audit:verify packet=field/clientbound/FieldSetQuestTime version=gms_v84 ida=0x543bcb
// packet-audit:verify packet=field/clientbound/FieldSetQuestTime version=gms_v87 ida=0x55f242
// packet-audit:verify packet=field/clientbound/FieldSetQuestTime version=gms_v95 ida=0x52b790
// packet-audit:verify packet=field/clientbound/FieldSetQuestTime version=jms_v185 ida=0x574b06
func TestSetQuestTimeGolden(t *testing.T) {
	input := NewSetQuestTime([]QuestTime{NewQuestTime(0x000004D2, 0x0000000000000001, 0x00000000000000FF)})
	ctx := test.CreateContext("GMS", 83, 1)
	expected := []byte{
		0x01,                                           // count
		0xD2, 0x04, 0x00, 0x00,                         // questId 1234
		0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // startTime
		0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // endTime
	}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestSetQuestTimeRoundTrip(t *testing.T) {
	input := NewSetQuestTime([]QuestTime{
		NewQuestTime(0x000004D2, 0x0000000000000001, 0x00000000000000FF),
		NewQuestTime(0x00000929, 0xDEADBEEF12345678, 0xCAFEBABE87654321),
	})
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
