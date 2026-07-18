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
// packet-audit:verify packet=character/clientbound/EffectSkillUse version=gms_v95 ida=0x8f9a70
// packet-audit:verify packet=character/clientbound/EffectSkillUse version=jms_v185 ida=0x9f6395
// packet-audit:verify packet=character/clientbound/EffectSkillUse version=gms_v79 ida=0x89112c
// packet-audit:verify packet=character/clientbound/EffectSkillUse version=gms_v72 ida=0x846e1e
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
			0x1e, // characterLevel (Decode1 v11)  /*0x9378d9*/
			0x0a, // skillLevel (Decode1 nSLV)     /*0x937902*/
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
			0x1e, // characterLevel                /*0x9378d9*/
			0x0a, // skillLevel                    /*0x937902*/
			0x01, // berserk darkForce (Decode1)   /*0x937b7b*/
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
			0x1e, // characterLevel
			0x0a, // skillLevel
		}
		if !bytes.Equal(got, want) {
			t.Errorf("foreign bytes:\n got %x\nwant %x", got, want)
		}
	})
}

// EffectSkillUse v79 byte-fixture — CUser::OnEffect (v79 @0x89112c) case 1
// (skill-use). VERSION DELTA: v79 does NOT carry the characterLevel byte that
// v83+ added. IDA-verified read order:
//
//	mode    = Decode1            // outer switch discriminator (== 1)   /*0x89113f*/
//	skillId = Decode4            //                                     /*0x891225*/
//	skillLvl= Decode1 (Value)    // fed to SKILLENTRY::IsActionAppointed /*0x89122f*/
//	// NO characterLevel byte (v83 reads an extra Decode1 here @0x9378d4)
//	// then, conditional on skillId, a trailing flag byte:
//	//   berserk (1121001/1221001/1321001)  -> Decode1                   /*0x891477*/
//	//   monster-magnet (1320006)           -> Decode1                   /*0x8912d9*/
//
// EffectSkillUse shares the CUser::OnEffect demux with EffectSimple/EffectQuest;
// the SHOW_FOREIGN_EFFECT/SHOW_ITEM_GAIN_INCHAT op-cells grade worst-of all three,
// so this sibling carries its own v79 marker+fixture+evidence to let the demux
// promote. The codec version-gates characterLevel (effect_skill_use.go).
func TestEffectSkillUseByteOutputV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)

	t.Run("plain/GMS v79", func(t *testing.T) {
		// mode=1, skillId=0x010203, characterLevel=0x1E (OMITTED on v79 wire), skillLevel=0x0A
		input := NewEffectSkillUse(1, 0x010203, 0x1E, 0x0A, false, false, false, false, false, false)
		got := input.Encode(nil, ctx)(nil)
		want := []byte{
			0x01,                   // mode (Decode1)               /*0x89113f*/
			0x03, 0x02, 0x01, 0x00, // skillId = 0x010203 (Decode4) /*0x891225*/
			0x0a, // skillLevel (Decode1)         /*0x89122f*/  (no characterLevel)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("plain v79 bytes:\n got %x\nwant %x", got, want)
		}
	})

	t.Run("berserk/GMS v79", func(t *testing.T) {
		// berserk skill 1121001 -> trailing darkForce flag byte; still no characterLevel
		input := NewEffectSkillUse(1, 1121001, 0x1E, 0x0A, true, true, false, false, false, false)
		got := input.Encode(nil, ctx)(nil)
		want := []byte{
			0x01,                   // mode                          /*0x89113f*/
			0xe9, 0x1a, 0x11, 0x00, // skillId = 1121001 = 0x111AE9 (Decode4) /*0x891225*/
			0x0a, // skillLevel                    /*0x89122f*/
			0x01, // berserk darkForce (Decode1)   /*0x891477*/
		}
		if !bytes.Equal(got, want) {
			t.Errorf("berserk v79 bytes:\n got %x\nwant %x", got, want)
		}
	})

	t.Run("foreign/GMS v79", func(t *testing.T) {
		// characterId prefix (read by CUserPool::OnUserRemotePacket) + skill-use body (no characterLevel)
		input := NewEffectSkillUseForeign(0x12345678, 1, 0x010203, 0x1E, 0x0A, false, false, false, false, false, false)
		got := input.Encode(nil, ctx)(nil)
		want := []byte{
			0x78, 0x56, 0x34, 0x12, // characterId (foreign prefix)
			0x01,                   // mode
			0x03, 0x02, 0x01, 0x00, // skillId
			0x0a, // skillLevel (no characterLevel)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("foreign v79 bytes:\n got %x\nwant %x", got, want)
		}
	})
}

// EffectSkillUse v72 byte-fixture — CUser::OnEffect (v72 @0x846e1e) case 1
// (skill-use). VERSION DELTA: v72 (GMS < 83) does NOT carry the characterLevel
// byte that v83+ added. IDA-verified read order (v72 IDB GMS_v72.1_U_DEVM.exe):
//
//	mode    = Decode1            // outer switch discriminator (== 1)   /*0x846e31*/
//	skillId = Decode4            // AdditionalLayer = Decode4(v2)        /*0x846f1c*/
//	skillLvl= Decode1 (Value)    // fed to SKILLENTRY::IsActionAppointed /*0x846f30*/
//	// NO characterLevel byte (v83 reads an extra Decode1 here @0x9378d4)
//	// then, conditional on skillId, a trailing flag byte:
//	//   monster-magnet (1320006)           -> Decode1                   /*0x846fc6*/
//	//   berserk (1121001/1221001/1321001)  -> Decode1                   /*0x847167*/
//
// EffectSkillUse shares the CUser::OnEffect demux with EffectSimple/EffectQuest;
// the SHOW_FOREIGN_EFFECT/SHOW_ITEM_GAIN_INCHAT op-cells grade worst-of all three,
// so this sibling carries its own v72 marker+fixture+evidence to let the demux
// promote. The codec version-gates characterLevel (effect_skill_use.go); byte-
// identical to the v79 fixture.
func TestEffectSkillUseByteOutputV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)

	t.Run("plain/GMS v72", func(t *testing.T) {
		// mode=1, skillId=0x010203, characterLevel=0x1E (OMITTED on v72 wire), skillLevel=0x0A
		input := NewEffectSkillUse(1, 0x010203, 0x1E, 0x0A, false, false, false, false, false, false)
		got := input.Encode(nil, ctx)(nil)
		want := []byte{
			0x01,                   // mode (Decode1)               /*0x846e31*/
			0x03, 0x02, 0x01, 0x00, // skillId = 0x010203 (Decode4) /*0x846f1c*/
			0x0a, // skillLevel (Decode1)         /*0x846f30*/  (no characterLevel)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("plain v72 bytes:\n got %x\nwant %x", got, want)
		}
	})

	t.Run("berserk/GMS v72", func(t *testing.T) {
		// berserk skill 1121001 -> trailing darkForce flag byte; still no characterLevel
		input := NewEffectSkillUse(1, 1121001, 0x1E, 0x0A, true, true, false, false, false, false)
		got := input.Encode(nil, ctx)(nil)
		want := []byte{
			0x01,                   // mode                          /*0x846e31*/
			0xe9, 0x1a, 0x11, 0x00, // skillId = 1121001 = 0x111AE9 (Decode4) /*0x846f1c*/
			0x0a, // skillLevel                    /*0x846f30*/
			0x01, // berserk darkForce (Decode1)   /*0x847167*/
		}
		if !bytes.Equal(got, want) {
			t.Errorf("berserk v72 bytes:\n got %x\nwant %x", got, want)
		}
	})

	t.Run("foreign/GMS v72", func(t *testing.T) {
		// characterId prefix (read by CUserPool::OnUserRemotePacket @0x87c050) + skill-use body (no characterLevel)
		input := NewEffectSkillUseForeign(0x12345678, 1, 0x010203, 0x1E, 0x0A, false, false, false, false, false, false)
		got := input.Encode(nil, ctx)(nil)
		want := []byte{
			0x78, 0x56, 0x34, 0x12, // characterId (foreign prefix)
			0x01,                   // mode
			0x03, 0x02, 0x01, 0x00, // skillId
			0x0a, // skillLevel (no characterLevel)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("foreign v72 bytes:\n got %x\nwant %x", got, want)
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
			0x1e, // characterLevel (Decode1 v10)  /*0x96eb8c*/
			0x0a, // skillLevel (Decode1 Value)    /*0x96ebb5*/
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
			0x1e, // characterLevel                /*0x96eb8c*/
			0x0a, // skillLevel                    /*0x96ebb5*/
			0x01, // berserk darkForce (Decode1)   /*0x96eee9*/
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
			0x1e, // characterLevel
			0x0a, // skillLevel
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
			0x1e, // characterLevel (Decode1 v9)   /*0x9b1ff0*/
			0x0a, // skillLevel (Decode1 Value)    /*0x9b2019*/
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
			0x1e, // characterLevel                /*0x9b1ff0*/
			0x0a, // skillLevel                    /*0x9b2019*/
			0x01, // berserk darkForce (Decode1)   /*0x9b2344*/
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
			0x1e, // characterLevel
			0x0a, // skillLevel
		}
		if !bytes.Equal(got, want) {
			t.Errorf("foreign v87 bytes:\n got %x\nwant %x", got, want)
		}
	})
}

// EffectSkillUse v95 byte-fixture — CUser::OnEffect (v95 @0x8f9a70) case 1
// (skill-use). The skill-use arm is case 1 in v95 (unchanged; only the quest arm
// shifted 3->5). The read order is byte-identical to v83/v84/v87:
//
//	mode    = Decode1            // outer switch discriminator (== 1)   /*0x8f9ab4*/
//	skillId = Decode4 (v8)       //                                     /*0x8f9b8f*/
//	charLvl = Decode1 (nCharLevel)// SLV cap byte stored on self          /*0x8f9b98*/
//	skillLvl= Decode1 (sItemName) //                                     /*0x8f9bb8*/
//	// then, conditional on skillId, a trailing flag byte:
//	//   monster-magnet (1320006)  -> Decode1 (LoadDarkForceEffect)       /*case 1320006*/
//	//   dragon-fury (22160000)    -> Decode1                             /*case 22160000*/
//	//   berserk / unregistered    -> Decode1 (is_unregisterd_skill path) /*0x8fa788*/
//
// In v95 the berserk-family trailing flag is read via the is_unregisterd_skill(v8)
// branch (Decode1 -> ShowSkillEffect), structurally equivalent to the v83/v87
// literal-skillId berserk arms; the wire byte is identical. EffectSkillUse shares
// the CUser::OnEffect demux with EffectSimple/EffectQuest; the EffectQuest op-cell
// grades worst-of all three, so this sibling carries its own v95 marker+fixture+
// evidence to let the demux promote.
func TestEffectSkillUseByteOutputV95(t *testing.T) {
	v95 := pt.Variants[3] // GMS v95
	ctx := pt.CreateContext(v95.Region, v95.MajorVersion, v95.MinorVersion)

	t.Run("plain/"+v95.Name, func(t *testing.T) {
		// mode=1, skillId=0x010203, characterLevel=0x1E, skillLevel=0x0A, no trailing flags
		input := NewEffectSkillUse(1, 0x010203, 0x1E, 0x0A, false, false, false, false, false, false)
		got := input.Encode(nil, ctx)(nil)
		want := []byte{
			0x01,                   // mode (Decode1)                /*0x8f9ab4*/
			0x03, 0x02, 0x01, 0x00, // skillId = 0x010203 (Decode4)  /*0x8f9b8f*/
			0x1e, // characterLevel (Decode1)      /*0x8f9b98*/
			0x0a, // skillLevel (Decode1)          /*0x8f9bb8*/
		}
		if !bytes.Equal(got, want) {
			t.Errorf("plain v95 bytes:\n got %x\nwant %x", got, want)
		}
	})

	t.Run("berserk/"+v95.Name, func(t *testing.T) {
		// berserk skill 1121001 -> trailing darkForce flag byte (is_unregisterd_skill path)
		input := NewEffectSkillUse(1, 1121001, 0x1E, 0x0A, true, true, false, false, false, false)
		got := input.Encode(nil, ctx)(nil)
		want := []byte{
			0x01,                   // mode                          /*0x8f9ab4*/
			0xe9, 0x1a, 0x11, 0x00, // skillId = 1121001 = 0x111AE9 (Decode4) /*0x8f9b8f*/
			0x1e, // characterLevel                /*0x8f9b98*/
			0x0a, // skillLevel                    /*0x8f9bb8*/
			0x01, // berserk darkForce (Decode1)   /*0x8fa788*/
		}
		if !bytes.Equal(got, want) {
			t.Errorf("berserk v95 bytes:\n got %x\nwant %x", got, want)
		}
	})

	t.Run("foreign/"+v95.Name, func(t *testing.T) {
		// characterId prefix (read by CUserPool::OnUserRemotePacket) + skill-use body
		input := NewEffectSkillUseForeign(0x12345678, 1, 0x010203, 0x1E, 0x0A, false, false, false, false, false, false)
		got := input.Encode(nil, ctx)(nil)
		want := []byte{
			0x78, 0x56, 0x34, 0x12, // characterId (foreign prefix)
			0x01,                   // mode
			0x03, 0x02, 0x01, 0x00, // skillId
			0x1e, // characterLevel
			0x0a, // skillLevel
		}
		if !bytes.Equal(got, want) {
			t.Errorf("foreign v95 bytes:\n got %x\nwant %x", got, want)
		}
	})
}

// EffectSkillUse jms byte-fixture — CUser::OnEffect (jms v185 @0x9f6395,
// MapleStory_dump_SCY.exe) case 1 (skill-use, block head @0x9f6487). The skill-use
// arm is case 1 in jms (unchanged from GMS; only the v95 quest arm shifted 3->5).
// The read order is byte-identical to v83/v84/v87:
//
//	mode    = Decode1            // outer switch discriminator (== 1)   /*0x9f63c0*/
//	skillId = Decode4 (a2)       //                                     /*0x9f6480*/
//	charLvl = Decode1 (v9)       // SLV cap byte stored on self          /*0x9f648a*/
//	skillLvl= Decode1 (Value)    //                                     /*0x9f64a7*/
//	// then, conditional on skillId, a trailing flag byte:
//	//   monster-magnet (1320006)                      -> Decode1        /*0x9f661d*/
//	//   dragon-fury (22160000)                        -> Decode1        /*0x9f6793*/
//	//   berserk (1121001/1221001/1321001 / job/1e7==9)-> Decode1        /*0x9f68b6*/
//
// jms uses the same literal-skillId berserk arms (a2==1121001||1221001||1321001||
// a2/10000000==9 -> Decode1 -> ShowSkillEffect) as v83/v87; the wire byte is
// identical. EffectSkillUse shares the CUser::OnEffect demux with EffectSimple/
// EffectQuest; the EffectQuest op-cell grades worst-of all three, so this sibling
// carries its own jms marker+fixture+evidence to let the demux promote.
func TestEffectSkillUseByteOutputJMS(t *testing.T) {
	jms := pt.Variants[4] // JMS v185
	ctx := pt.CreateContext(jms.Region, jms.MajorVersion, jms.MinorVersion)

	t.Run("plain/"+jms.Name, func(t *testing.T) {
		// mode=1, skillId=0x010203, characterLevel=0x1E, skillLevel=0x0A, no trailing flags
		input := NewEffectSkillUse(1, 0x010203, 0x1E, 0x0A, false, false, false, false, false, false)
		got := input.Encode(nil, ctx)(nil)
		want := []byte{
			0x01,                   // mode (Decode1)                /*0x9f63c0*/
			0x03, 0x02, 0x01, 0x00, // skillId = 0x010203 (Decode4)  /*0x9f6480*/
			0x1e, // characterLevel (Decode1 v9)   /*0x9f648a*/
			0x0a, // skillLevel (Decode1 Value)    /*0x9f64a7*/
		}
		if !bytes.Equal(got, want) {
			t.Errorf("plain jms bytes:\n got %x\nwant %x", got, want)
		}
	})

	t.Run("berserk/"+jms.Name, func(t *testing.T) {
		// berserk skill 1121001 -> trailing darkForce flag byte
		input := NewEffectSkillUse(1, 1121001, 0x1E, 0x0A, true, true, false, false, false, false)
		got := input.Encode(nil, ctx)(nil)
		want := []byte{
			0x01,                   // mode                          /*0x9f63c0*/
			0xe9, 0x1a, 0x11, 0x00, // skillId = 1121001 = 0x111AE9 (Decode4) /*0x9f6480*/
			0x1e, // characterLevel                /*0x9f648a*/
			0x0a, // skillLevel                    /*0x9f64a7*/
			0x01, // berserk darkForce (Decode1)   /*0x9f68b6*/
		}
		if !bytes.Equal(got, want) {
			t.Errorf("berserk jms bytes:\n got %x\nwant %x", got, want)
		}
	})

	t.Run("foreign/"+jms.Name, func(t *testing.T) {
		// characterId prefix (read by CUserPool::OnUserRemotePacket) + skill-use body
		input := NewEffectSkillUseForeign(0x12345678, 1, 0x010203, 0x1E, 0x0A, false, false, false, false, false, false)
		got := input.Encode(nil, ctx)(nil)
		want := []byte{
			0x78, 0x56, 0x34, 0x12, // characterId (foreign prefix)
			0x01,                   // mode
			0x03, 0x02, 0x01, 0x00, // skillId
			0x1e, // characterLevel
			0x0a, // skillLevel
		}
		if !bytes.Equal(got, want) {
			t.Errorf("foreign jms bytes:\n got %x\nwant %x", got, want)
		}
	})
}
