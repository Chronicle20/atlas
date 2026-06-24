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
