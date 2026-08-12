package model

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

// TestMagnetByteLayoutPerVersion is the per-version regression fixture for the
// Monster Magnet arm of the use-skill opcode. Each row's expected byte count and
// field order was derived by reading that version's
// CUserLocal::TryDoingMonsterMagnet packet-build tail instruction-by-instruction
// (addresses below; four of the ten were unnamed in the IDB and were renamed to
// CUserLocal__TryDoingMonsterMagnet there during derivation).
//
// This deliberately carries NO `packet-audit:verify` marker and NO pinned
// evidence record: the coverage matrix's unit of promotion is the whole
// SPECIAL_MOVE op cell, which carries sixteen fnames of which this is one, and a
// marker would either orphan (failing `matrix --check`) or promote a cell whose
// other fifteen fnames are unverified. See
// docs/tasks/task-215-monster-magnet/context.md section 2.
//
// The load-bearing assertion is `left == 0`: a layout that is wrong in any
// width, order, or presence/absence of a trailing field leaves the reader short
// of or past the end of the body.
func TestMagnetByteLayoutPerVersion(t *testing.T) {
	grabs := []MagnetGrab{
		NewMagnetGrab(1001, true),
		NewMagnetGrab(1002, false),
	}

	tests := []struct {
		name        string
		region      string
		major       uint16
		minor       uint16
		skillId     skill.Id
		legacy      bool
		ida         string
		wantGrabs   int
		wantDelay   uint16
		wantLeft    bool
		wantByteLen int
	}{
		{
			name: "gms_v48_legacy", region: "GMS", major: 48, minor: 1,
			skillId: skill.HeroMonsterMagnetId, legacy: true,
			// CUserLocal::TryDoingMonsterMagnet @0x6AD842; COutPacket ctor
			// `push 46h` @0x6ADABC (opcode 0x46, matching template_gms_48_1.json's
			// CharacterUseSkillHandle). updateTime Encode4 @0x6ADAD3, skillId
			// Encode4 @0x6ADAE0, skillLevel Encode1 @0x6ADAEB, entryCount Encode1
			// @0x6ADB02 (ONE byte), per-entry Encode4 @0x6ADB1B (NO result byte),
			// delay Encode2 @0x6ADB29 (NO direction byte). entry[0] is the caster:
			// ZArray<ulong>::InsertBefore @0x6AD977-0x6AD987 pushes [esi+0x654]
			// (esi = CUserLocal) before the mob loop @0x6ADA89-0x6ADA99 appends
			// [mob+0x654].
			ida: "0x6AD842", wantGrabs: 2, wantDelay: 750, wantLeft: false,
			// 4+4+1 + 1 + 4*(1 caster + 2 mobs) + 2 = 24
			wantByteLen: 24,
		},
		{
			name: "gms_v61_modern", region: "GMS", major: 61, minor: 1,
			skillId: skill.HeroMonsterMagnetId, legacy: false,
			// CUserLocal::TryDoingMonsterMagnet @0x7B9684; COutPacket(83) = 0x53,
			// matching template_gms_61_1.json's CharacterUseSkillHandle.
			ida: "0x7B9684", wantGrabs: 2, wantDelay: 0, wantLeft: true,
			// 4+4+1 + 4 + 5*2 + 1 = 24
			wantByteLen: 24,
		},
		{
			name: "gms_v72_modern", region: "GMS", major: 72, minor: 1,
			skillId: skill.PaladinMonsterMagnetId, legacy: false,
			// @0x876605; encodes @0x876A2B/38/43/4E, loop @0x876A86,0x876A95,
			// tail (direction) @0x876AB2. Opcode 0x5A.
			ida: "0x876605", wantGrabs: 2, wantDelay: 0, wantLeft: true,
			wantByteLen: 24,
		},
		{
			name: "gms_v79_modern", region: "GMS", major: 79, minor: 1,
			skillId: skill.PaladinMonsterMagnetId, legacy: false,
			// @0x8C3117; encodes @0x8C3540/4D/58/63, loop @0x8C359B,0x8C35AA,
			// tail @0x8C35C7. Opcode 0x59.
			ida: "0x8C3117", wantGrabs: 2, wantDelay: 0, wantLeft: false,
			wantByteLen: 24,
		},
		{
			name: "gms_v83_modern", region: "GMS", major: 83, minor: 1,
			skillId: skill.DarkKnightMonsterMagnetId, legacy: false,
			// @0x96C215; `COutPacket::Encode1(v65, *v40 == 3)` is the per-entry
			// grab bool. Opcode 0x5B.
			ida: "0x96C215", wantGrabs: 2, wantDelay: 0, wantLeft: true,
			wantByteLen: 24,
		},
		{
			name: "gms_v84_modern", region: "GMS", major: 84, minor: 1,
			skillId: skill.DarkKnightMonsterMagnetId, legacy: false,
			// @0x9ABDB7; encodes @0x9AC1F3/200/20B/216, loop @0x9AC24E,0x9AC25D,
			// tail @0x9AC27D. Opcode 0x5B.
			ida: "0x9ABDB7", wantGrabs: 2, wantDelay: 0, wantLeft: false,
			wantByteLen: 24,
		},
		{
			name: "gms_v87_modern", region: "GMS", major: 87, minor: 1,
			skillId: skill.HeroMonsterMagnetId, legacy: false,
			// @0x9F086F; encodes @0x9F0CAB/CB8/CC3/CCE, loop @0x9F0D06,0x9F0D15,
			// tail @0x9F0D35. Opcode 0x5E.
			ida: "0x9F086F", wantGrabs: 2, wantDelay: 0, wantLeft: true,
			wantByteLen: 24,
		},
		{
			name: "gms_v92_modern", region: "GMS", major: 92, minor: 1,
			skillId: skill.HeroMonsterMagnetId, legacy: false,
			// @0x91F2A0; encodes @0x91FA54/60/71/7F, loop @0x91FABD,0x91FAE1,
			// tail @0x91FAFB. Opcode 0x66.
			ida: "0x91F2A0", wantGrabs: 2, wantDelay: 0, wantLeft: false,
			wantByteLen: 24,
		},
		{
			name: "gms_v95_modern", region: "GMS", major: 95, minor: 1,
			// PaladinMonsterMagnetId (1221001) is absent from
			// libs/atlas-constants/gen/wzsnapshot/gms_95_1.json (it lists
			// 1221000/1221002/1221004/1221007/1221009/1221011/1221012 but not
			// 1221001), so version_gms_95_1_gen.go has no 1221001 mapping and
			// isMonsterMagnet returns false for a Paladin magnet cast at v95 —
			// an upstream wzsnapshot data gap, not a decoder fact (also hit and
			// independently confirmed while deriving Task 1's fixtures). Using
			// DarkKnightMonsterMagnetId instead, which IS mapped at v95, keeps
			// this row's point without depending on that gap.
			skillId: skill.DarkKnightMonsterMagnetId, legacy: false,
			// @0x940570; encodes @0x940D25/31/42/50, loop @0x940D8D,0x940DB1,
			// tail @0x940DCB. Opcode 0x67.
			ida: "0x940570", wantGrabs: 2, wantDelay: 0, wantLeft: true,
			wantByteLen: 24,
		},
		{
			name: "jms_v185_modern", region: "JMS", major: 185, minor: 1,
			skillId: skill.DarkKnightMonsterMagnetId, legacy: false,
			// @0xA3C61C; encodes @0xA3CC52/5C/67/72, loop @0xA3CCAA,0xA3CCC6,
			// tail @0xA3CCE3. Opcode 0x56. jms takes the MODERN branch.
			ida: "0xA3C61C", wantGrabs: 2, wantDelay: 0, wantLeft: false,
			wantByteLen: 24,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf []byte
			if tc.legacy {
				buf = legacyMagnetBody(tc.skillId, 30, 900001, []uint32{1001, 1002}, tc.wantDelay)
			} else {
				buf = modernMagnetBody(tc.skillId, 30, grabs, tc.wantLeft)
			}
			if len(buf) != tc.wantByteLen {
				t.Fatalf("fixture is %d bytes, want %d (%s)", len(buf), tc.wantByteLen, tc.ida)
			}

			m, left := decodeMagnetBody(t, tc.region, tc.major, tc.minor, buf)
			if left != 0 {
				t.Fatalf("%s: reader has %d unconsumed bytes — layout wrong (%s)", tc.name, left, tc.ida)
			}
			if len(m.MagnetGrabs()) != tc.wantGrabs {
				t.Fatalf("%s: MagnetGrabs len = %d, want %d (%s)",
					tc.name, len(m.MagnetGrabs()), tc.wantGrabs, tc.ida)
			}
			if m.MagnetGrabs()[0].ObjectId() != 1001 || m.MagnetGrabs()[1].ObjectId() != 1002 {
				t.Fatalf("%s: object ids = [%d %d], want [1001 1002] (%s)",
					tc.name, m.MagnetGrabs()[0].ObjectId(), m.MagnetGrabs()[1].ObjectId(), tc.ida)
			}
			if m.Delay() != tc.wantDelay {
				t.Fatalf("%s: Delay = %d, want %d (%s)", tc.name, m.Delay(), tc.wantDelay, tc.ida)
			}
			if m.Direction() != tc.wantLeft {
				t.Fatalf("%s: Direction = %v, want %v (%s)", tc.name, m.Direction(), tc.wantLeft, tc.ida)
			}

			// Per-entry grab results: the modern shape carries a bool per entry;
			// the legacy shape has none and every surviving entry is a grab.
			if tc.legacy {
				for i, g := range m.MagnetGrabs() {
					if !g.Grabbed() {
						t.Fatalf("%s: grab[%d] not grabbed; v48 sends no per-entry result (%s)", tc.name, i, tc.ida)
					}
				}
			} else {
				if !m.MagnetGrabs()[0].Grabbed() || m.MagnetGrabs()[1].Grabbed() {
					t.Fatalf("%s: grab results = [%v %v], want [true false] (%s)",
						tc.name, m.MagnetGrabs()[0].Grabbed(), m.MagnetGrabs()[1].Grabbed(), tc.ida)
				}
			}
		})
	}
}
