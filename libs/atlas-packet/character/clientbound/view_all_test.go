package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/clientbound/CharacterViewAllCharacters version=gms_v83 ida=0x5facca
// packet-audit:verify packet=character/clientbound/CharacterViewAllCount version=gms_v83 ida=0x5facca
// packet-audit:verify packet=character/clientbound/CharacterViewAllSearchFailed version=gms_v83 ida=0x5facca
// packet-audit:verify packet=character/clientbound/CharacterViewAllCharacters version=gms_v87 ida=0x6328eb
// packet-audit:verify packet=character/clientbound/CharacterViewAllCount version=gms_v87 ida=0x6328eb
// packet-audit:verify packet=character/clientbound/CharacterViewAllSearchFailed version=gms_v87 ida=0x6328eb
// packet-audit:verify packet=character/clientbound/CharacterViewAllCharacters version=gms_v95 ida=0x5de435
// packet-audit:verify packet=character/clientbound/CharacterViewAllCount version=gms_v95 ida=0x5de17f
// packet-audit:verify packet=character/clientbound/CharacterViewAllSearchFailed version=gms_v95 ida=0x5de284
// CharacterViewAllError is the error/notice case of the same CLogin::OnViewAllCharResult
// dispatcher (addr identical to the SearchFailed slice per version); the decompose
// emitted no distinct #CharacterViewAllError export slice, so its evidence is pinned
// against the same-function #CharacterViewAllSearchFailed slice (same address/hash).
// packet-audit:verify packet=character/clientbound/CharacterViewAllError version=gms_v83 ida=0x5facca
// packet-audit:verify packet=character/clientbound/CharacterViewAllError version=gms_v87 ida=0x6328eb
// packet-audit:verify packet=character/clientbound/CharacterViewAllError version=gms_v95 ida=0x5de284
// packet-audit:verify packet=character/clientbound/CharacterViewAllCount version=gms_v84 ida=0x60ffe8
// packet-audit:verify packet=character/clientbound/CharacterViewAllCharacters version=gms_v84 ida=0x60ffe8
// packet-audit:verify packet=character/clientbound/CharacterViewAllSearchFailed version=gms_v84 ida=0x60ffe8
// packet-audit:verify packet=character/clientbound/CharacterViewAllError version=gms_v84 ida=0x60ffe8
// jms VIEW_ALL_CHAR result is dispatched by CLogin::OnViewAllCharResult@0x6709e4 on a
// leading Decode1(mode): mode 0 = NORMAL (nWorldID + nCount, then per-char
// GW_CharacterStat::Decode@0x50ec17 + AvatarLook::Decode@0x51517e + rank block); mode 1 =
// CHARACTER_COUNT (Decode4 svrCount + Decode4 charCount); modes 2/3/4/5 = error (code byte
// only). The jms GW_CharacterStat block differs from v83/v84 (nAP widened to int32, jms tail);
// the Atlas CharacterListEntry jms branch (verified by the 8c CharacterList byte-fixture) emits
// it exactly. The base + #suffix export keys were spliced from the live jms decompile (the 3
// #suffix entries were ABSENT on jms). No codec delta — verification-only.
// packet-audit:verify packet=character/clientbound/CharacterViewAllCount version=jms_v185 ida=0x6709e4
// packet-audit:verify packet=character/clientbound/CharacterViewAllCharacters version=jms_v185 ida=0x6709e4
// packet-audit:verify packet=character/clientbound/CharacterViewAllSearchFailed version=jms_v185 ida=0x6709e4
// packet-audit:verify packet=character/clientbound/CharacterViewAllError version=jms_v185 ida=0x6709e4
// v48 VIEW_ALL_CHAR result dispatcher — sub_50232D @0x50232d (GMS_v48_1_DEVM.exe,
// port 13337), keyed on a leading Decode1(mode):
//   mode 0 (case 0u @0x50257d): Decode1(worldId) + Decode1(count) + per-char
//     GW_CharacterStat::Decode (sub_49B627) + AvatarLook::Decode (sub_49E1E0) +
//     Decode1(rankFlag)?DecodeBuffer(16):memset  → CharacterViewAllCharacters.
//     No trailing PIC byte (v48 < 87). Stat/avatar use the v48 single-pet legacy
//     shape (see TestCharacterListByteOutputV48).
//   mode 1 (case 1u @0x502380): Decode4(worldCount) + Decode4(unk)
//     → CharacterViewAllCount.
//   modes 2/4/5 (@0x50240b / @0x50249d): reset/notice arms that read NOTHING beyond
//     the mode byte → the code-only CharacterViewAllError (mode 2) /
//     CharacterViewAllSearchFailed. (modes 3/6/7 additionally read Decode1(hasMsg)+
//     optional DecodeStr — a client notice variant Atlas does not model/emit.)
// packet-audit:verify packet=character/clientbound/CharacterViewAllCharacters version=gms_v48 ida=0x50232d
// packet-audit:verify packet=character/clientbound/CharacterViewAllCount version=gms_v48 ida=0x50232d
// packet-audit:verify packet=character/clientbound/CharacterViewAllSearchFailed version=gms_v48 ida=0x50232d
// packet-audit:verify packet=character/clientbound/CharacterViewAllError version=gms_v48 ida=0x50232d
func TestCharacterViewAllByteOutputV48(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)

	t.Run("Count", func(t *testing.T) {
		got := NewCharacterViewAllCount(3, 5, 0).Encode(nil, ctx)(nil)
		want := []byte{
			0x03,                   // code/mode (Decode1)          /*0x502363*/
			0x05, 0x00, 0x00, 0x00, // worldCount (Decode4)         /*0x502380*/
			0x00, 0x00, 0x00, 0x00, // unk (Decode4)                /*0x502397*/
		}
		if !bytes.Equal(got, want) {
			t.Errorf("Count v48:\n got %x\nwant %x", got, want)
		}
	})

	t.Run("Characters", func(t *testing.T) {
		stats := model.NewCharacterStatistics(
			0x01020304, "Hero", 0, 0, 0x4D2, 0x7B, [3]uint64{0, 0, 0},
			0x0A, 0x64, 4, 5, 6, 7, 0x64, 0x64, 0x32, 0x32, 3, false, 2, 0, 8, 0, 0x0BB8, 0,
		)
		avatar := model.NewAvatar(0, 0, 0x4D2, false, 0x7B, nil, nil, nil)
		entry := model.NewCharacterListEntry(stats, avatar, true /*viewAll*/, false /*gm*/, 1, 2, 3, 4)
		got := NewCharacterViewAllCharacters(0, world.Id(0), []model.CharacterListEntry{entry}).Encode(nil, ctx)(nil)

		want := []byte{
			0x00, // code/mode (Decode1)                            /*0x502363*/
			0x00, // worldId (Decode1)                              /*0x50257d*/
			0x01, // count = 1 (Decode1)                            /*0x50258d*/

			// --- GW_CharacterStat block --- sub_49B627 @0x49b627
			0x04, 0x03, 0x02, 0x01, // id
			0x48, 0x65, 0x72, 0x6f, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // "Hero"+pad13
			0x00,                   // gender
			0x00,                   // skin
			0xd2, 0x04, 0x00, 0x00, // face
			0x7b, 0x00, 0x00, 0x00, // hair
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // SINGLE pet long
			0x0a,       // level
			0x64, 0x00, // jobId
			0x04, 0x00, // str
			0x05, 0x00, // dex
			0x06, 0x00, // int
			0x07, 0x00, // luck
			0x64, 0x00, // hp
			0x64, 0x00, // maxHp
			0x32, 0x00, // mp
			0x32, 0x00, // maxMp
			0x03, 0x00, // ap
			0x02, 0x00, // sp
			0x00, 0x00, 0x00, 0x00, // exp
			0x08, 0x00,             // fame
			0xb8, 0x0b, 0x00, 0x00, // mapId
			0x00,                   // spawnPoint

			// --- AvatarLook block --- sub_49E1E0 @0x49e1e0
			0x00,                   // gender
			0x00,                   // skin
			0xd2, 0x04, 0x00, 0x00, // face
			0x01,                   // !mega
			0x7b, 0x00, 0x00, 0x00, // hair
			0xff,                   // equip terminator
			0xff,                   // masked terminator
			0x00, 0x00, 0x00, 0x00, // cash weapon
			0x00, 0x00, 0x00, 0x00, // SINGLE pet int

			// --- entry trailer (viewAll=true: no family byte) ---
			0x01,                   // rankEnabled = !gm (Decode1)  /*0x5025ef*/
			0x01, 0x00, 0x00, 0x00, // rank                         /*0x50260a DecodeBuffer 16*/
			0x02, 0x00, 0x00, 0x00, // rankMove
			0x03, 0x00, 0x00, 0x00, // jobRank
			0x04, 0x00, 0x00, 0x00, // jobRankMove
			// --- NO trailing PIC byte: v48 < 87 ---
		}
		if !bytes.Equal(got, want) {
			t.Errorf("Characters v48:\n got %x\nwant %x", got, want)
		}
	})

	t.Run("SearchFailed", func(t *testing.T) {
		got := NewCharacterViewAllSearchFailed(4).Encode(nil, ctx)(nil)
		if len(got) != 1 || got[0] != 4 {
			t.Errorf("SearchFailed v48: got % x, want [04]", got)
		}
	})

	t.Run("Error", func(t *testing.T) {
		got := NewCharacterViewAllError(2).Encode(nil, ctx)(nil)
		if len(got) != 1 || got[0] != 2 {
			t.Errorf("Error v48: got % x, want [02]", got)
		}
	})
}

func TestCharacterViewAllCountRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := CharacterViewAllCount{code: 3, worldCount: 5, unk: 0}
			output := CharacterViewAllCount{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Code() != input.Code() {
				t.Errorf("code: got %v, want %v", output.Code(), input.Code())
			}
			if output.WorldCount() != input.WorldCount() {
				t.Errorf("worldCount: got %v, want %v", output.WorldCount(), input.WorldCount())
			}
		})
	}
}

func TestCharacterViewAllCharactersRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			stats := model.NewCharacterStatistics(
				99, "ViewAllChar", 0, 2, 20000, 30000,
				[3]uint64{10, 20, 30},
				40, 100,
				30, 25, 20, 15,
				1000, 1000, 500, 500,
				3, false, 2,
				50000, 50, 1000,
				100000000, 0,
			)
			avatar := model.NewAvatar(0, 2, 20000, false, 30000, nil, nil, nil)
			// viewAll=true: no family byte; gm=false: rank fields are written
			entry := model.NewCharacterListEntry(stats, avatar, true, false, 5, 1, 3, 2)
			input := NewCharacterViewAllCharacters(0, world.Id(0), []model.CharacterListEntry{entry})
			output := CharacterViewAllCharacters{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Code() != input.Code() {
				t.Errorf("code: got %v, want %v", output.Code(), input.Code())
			}
			if output.WorldId() != input.WorldId() {
				t.Errorf("worldId: got %v, want %v", output.WorldId(), input.WorldId())
			}
			if len(output.Characters()) != len(input.Characters()) {
				t.Errorf("characters len: got %v, want %v", len(output.Characters()), len(input.Characters()))
			}
		})
	}
}

func TestCharacterViewAllSearchFailedRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := CharacterViewAllSearchFailed{code: 4}
			output := CharacterViewAllSearchFailed{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Code() != input.Code() {
				t.Errorf("code: got %v, want %v", output.Code(), input.Code())
			}
		})
	}
}

// TestCharacterViewAllErrorByteOutput pins the CharacterViewAllError wire body.
// CLogin::OnViewAllCharResult is a server→client dispatcher keyed on a leading
// Decode1(mode/code). The error/notice modes resolve as:
//
//	v83 @0x5facca, v87 @0x6328eb, v95 @0x5de284 (case 2/3/6/7 block).
//	case 2 (RemoveNoticeConnecting+ResetVAC, StringPool 0xFBE): NO further reads
//	       — body is the single code byte. This is the path the code-only
//	       CharacterViewAllError struct models, identical in shape to its
//	       already-verified sibling CharacterViewAllSearchFailed.
//
// NOTE (coverage boundary, derived from the decompile, not papered over): the
// case 3/6/7 path at the report address additionally reads Decode1(hasMsg) and,
// if set, DecodeStr(msg) (v83 @0x5fadd5/0x5fade4; v95 @0x5de292/0x5de2a2).
// Atlas's code-only struct does NOT model that flag+string variant and Atlas
// never emits it (the struct is unused by services; the code byte is the
// dispatcher selector). The single-byte fixture below is the exact, faithful
// wire for the mode-2 error path the struct represents.
//
// EVIDENCE: the IDA exports harvested a `#CharacterViewAllSearchFailed` slice at
// this address but NOT a distinct `#CharacterViewAllError` slice. Since the two
// cases share one CLogin::OnViewAllCharResult function (identical address/hash
// per version), CharacterViewAllError's evidence is pinned against the
// same-function SearchFailed slice — see the packet-audit:verify markers at the
// top of this file (v83 0x5facca / v87 0x6328eb / v95 0x5de284). This test
// provides the fresh byte-fixture backing those tier-1 cells.
func TestCharacterViewAllErrorByteOutput(t *testing.T) {
	for _, v := range []struct {
		Name         string
		Region       string
		Major, Minor uint16
	}{
		{"GMS v83", "GMS", 83, 1},
		{"GMS v87", "GMS", 87, 1},
		{"GMS v95", "GMS", 95, 1},
	} {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.Major, v.Minor)
			got := pt.Encode(t, ctx, NewCharacterViewAllError(2).Encode, nil)
			// body = code byte (the dispatcher mode selector). No further reads
			// on the mode-2 path.
			if len(got) != 1 || got[0] != 2 {
				t.Errorf("%s: got % x, want [02]", v.Name, got)
			}
		})
	}
}

func TestCharacterViewAllErrorRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := CharacterViewAllError{code: 5}
			output := CharacterViewAllError{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Code() != input.Code() {
				t.Errorf("code: got %v, want %v", output.Code(), input.Code())
			}
		})
	}
}
