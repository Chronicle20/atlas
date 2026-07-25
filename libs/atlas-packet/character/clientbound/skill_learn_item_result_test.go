package clientbound

import (
	"bytes"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestSkillLearnItemResultRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewSkillLearnItemResult(12345, true, 1121001, 20, true, false)
			output := SkillLearnItemResult{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.CharacterId() != input.CharacterId() {
				t.Errorf("characterId: got %v, want %v", output.CharacterId(), input.CharacterId())
			}
			if output.IsMasteryBook() != input.IsMasteryBook() {
				t.Errorf("isMasteryBook: got %v, want %v", output.IsMasteryBook(), input.IsMasteryBook())
			}
			if output.SkillId() != input.SkillId() {
				t.Errorf("skillId: got %v, want %v", output.SkillId(), input.SkillId())
			}
			if output.MasterLevel() != input.MasterLevel() {
				t.Errorf("masterLevel: got %v, want %v", output.MasterLevel(), input.MasterLevel())
			}
			if output.CanUse() != input.CanUse() {
				t.Errorf("canUse: got %v, want %v", output.CanUse(), input.CanUse())
			}
			if output.Success() != input.Success() {
				t.Errorf("success: got %v, want %v", output.Success(), input.Success())
			}
		})
	}
}

// Golden bytes, v83 — 15-byte body (NO leading bOnExclRequest byte):
// characterId(4 LE) + isMasteryBook(1) + skillId(4 LE) + masterLevel(4 LE) + canUse(1) + success(1).
// Trivially-readable values: characterId=1, mastery, skillId=2, masterLevel=3, canUse=1, success=0.
//
// packet-audit:verify packet=character/clientbound/CharacterSkillLearnItemResult version=gms_v83 ida=0xa1e5af
func TestSkillLearnItemResultGoldenBytesV83(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	l, _ := testlog.NewNullLogger()
	got := NewSkillLearnItemResult(1, true, 2, 3, true, false).Encode(l, ctx)(nil)
	want := []byte{
		0x01, 0x00, 0x00, 0x00, // characterId
		0x01,                   // isMasteryBook
		0x02, 0x00, 0x00, 0x00, // skillId
		0x03, 0x00, 0x00, 0x00, // masterLevel
		0x01, // canUse
		0x00, // success
	}
	if !bytes.Equal(got, want) {
		t.Errorf("golden bytes: got % X, want % X", got, want)
	}
}

// Golden bytes, v84 — 16-byte body (LEADING bOnExclRequest byte = 0x01).
// Proves the MajorVersion() >= 84 gate. Same field values as the v83 golden,
// so the only difference is the extra leading 0x01. (v84 clientbound diverges
// from v83 despite identical serverbound — the v84≠v83 exception.)
func TestSkillLearnItemResultGoldenBytesV84(t *testing.T) {
	ctx := pt.CreateContext("GMS", 84, 1)
	l, _ := testlog.NewNullLogger()
	got := NewSkillLearnItemResult(1, true, 2, 3, true, false).Encode(l, ctx)(nil)
	want := []byte{
		0x01,                   // bOnExclRequest (v84+ leading byte)
		0x01, 0x00, 0x00, 0x00, // characterId
		0x01,                   // isMasteryBook
		0x02, 0x00, 0x00, 0x00, // skillId
		0x03, 0x00, 0x00, 0x00, // masterLevel
		0x01, // canUse
		0x00, // success
	}
	if !bytes.Equal(got, want) {
		t.Errorf("golden bytes: got % X, want % X", got, want)
	}
}
