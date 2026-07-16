package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// EffectQuest / EffectQuestForeign byte-fixture.
//
// Client read order — CUser::OnEffect (v83 @0x9377d9), the SHOW_FOREIGN_EFFECT /
// SHOW_ITEM_GAIN_INCHAT opcodes both dispatch to this single function whose first
// decoded byte is the effect-mode discriminator (switch on Decode1). The quest /
// item-gain effect is case 3 (0x9377ef):
//
//	mode   = Decode1            // outer switch discriminator (== 3 here) /*0x9377ec*/
//	count  = Decode1 (v36)      // number of item rewards                  /*0x937bf6*/
//	if count == 0:              // /*0x937c00*/ -> /*0x937dbe*/ path
//	    message = DecodeStr     // ZXString length-prefixed                /*0x937dbe*/
//	    nEffect = Decode4       // quest-effect id -> Effect_Quest         /*0x937dfe*/
//	else:                       // /*0x937c12*/ loop body, count times
//	    itemId = Decode4        //                                        /*0x937c12*/
//	    amount = Decode4 (v37)  // signed: >1 gained, ==1, ==-1, < -1     /*0x937c23*/
//
// The foreign (SHOW_FOREIGN_EFFECT) opcode carries a leading characterId int that
// CUserPool::OnUserRemotePacket consumes before dispatch into OnEffect (audit report
// row 0 "characterId — read ... before dispatch (foreign path)"); EffectQuestForeign
// prepends it. The remaining body is identical to the self form.
//
// packet-audit:verify packet=character/clientbound/EffectQuest version=gms_v83 ida=0x9377d9
// packet-audit:verify packet=character/clientbound/EffectQuest version=gms_v84 ida=0x96ea92
// packet-audit:verify packet=character/clientbound/EffectQuest version=gms_v87 ida=0x9b1ef0
// packet-audit:verify packet=character/clientbound/EffectQuest version=gms_v95 ida=0x8f9a70
// packet-audit:verify packet=character/clientbound/EffectQuest version=gms_v79 ida=0x89112c
func TestEffectQuestByteOutput(t *testing.T) {
	v83 := pt.Variants[1] // GMS v83
	ctx := pt.CreateContext(v83.Region, v83.MajorVersion, v83.MinorVersion)

	t.Run("self/no-rewards/"+v83.Name, func(t *testing.T) {
		// mode=3, count=0, message="Hello", nEffect=0x10
		input := NewEffectQuest(3, "Hello", 0x10, nil)
		got := input.Encode(nil, ctx)(nil)
		want := []byte{
			0x03,                   // mode (Decode1, switch discriminator) /*0x9377ec*/
			0x00,                   // count (Decode1 v36)                  /*0x937bf6*/
			0x05, 0x00,             // message len = 5 (DecodeStr short)    /*0x937dbe*/
			0x48, 0x65, 0x6c, 0x6c, 0x6f, // "Hello"                       /*0x937dbe*/
			0x10, 0x00, 0x00, 0x00, // nEffect = 0x10 (Decode4)             /*0x937dfe*/
		}
		if !bytes.Equal(got, want) {
			t.Errorf("self/no-rewards bytes:\n got %x\nwant %x", got, want)
		}
	})

	t.Run("self/rewards/"+v83.Name, func(t *testing.T) {
		// mode=3, two rewards: gain 5 of item 0x010203, lose 1 of item 0x0A
		input := NewEffectQuest(3, "", 0, []QuestReward{
			{ItemId: 0x010203, Amount: 5},
			{ItemId: 0x00000A, Amount: -1},
		})
		got := input.Encode(nil, ctx)(nil)
		want := []byte{
			0x03,                   // mode                                /*0x9377ec*/
			0x02,                   // count = 2 (Decode1 v36)             /*0x937bf6*/
			0x03, 0x02, 0x01, 0x00, // reward0 itemId = 0x010203 (Decode4) /*0x937c12*/
			0x05, 0x00, 0x00, 0x00, // reward0 amount = 5 (Decode4 v37)    /*0x937c23*/
			0x0a, 0x00, 0x00, 0x00, // reward1 itemId = 0x0A (Decode4)     /*0x937c12*/
			0xff, 0xff, 0xff, 0xff, // reward1 amount = -1 (Decode4 v37)   /*0x937c23*/
		}
		if !bytes.Equal(got, want) {
			t.Errorf("self/rewards bytes:\n got %x\nwant %x", got, want)
		}
	})

	t.Run("foreign/no-rewards/"+v83.Name, func(t *testing.T) {
		// characterId=0x12345678, mode=3, count=0, message="Hi", nEffect=7
		input := NewEffectQuestForeign(0x12345678, 3, "Hi", 7, nil)
		got := input.Encode(nil, ctx)(nil)
		want := []byte{
			0x78, 0x56, 0x34, 0x12, // characterId (foreign-path prefix)    /*OnUserRemotePacket*/
			0x03,                   // mode                                /*0x9377ec*/
			0x00,                   // count = 0                           /*0x937bf6*/
			0x02, 0x00,             // message len = 2                     /*0x937dbe*/
			0x48, 0x69,             // "Hi"                                /*0x937dbe*/
			0x07, 0x00, 0x00, 0x00, // nEffect = 7 (Decode4)               /*0x937dfe*/
		}
		if !bytes.Equal(got, want) {
			t.Errorf("foreign/no-rewards bytes:\n got %x\nwant %x", got, want)
		}
	})
}

// EffectQuest / EffectQuestForeign v84 byte-fixture.
//
// Client read order — CUser::OnEffect (v84 @0x96ea92). The SHOW_FOREIGN_EFFECT /
// SHOW_ITEM_GAIN_INCHAT opcodes both dispatch to this single function whose first
// decoded byte is the outer effect-mode discriminator (switch on Decode1 @0x96eaa5).
// The quest / item-gain effect is case 3 (0x96ef60). The read order is byte-identical
// to v83 (v84 body ≡ v83 below ~0x3D, IDA-confirmed):
//
//	mode   = Decode1            // outer switch discriminator (== 3 here) /*0x96eaa5*/
//	count  = Decode1 (v41)      // number of item rewards                  /*0x96ef6c*/
//	if count == 0:              // !v41 path @0x96ef76
//	    message = DecodeStr     // ZXString length-prefixed                /*0x96f134*/
//	    nEffect = Decode4       // quest-effect id                         /*0x96f174*/
//	else:                       // loop @0x96ef88, count times
//	    itemId = Decode4 (v200) //                                        /*0x96ef88*/
//	    amount = Decode4 (v42)  // signed: >1 gained, ==1, ==-1, < -1     /*0x96ef99*/
//
// The foreign (SHOW_FOREIGN_EFFECT) opcode carries a leading characterId int that
// CUserPool::OnUserRemotePacket consumes before dispatch into OnEffect;
// EffectQuestForeign prepends it. The remaining body is identical to the self form.
//
// The wire bytes mirror the v83 fixture exactly.
func TestEffectQuestByteOutputV84(t *testing.T) {
	v84 := pt.Variants[5] // GMS v84
	ctx := pt.CreateContext(v84.Region, v84.MajorVersion, v84.MinorVersion)

	t.Run("self/no-rewards/"+v84.Name, func(t *testing.T) {
		// mode=3, count=0, message="Hello", nEffect=0x10
		input := NewEffectQuest(3, "Hello", 0x10, nil)
		got := input.Encode(nil, ctx)(nil)
		want := []byte{
			0x03,                         // mode (Decode1, switch discriminator) /*0x96eaa5*/
			0x00,                         // count (Decode1 v41)                  /*0x96ef6c*/
			0x05, 0x00,                   // message len = 5 (DecodeStr short)    /*0x96f134*/
			0x48, 0x65, 0x6c, 0x6c, 0x6f, // "Hello"                              /*0x96f134*/
			0x10, 0x00, 0x00, 0x00,       // nEffect = 0x10 (Decode4)             /*0x96f174*/
		}
		if !bytes.Equal(got, want) {
			t.Errorf("self/no-rewards v84 bytes:\n got %x\nwant %x", got, want)
		}
	})

	t.Run("self/rewards/"+v84.Name, func(t *testing.T) {
		// mode=3, two rewards: gain 5 of item 0x010203, lose 1 of item 0x0A
		input := NewEffectQuest(3, "", 0, []QuestReward{
			{ItemId: 0x010203, Amount: 5},
			{ItemId: 0x00000A, Amount: -1},
		})
		got := input.Encode(nil, ctx)(nil)
		want := []byte{
			0x03,                   // mode                                /*0x96eaa5*/
			0x02,                   // count = 2 (Decode1 v41)             /*0x96ef6c*/
			0x03, 0x02, 0x01, 0x00, // reward0 itemId = 0x010203 (Decode4) /*0x96ef88*/
			0x05, 0x00, 0x00, 0x00, // reward0 amount = 5 (Decode4 v42)    /*0x96ef99*/
			0x0a, 0x00, 0x00, 0x00, // reward1 itemId = 0x0A (Decode4)     /*0x96ef88*/
			0xff, 0xff, 0xff, 0xff, // reward1 amount = -1 (Decode4 v42)   /*0x96ef99*/
		}
		if !bytes.Equal(got, want) {
			t.Errorf("self/rewards v84 bytes:\n got %x\nwant %x", got, want)
		}
	})

	t.Run("foreign/no-rewards/"+v84.Name, func(t *testing.T) {
		// characterId=0x12345678, mode=3, count=0, message="Hi", nEffect=7
		input := NewEffectQuestForeign(0x12345678, 3, "Hi", 7, nil)
		got := input.Encode(nil, ctx)(nil)
		want := []byte{
			0x78, 0x56, 0x34, 0x12, // characterId (foreign-path prefix)    /*OnUserRemotePacket*/
			0x03,                   // mode                                /*0x96eaa5*/
			0x00,                   // count = 0                           /*0x96ef6c*/
			0x02, 0x00,             // message len = 2                     /*0x96f134*/
			0x48, 0x69,             // "Hi"                                /*0x96f134*/
			0x07, 0x00, 0x00, 0x00, // nEffect = 7 (Decode4)               /*0x96f174*/
		}
		if !bytes.Equal(got, want) {
			t.Errorf("foreign/no-rewards v84 bytes:\n got %x\nwant %x", got, want)
		}
	})
}

// EffectQuest / EffectQuestForeign v87 byte-fixture.
//
// Client read order — CUser::OnEffect (v87 @0x9b1ef0). The SHOW_FOREIGN_EFFECT /
// SHOW_ITEM_GAIN_INCHAT opcodes both dispatch to this single function whose first
// decoded byte is the outer effect-mode discriminator (switch on Decode1 @0x9b1f03).
// The quest / item-gain effect is case 3 (@0x9b23bc). The read order is byte-identical
// to v83/v84:
//
//	mode   = Decode1            // outer switch discriminator (== 3 here) /*0x9b1f03*/
//	count  = Decode1 (v41)      // number of item rewards                  /*0x9b23c8*/
//	if count == 0:              // !v41 path @0x9b23d2
//	    message = DecodeStr     // ZXString length-prefixed                /*0x9b2557*/
//	    nEffect = Decode4       // quest-effect id                         /*0x9b258e*/
//	else:                       // loop @0x9b23e4, count times
//	    itemId = Decode4 (v193) //                                        /*0x9b23e4*/
//	    amount = Decode4 (v42)  // signed: >1 gained, ==1, ==-1, < -1     /*0x9b23f5*/
//
// The foreign (SHOW_FOREIGN_EFFECT) opcode carries a leading characterId int that
// CUserPool::OnUserRemotePacket consumes before dispatch into OnEffect;
// EffectQuestForeign prepends it. The remaining body is identical to the self form.
//
// The wire bytes mirror the v83/v84 fixtures exactly.
func TestEffectQuestByteOutputV87(t *testing.T) {
	v87 := pt.Variants[2] // GMS v87
	ctx := pt.CreateContext(v87.Region, v87.MajorVersion, v87.MinorVersion)

	t.Run("self/no-rewards/"+v87.Name, func(t *testing.T) {
		// mode=3, count=0, message="Hello", nEffect=0x10
		input := NewEffectQuest(3, "Hello", 0x10, nil)
		got := input.Encode(nil, ctx)(nil)
		want := []byte{
			0x03,                         // mode (Decode1, switch discriminator) /*0x9b1f03*/
			0x00,                         // count (Decode1 v41)                  /*0x9b23c8*/
			0x05, 0x00,                   // message len = 5 (DecodeStr short)    /*0x9b2557*/
			0x48, 0x65, 0x6c, 0x6c, 0x6f, // "Hello"                              /*0x9b2557*/
			0x10, 0x00, 0x00, 0x00,       // nEffect = 0x10 (Decode4)             /*0x9b258e*/
		}
		if !bytes.Equal(got, want) {
			t.Errorf("self/no-rewards v87 bytes:\n got %x\nwant %x", got, want)
		}
	})

	t.Run("self/rewards/"+v87.Name, func(t *testing.T) {
		// mode=3, two rewards: gain 5 of item 0x010203, lose 1 of item 0x0A
		input := NewEffectQuest(3, "", 0, []QuestReward{
			{ItemId: 0x010203, Amount: 5},
			{ItemId: 0x00000A, Amount: -1},
		})
		got := input.Encode(nil, ctx)(nil)
		want := []byte{
			0x03,                   // mode                                /*0x9b1f03*/
			0x02,                   // count = 2 (Decode1 v41)             /*0x9b23c8*/
			0x03, 0x02, 0x01, 0x00, // reward0 itemId = 0x010203 (Decode4) /*0x9b23e4*/
			0x05, 0x00, 0x00, 0x00, // reward0 amount = 5 (Decode4 v42)    /*0x9b23f5*/
			0x0a, 0x00, 0x00, 0x00, // reward1 itemId = 0x0A (Decode4)     /*0x9b23e4*/
			0xff, 0xff, 0xff, 0xff, // reward1 amount = -1 (Decode4 v42)   /*0x9b23f5*/
		}
		if !bytes.Equal(got, want) {
			t.Errorf("self/rewards v87 bytes:\n got %x\nwant %x", got, want)
		}
	})

	t.Run("foreign/no-rewards/"+v87.Name, func(t *testing.T) {
		// characterId=0x12345678, mode=3, count=0, message="Hi", nEffect=7
		input := NewEffectQuestForeign(0x12345678, 3, "Hi", 7, nil)
		got := input.Encode(nil, ctx)(nil)
		want := []byte{
			0x78, 0x56, 0x34, 0x12, // characterId (foreign-path prefix)    /*OnUserRemotePacket*/
			0x03,                   // mode                                /*0x9b1f03*/
			0x00,                   // count = 0                           /*0x9b23c8*/
			0x02, 0x00,             // message len = 2                     /*0x9b2557*/
			0x48, 0x69,             // "Hi"                                /*0x9b2557*/
			0x07, 0x00, 0x00, 0x00, // nEffect = 7 (Decode4)               /*0x9b258e*/
		}
		if !bytes.Equal(got, want) {
			t.Errorf("foreign/no-rewards v87 bytes:\n got %x\nwant %x", got, want)
		}
	})
}

// EffectQuest / EffectQuestForeign v95 byte-fixture.
//
// Client read order — CUser::OnEffect (v95 @0x8f9a70). The SHOW_FOREIGN_EFFECT /
// SHOW_ITEM_GAIN_INCHAT opcodes both dispatch to this single function whose first
// decoded byte is the outer effect-mode discriminator (switch on Decode1 @0x8f9ab4).
//
// VERSION DELTA: the quest / item-gain effect is case **5** in v95 (@0x8faa01),
// NOT case 3 as in v83/v84/v87. The v95 OnEffect switch arms shifted — case 3 is
// now a ShowSkillAffected variant (Decode4/Decode4/Decode1), and the count+reward/
// message+nEffect body moved to case 5. The mode byte is config-resolved from the
// tenant template's effect operations table (effect_body.go ResolveCode "QUEST"),
// so the codec is mode-agnostic; this fixture passes the v95-correct discriminator
// (5) and asserts the case-5 body. The body wire format itself is byte-identical
// to v83/v84/v87:
//
//	mode   = Decode1            // outer switch discriminator (== 5 here) /*0x8f9ab4*/
//	count  = Decode1 (v66)      // number of item rewards                  /*0x8faa01*/
//	if count == 0:              // !v66 path
//	    message = DecodeStr     // ZXString length-prefixed                /*0x8faaf1*/
//	    nEffect = Decode4       // quest-effect id -> Effect_Quest         /*0x8fab9d*/
//	else:                       // loop @0x8faa1a, count times
//	    itemId = Decode4 (v67)  //                                        /*0x8faa1a*/
//	    amount = Decode4 (v68)  // signed: >1 gained, ==1, ==-1, < -1     /*0x8faa30*/
//
// The foreign (SHOW_FOREIGN_EFFECT) opcode carries a leading characterId int that
// CUserPool::OnUserRemotePacket consumes before dispatch into OnEffect;
// EffectQuestForeign prepends it. The remaining body is identical to the self form.
func TestEffectQuestByteOutputV95(t *testing.T) {
	v95 := pt.Variants[3] // GMS v95
	ctx := pt.CreateContext(v95.Region, v95.MajorVersion, v95.MinorVersion)

	t.Run("self/no-rewards/"+v95.Name, func(t *testing.T) {
		// mode=5 (v95 quest case), count=0, message="Hello", nEffect=0x10
		input := NewEffectQuest(5, "Hello", 0x10, nil)
		got := input.Encode(nil, ctx)(nil)
		want := []byte{
			0x05,                         // mode (Decode1, switch discriminator) /*0x8f9ab4*/
			0x00,                         // count (Decode1 v66)                  /*0x8faa01*/
			0x05, 0x00,                   // message len = 5 (DecodeStr short)    /*0x8faaf1*/
			0x48, 0x65, 0x6c, 0x6c, 0x6f, // "Hello"                              /*0x8faaf1*/
			0x10, 0x00, 0x00, 0x00,       // nEffect = 0x10 (Decode4)             /*0x8fab9d*/
		}
		if !bytes.Equal(got, want) {
			t.Errorf("self/no-rewards v95 bytes:\n got %x\nwant %x", got, want)
		}
	})

	t.Run("self/rewards/"+v95.Name, func(t *testing.T) {
		// mode=5, two rewards: gain 5 of item 0x010203, lose 1 of item 0x0A
		input := NewEffectQuest(5, "", 0, []QuestReward{
			{ItemId: 0x010203, Amount: 5},
			{ItemId: 0x00000A, Amount: -1},
		})
		got := input.Encode(nil, ctx)(nil)
		want := []byte{
			0x05,                   // mode                                /*0x8f9ab4*/
			0x02,                   // count = 2 (Decode1 v66)             /*0x8faa01*/
			0x03, 0x02, 0x01, 0x00, // reward0 itemId = 0x010203 (Decode4) /*0x8faa1a*/
			0x05, 0x00, 0x00, 0x00, // reward0 amount = 5 (Decode4 v68)    /*0x8faa30*/
			0x0a, 0x00, 0x00, 0x00, // reward1 itemId = 0x0A (Decode4)     /*0x8faa1a*/
			0xff, 0xff, 0xff, 0xff, // reward1 amount = -1 (Decode4 v68)   /*0x8faa30*/
		}
		if !bytes.Equal(got, want) {
			t.Errorf("self/rewards v95 bytes:\n got %x\nwant %x", got, want)
		}
	})

	t.Run("foreign/no-rewards/"+v95.Name, func(t *testing.T) {
		// characterId=0x12345678, mode=5, count=0, message="Hi", nEffect=7
		input := NewEffectQuestForeign(0x12345678, 5, "Hi", 7, nil)
		got := input.Encode(nil, ctx)(nil)
		want := []byte{
			0x78, 0x56, 0x34, 0x12, // characterId (foreign-path prefix)    /*OnUserRemotePacket*/
			0x05,                   // mode                                /*0x8f9ab4*/
			0x00,                   // count = 0                           /*0x8faa01*/
			0x02, 0x00,             // message len = 2                     /*0x8faaf1*/
			0x48, 0x69,             // "Hi"                                /*0x8faaf1*/
			0x07, 0x00, 0x00, 0x00, // nEffect = 7 (Decode4)               /*0x8fab9d*/
		}
		if !bytes.Equal(got, want) {
			t.Errorf("foreign/no-rewards v95 bytes:\n got %x\nwant %x", got, want)
		}
	})
}

// EffectQuest / EffectQuestForeign jms byte-fixture.
//
// Client read order — CUser::OnEffect (jms v185 @0x9f6395, MapleStory_dump_SCY.exe).
// The SHOW_FOREIGN_EFFECT / SHOW_ITEM_GAIN_INCHAT opcodes both dispatch to this
// single function whose first decoded byte is the outer effect-mode discriminator
// (switch on Decode1 @0x9f63c0). UNLIKE v95 (where the quest arm shifted 3->5), the
// jms quest / item-gain effect is case **3** (block head @0x9f6981) — the same
// discriminator as v83/v84/v87. The body wire format is byte-identical to GMS:
//
//	mode   = Decode1            // outer switch discriminator (== 3 here) /*0x9f63c0*/
//	count  = Decode1 (v48)      // number of item rewards                  /*0x9f698d*/
//	if count == 0:              // !v48 path
//	    message = DecodeStr     // ZXString length-prefixed                /*0x9f6b1d*/
//	    nEffect = Decode4       // quest-effect id -> Effect_Quest (sub_44477B) /*0x9f6b4f*/
//	else:                       // loop, count times
//	    itemId = Decode4 (a2)   //                                        /*0x9f69a2*/
//	    amount = Decode4 (v49)  // signed: >1 gained, ==1, ==-1, < -1     /*0x9f69ac*/
//
// The foreign (SHOW_FOREIGN_EFFECT) opcode carries a leading characterId int that
// CUserPool::OnUserRemotePacket consumes before dispatch into OnEffect;
// EffectQuestForeign prepends it. The remaining body is identical to the self form.
// The mode byte is config-resolved from the tenant template's effect operations
// table (effect_body.go ResolveCode "QUEST"), so the codec is mode-agnostic; this
// fixture passes the jms-correct discriminator (3) and asserts the case-3 body. The
// jms quest wire matches the codec exactly — no per-version delta.
//
// packet-audit:verify packet=character/clientbound/EffectQuest version=jms_v185 ida=0x9f6395
func TestEffectQuestByteOutputJMS(t *testing.T) {
	jms := pt.Variants[4] // JMS v185
	ctx := pt.CreateContext(jms.Region, jms.MajorVersion, jms.MinorVersion)

	t.Run("self/no-rewards/"+jms.Name, func(t *testing.T) {
		// mode=3 (jms quest case), count=0, message="Hello", nEffect=0x10
		input := NewEffectQuest(3, "Hello", 0x10, nil)
		got := input.Encode(nil, ctx)(nil)
		want := []byte{
			0x03,                         // mode (Decode1, switch discriminator) /*0x9f63c0*/
			0x00,                         // count (Decode1 v48)                  /*0x9f698d*/
			0x05, 0x00,                   // message len = 5 (DecodeStr short)    /*0x9f6b1d*/
			0x48, 0x65, 0x6c, 0x6c, 0x6f, // "Hello"                              /*0x9f6b1d*/
			0x10, 0x00, 0x00, 0x00,       // nEffect = 0x10 (Decode4)             /*0x9f6b4f*/
		}
		if !bytes.Equal(got, want) {
			t.Errorf("self/no-rewards jms bytes:\n got %x\nwant %x", got, want)
		}
	})

	t.Run("self/rewards/"+jms.Name, func(t *testing.T) {
		// mode=3, two rewards: gain 5 of item 0x010203, lose 1 of item 0x0A
		input := NewEffectQuest(3, "", 0, []QuestReward{
			{ItemId: 0x010203, Amount: 5},
			{ItemId: 0x00000A, Amount: -1},
		})
		got := input.Encode(nil, ctx)(nil)
		want := []byte{
			0x03,                   // mode                                /*0x9f63c0*/
			0x02,                   // count = 2 (Decode1 v48)             /*0x9f698d*/
			0x03, 0x02, 0x01, 0x00, // reward0 itemId = 0x010203 (Decode4) /*0x9f69a2*/
			0x05, 0x00, 0x00, 0x00, // reward0 amount = 5 (Decode4 v49)    /*0x9f69ac*/
			0x0a, 0x00, 0x00, 0x00, // reward1 itemId = 0x0A (Decode4)     /*0x9f69a2*/
			0xff, 0xff, 0xff, 0xff, // reward1 amount = -1 (Decode4 v49)   /*0x9f69ac*/
		}
		if !bytes.Equal(got, want) {
			t.Errorf("self/rewards jms bytes:\n got %x\nwant %x", got, want)
		}
	})

	t.Run("foreign/no-rewards/"+jms.Name, func(t *testing.T) {
		// characterId=0x12345678, mode=3, count=0, message="Hi", nEffect=7
		input := NewEffectQuestForeign(0x12345678, 3, "Hi", 7, nil)
		got := input.Encode(nil, ctx)(nil)
		want := []byte{
			0x78, 0x56, 0x34, 0x12, // characterId (foreign-path prefix)    /*OnUserRemotePacket*/
			0x03,                   // mode                                /*0x9f63c0*/
			0x00,                   // count = 0                           /*0x9f698d*/
			0x02, 0x00,             // message len = 2                     /*0x9f6b1d*/
			0x48, 0x69,             // "Hi"                                /*0x9f6b1d*/
			0x07, 0x00, 0x00, 0x00, // nEffect = 7 (Decode4)               /*0x9f6b4f*/
		}
		if !bytes.Equal(got, want) {
			t.Errorf("foreign/no-rewards jms bytes:\n got %x\nwant %x", got, want)
		}
	})
}

// EffectQuest / EffectQuestForeign v79 byte-fixture.
//
// Client read order — CUser::OnEffect (v79 @0x89112c). The SHOW_FOREIGN_EFFECT
// (op 184) and SHOW_ITEM_GAIN_INCHAT (op 192) opcodes both dispatch to this
// single function whose first decoded byte is the outer effect-mode discriminator
// (switch on Decode1 @0x89113f). Like v83/v84/v87/jms (and UNLIKE v95, where the
// quest arm shifted 3->5), the v79 quest / item-gain effect is case **3**
// (block head @0x8914e1). The body wire format is byte-identical to v83:
//
//	mode   = Decode1            // outer switch discriminator (== 3 here) /*0x89113f*/
//	count  = Decode1 (v30)      // number of item rewards                  /*0x8914ed*/
//	if count == 0:              // !v30 path @0x8914f7
//	    message = DecodeStr     // ZXString length-prefixed                /*0x89166d*/
//	    nEffect = Decode4       // quest-effect id -> Effect_Quest         /*0x8916ad*/
//	else:                       // loop @0x891509, count times
//	    itemId = Decode4 (Src)  //                                        /*0x891509*/
//	    amount = Decode4 (v31)  // signed: >1 gained, ==1, ==-1, < -1     /*0x89151a*/
//
// The foreign (SHOW_FOREIGN_EFFECT) opcode carries a leading characterId int that
// CUserPool::OnUserRemotePacket consumes before dispatch into OnEffect;
// EffectQuestForeign prepends it. The remaining body is identical to the self form.
func TestEffectQuestByteOutputV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)

	t.Run("self/no-rewards", func(t *testing.T) {
		// mode=3, count=0, message="Hello", nEffect=0x10
		got := NewEffectQuest(3, "Hello", 0x10, nil).Encode(nil, ctx)(nil)
		want := []byte{
			0x03,                         // mode (Decode1, switch discriminator) /*0x89113f*/
			0x00,                         // count (Decode1 v30)                  /*0x8914ed*/
			0x05, 0x00,                   // message len = 5 (DecodeStr short)    /*0x89166d*/
			0x48, 0x65, 0x6c, 0x6c, 0x6f, // "Hello"                              /*0x89166d*/
			0x10, 0x00, 0x00, 0x00,       // nEffect = 0x10 (Decode4)             /*0x8916ad*/
		}
		if !bytes.Equal(got, want) {
			t.Errorf("self/no-rewards v79 bytes:\n got %x\nwant %x", got, want)
		}
	})

	t.Run("self/rewards", func(t *testing.T) {
		// mode=3, two rewards: gain 5 of item 0x010203, lose 1 of item 0x0A
		got := NewEffectQuest(3, "", 0, []QuestReward{
			{ItemId: 0x010203, Amount: 5},
			{ItemId: 0x00000A, Amount: -1},
		}).Encode(nil, ctx)(nil)
		want := []byte{
			0x03,                   // mode                                /*0x89113f*/
			0x02,                   // count = 2 (Decode1 v30)             /*0x8914ed*/
			0x03, 0x02, 0x01, 0x00, // reward0 itemId = 0x010203 (Decode4) /*0x891509*/
			0x05, 0x00, 0x00, 0x00, // reward0 amount = 5 (Decode4 v31)    /*0x89151a*/
			0x0a, 0x00, 0x00, 0x00, // reward1 itemId = 0x0A (Decode4)     /*0x891509*/
			0xff, 0xff, 0xff, 0xff, // reward1 amount = -1 (Decode4 v31)   /*0x89151a*/
		}
		if !bytes.Equal(got, want) {
			t.Errorf("self/rewards v79 bytes:\n got %x\nwant %x", got, want)
		}
	})

	t.Run("foreign/no-rewards", func(t *testing.T) {
		// characterId=0x12345678, mode=3, count=0, message="Hi", nEffect=7
		got := NewEffectQuestForeign(0x12345678, 3, "Hi", 7, nil).Encode(nil, ctx)(nil)
		want := []byte{
			0x78, 0x56, 0x34, 0x12, // characterId (foreign-path prefix)    /*OnUserRemotePacket*/
			0x03,                   // mode                                /*0x89113f*/
			0x00,                   // count = 0                           /*0x8914ed*/
			0x02, 0x00,             // message len = 2                     /*0x89166d*/
			0x48, 0x69,             // "Hi"                                /*0x89166d*/
			0x07, 0x00, 0x00, 0x00, // nEffect = 7 (Decode4)               /*0x8916ad*/
		}
		if !bytes.Equal(got, want) {
			t.Errorf("foreign/no-rewards v79 bytes:\n got %x\nwant %x", got, want)
		}
	})
}
