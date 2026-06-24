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
// packet-audit:verify packet=character/clientbound/EffectSkillUse version=gms_v84 ida=0x96ea92
// packet-audit:verify packet=character/clientbound/EffectSkillUse version=gms_v87 ida=0x9b1ef0
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

// EffectSkillUse v84 byte-fixture — CUser::OnEffect (v84 @0x96ea92) case 1
// (skill-use, @0x96eb84). The read order is byte-identical to v83 (v84 body ≡ v83
// below ~0x3D, IDA-confirmed):
//
//	mode    = Decode1            // outer switch discriminator (== 1)   /*0x96eaa5*/
//	skillId = Decode4 (v200)     //                                     /*0x96eb84*/
//	charLvl = Decode1 (v10)      // SLV cap byte stored on self          /*0x96eb8c*/
//	skillLvl= Decode1 (Value)    //                                     /*0x96ebb5*/
//	// then, conditional on skillId, a trailing flag byte:
//	//   monster-magnet (1320006)                      -> Decode1        /*0x96ecfa*/
//	//   dragon-fury (22160000)                        -> Decode1        /*0x96ee70*/
//	//   berserk (1121001/1221001/1321001)             -> Decode1        /*0x96eee9*/
//
// EffectSkillUse shares the CUser::OnEffect demux with EffectSimple/EffectQuest;
// the EffectQuest op-cell grades worst-of all three, so this sibling carries its
// own v84 marker+fixture+evidence to let the demux promote.
func TestEffectSkillUseByteOutputV84(t *testing.T) {
	v84 := pt.Variants[5] // GMS v84
	ctx := pt.CreateContext(v84.Region, v84.MajorVersion, v84.MinorVersion)

	t.Run("plain/"+v84.Name, func(t *testing.T) {
		// mode=1, skillId=0x010203, characterLevel=0x1E, skillLevel=0x0A, no trailing flags
		input := NewEffectSkillUse(1, 0x010203, 0x1E, 0x0A, false, false, false, false, false, false)
		got := input.Encode(nil, ctx)(nil)
		want := []byte{
			0x01,                   // mode (Decode1)                /*0x96eaa5*/
			0x03, 0x02, 0x01, 0x00, // skillId = 0x010203 (Decode4)  /*0x96eb84*/
			0x1e,                   // characterLevel (Decode1 v10)  /*0x96eb8c*/
			0x0a,                   // skillLevel (Decode1 Value)    /*0x96ebb5*/
		}
		if !bytes.Equal(got, want) {
			t.Errorf("plain v84 bytes:\n got %x\nwant %x", got, want)
		}
	})

	t.Run("berserk/"+v84.Name, func(t *testing.T) {
		// berserk skill 1121001 -> trailing darkForce flag byte
		input := NewEffectSkillUse(1, 1121001, 0x1E, 0x0A, true, true, false, false, false, false)
		got := input.Encode(nil, ctx)(nil)
		want := []byte{
			0x01,                   // mode                          /*0x96eaa5*/
			0xe9, 0x1a, 0x11, 0x00, // skillId = 1121001 = 0x111AE9 (Decode4) /*0x96eb84*/
			0x1e,                   // characterLevel                /*0x96eb8c*/
			0x0a,                   // skillLevel                    /*0x96ebb5*/
			0x01,                   // berserk darkForce (Decode1)   /*0x96eee9*/
		}
		if !bytes.Equal(got, want) {
			t.Errorf("berserk v84 bytes:\n got %x\nwant %x", got, want)
		}
	})

	t.Run("foreign/"+v84.Name, func(t *testing.T) {
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
			t.Errorf("foreign v84 bytes:\n got %x\nwant %x", got, want)
		}
	})
}

// EffectSkillUse v87 byte-fixture — CUser::OnEffect (v87 @0x9b1ef0) case 1
// (skill-use, @0x9b1fe8). The read order is byte-identical to v83/v84:
//
//	mode    = Decode1            // outer switch discriminator (== 1)   /*0x9b1f03*/
//	skillId = Decode4 (v193)     //                                     /*0x9b1fe8*/
//	charLvl = Decode1 (v9)       // SLV cap byte stored on self          /*0x9b1ff0*/
//	skillLvl= Decode1 (Value)    //                                     /*0x9b2019*/
//	// then, conditional on skillId, a trailing flag byte:
//	//   monster-magnet (1320006)                      -> Decode1        /*0x9b215f*/
//	//   dragon-fury (22160000)                        -> Decode1        /*0x9b22cf*/
//	//   berserk (1121001/1221001/1321001/.../100==9)  -> Decode1        /*0x9b2344*/
//
// EffectSkillUse shares the CUser::OnEffect demux with EffectSimple/EffectQuest;
// the EffectQuest op-cell grades worst-of all three, so this sibling carries its
// own v87 marker+fixture+evidence to let the demux promote.
func TestEffectSkillUseByteOutputV87(t *testing.T) {
	v87 := pt.Variants[2] // GMS v87
	ctx := pt.CreateContext(v87.Region, v87.MajorVersion, v87.MinorVersion)

	t.Run("plain/"+v87.Name, func(t *testing.T) {
		// mode=1, skillId=0x010203, characterLevel=0x1E, skillLevel=0x0A, no trailing flags
		input := NewEffectSkillUse(1, 0x010203, 0x1E, 0x0A, false, false, false, false, false, false)
		got := input.Encode(nil, ctx)(nil)
		want := []byte{
			0x01,                   // mode (Decode1)                /*0x9b1f03*/
			0x03, 0x02, 0x01, 0x00, // skillId = 0x010203 (Decode4)  /*0x9b1fe8*/
			0x1e,                   // characterLevel (Decode1 v9)   /*0x9b1ff0*/
			0x0a,                   // skillLevel (Decode1 Value)    /*0x9b2019*/
		}
		if !bytes.Equal(got, want) {
			t.Errorf("plain v87 bytes:\n got %x\nwant %x", got, want)
		}
	})

	t.Run("berserk/"+v87.Name, func(t *testing.T) {
		// berserk skill 1121001 -> trailing darkForce flag byte
		input := NewEffectSkillUse(1, 1121001, 0x1E, 0x0A, true, true, false, false, false, false)
		got := input.Encode(nil, ctx)(nil)
		want := []byte{
			0x01,                   // mode                          /*0x9b1f03*/
			0xe9, 0x1a, 0x11, 0x00, // skillId = 1121001 = 0x111AE9 (Decode4) /*0x9b1fe8*/
			0x1e,                   // characterLevel                /*0x9b1ff0*/
			0x0a,                   // skillLevel                    /*0x9b2019*/
			0x01,                   // berserk darkForce (Decode1)   /*0x9b2344*/
		}
		if !bytes.Equal(got, want) {
			t.Errorf("berserk v87 bytes:\n got %x\nwant %x", got, want)
		}
	})

	t.Run("foreign/"+v87.Name, func(t *testing.T) {
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
			t.Errorf("foreign v87 bytes:\n got %x\nwant %x", got, want)
		}
	})
}
