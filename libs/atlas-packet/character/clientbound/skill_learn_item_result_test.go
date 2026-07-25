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
// Golden bytes, v48 — 15-byte body (NO leading bOnExclRequest byte, same as
// v83; MajorVersion()=48 < 84):
// characterId(4 LE) + isMasteryBook(1) + skillId(4 LE) + masterLevel(4 LE) + canUse(1) + success(1).
//
// IDA evidence (task-125): CWvsContext::OnSkillLearnItemResult @0x71a135 (v48
// IDB GMS_v48_1_DEVM.exe.i64, session 0bb5f11a — already named in the IDB).
// CWvsContext::OnPacket case 43 (0x2B, @0x70d3aa) delegates directly. Body:
// Decode4 characterId (CUserPool::GetUser lookup), then under the
// user-found guard (v29): Decode1 isMasteryBook, Decode4 skillId (decoded,
// discarded), Decode4 masterLevel (decoded, discarded), Decode1 canUse,
// Decode1 success — matches the existing 15-byte golden fixture shape
// exactly. Opcode 0x2B == registry op 43.
//
// packet-audit:verify packet=character/clientbound/CharacterSkillLearnItemResult version=gms_v48 ida=0x71a135
func TestSkillLearnItemResultGoldenBytesV48(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
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

// Golden bytes, v61 — 15-byte body (NO leading bOnExclRequest byte, same as
// v48/v83; MajorVersion()=61 < 84):
// characterId(4 LE) + isMasteryBook(1) + skillId(4 LE) + masterLevel(4 LE) + canUse(1) + success(1).
//
// IDA evidence (task-125): CWvsContext::OnSkillLearnItemResult @0x841e5f (v61
// IDB GMS_v61.1_U_DEVM.exe.i64, session 965202bf). The IDB already carries the
// mangled symbol ?OnSkillLearnItemResult@CWvsContext@@QAEXAAVCInPacket@@@Z at
// this address (func_query confirms it — Hex-Rays merely displayed
// sub_841E5F in the pseudocode; the prior registry note claiming it was
// unnamed was stale, corrected in docs/packets/registry/gms_v61.yaml,
// task-125). CWvsContext::OnPacket case 48 (0x30, @0x8305b4) delegates
// directly. Body: Decode4 characterId (CUserPool::GetUser lookup), then under
// the user-found guard (v28): Decode1 isMasteryBook, Decode4 skillId
// (decoded, discarded), Decode4 masterLevel (decoded, discarded), Decode1
// canUse, Decode1 success — matches the existing 15-byte golden fixture shape
// exactly. Opcode 0x30 == registry op 48.
//
// packet-audit:verify packet=character/clientbound/CharacterSkillLearnItemResult version=gms_v61 ida=0x841e5f
func TestSkillLearnItemResultGoldenBytesV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
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

// Golden bytes, v72 — 15-byte body (NO leading bOnExclRequest byte, same as
// v48/v61/v83; MajorVersion()=72 < 84):
// characterId(4 LE) + isMasteryBook(1) + skillId(4 LE) + masterLevel(4 LE) + canUse(1) + success(1).
//
// IDA evidence (task-125): sub_9175E6 @0x9175e6 (v72 IDB
// GMS_v72.1_U_DEVM.exe.i64, session 90e36cb0). Hex-Rays displays the
// pseudocode under the sub_ label, but xrefs_to the dispatch call site
// resolves this address to the mangled symbol
// CWvsContext::OnSkillLearnItemResult — already named in the IDB (the prior
// registry note claiming it was unnamed was stale, corrected in
// docs/packets/registry/gms_v72.yaml, same class as the v61 correction).
// CWvsContext::OnPacket case 48 (0x30, @0x902791) delegates directly. Body:
// Decode4 characterId (CUserPool::GetUser lookup, @0x917602/0x91760e), then
// under the user-found guard (v28): Decode1 isMasteryBook @0x91764a, Decode4
// skillId (decoded, discarded) @0x91764d, Decode4 masterLevel (decoded,
// discarded) @0x917654, Decode1 canUse @0x917665, Decode1 success @0x917668 —
// matches the existing 15-byte golden fixture shape exactly. Opcode 0x30 ==
// registry op 48.
//
// packet-audit:verify packet=character/clientbound/CharacterSkillLearnItemResult version=gms_v72 ida=0x9175e6
func TestSkillLearnItemResultGoldenBytesV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
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
