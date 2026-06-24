package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// EffectSkillUse byte-fixture — CUser::OnEffect (v83 @0x9377d9) case 1 (skill-use):
//
//	mode    = Decode1            // outer switch discriminator (== 1)   /*0x9377ec*/
//	skillId = Decode4 (skillId)  //                                     /*0x9378d1*/
//	charLvl = Decode1 (v11)      // SLV cap byte stored on self          /*0x9378d9*/
//	skillLvl= Decode1 (nSLV)     //                                     /*0x937902*/
//	// then, conditional on skillId, a trailing flag byte:
//	//   keydown-aura skills (1320006 monster-magnet)  -> Decode1        /*0x93799b*/
//	//   berserk (1121001/1221001/1321001)             -> Decode1        /*0x937b7b*/
//	//   dragon-fury (22160000)                        -> Decode1        /*0x937b0b*/
//
// EffectSkillUse shares the CUser::OnEffect demux with EffectSimple/EffectQuest;
// the EffectQuest op-cell grades worst-of all three, so this sibling carries its
// own v83 marker+fixture+evidence to let the demux promote.
//
// packet-audit:verify packet=character/clientbound/EffectSkillUse version=gms_v83 ida=0x9377d9
func TestEffectSkillUseByteOutput(t *testing.T) {
	v83 := pt.Variants[1] // GMS v83
	ctx := pt.CreateContext(v83.Region, v83.MajorVersion, v83.MinorVersion)

	t.Run("plain/"+v83.Name, func(t *testing.T) {
		// mode=1, skillId=0x010203, characterLevel=0x1E, skillLevel=0x0A, no trailing flags
		input := NewEffectSkillUse(1, 0x010203, 0x1E, 0x0A, false, false, false, false, false, false)
		got := input.Encode(nil, ctx)(nil)
		want := []byte{
			0x01,                   // mode (Decode1)                /*0x9377ec*/
			0x03, 0x02, 0x01, 0x00, // skillId = 0x010203 (Decode4)  /*0x9378d1*/
			0x1e,                   // characterLevel (Decode1 v11)  /*0x9378d9*/
			0x0a,                   // skillLevel (Decode1 nSLV)     /*0x937902*/
		}
		if !bytes.Equal(got, want) {
			t.Errorf("plain bytes:\n got %x\nwant %x", got, want)
		}
	})

	t.Run("berserk/"+v83.Name, func(t *testing.T) {
		// berserk skill 1121001 -> trailing darkForce flag byte
		input := NewEffectSkillUse(1, 1121001, 0x1E, 0x0A, true, true, false, false, false, false)
		got := input.Encode(nil, ctx)(nil)
		want := []byte{
			0x01,                   // mode                          /*0x9377ec*/
			0xe9, 0x1a, 0x11, 0x00, // skillId = 1121001 = 0x111AE9 (Decode4) /*0x9378d1*/
			0x1e,                   // characterLevel                /*0x9378d9*/
			0x0a,                   // skillLevel                    /*0x937902*/
			0x01,                   // berserk darkForce (Decode1)   /*0x937b7b*/
		}
		if !bytes.Equal(got, want) {
			t.Errorf("berserk bytes:\n got %x\nwant %x", got, want)
		}
	})

	t.Run("foreign/"+v83.Name, func(t *testing.T) {
		// characterId prefix (read by CUserPool::OnUserRemotePacket) + skill-use body
		input := NewEffectSkillUseForeign(0x12345678, 1, 0x010203, 0x1E, 0x0A, false, false, false, false, false, false)
		got := input.Encode(nil, ctx)(nil)
		want := []byte{
			0x78, 0x56, 0x34, 0x12, // characterId (foreign prefix)
			0x01,                   // mode
			0x03, 0x02, 0x01, 0x00, // skillId
			0x1e,                   // characterLevel
			0x0a,                   // skillLevel
		}
		if !bytes.Equal(got, want) {
			t.Errorf("foreign bytes:\n got %x\nwant %x", got, want)
		}
	})
}
