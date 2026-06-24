package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// CharacterList byte-fixture.
//
// Client read order — CLogin::OnSelectWorldResult (v83 @0x5f9891), the
// world-select-success path (v3==0, LABEL_34 @0x5f9a22):
//
//	status = Decode1                       // result/status byte             /*0x5f98e9*/
//	count  = Decode1 (v12)                 // number of avatar entries        /*0x5f9a61*/
//	for each of count entries:             // loop @0x5f9a83
//	    GW_CharacterStat::Decode           // statistics block (@0x4e2a84)    /*0x5f9a90*/
//	    AvatarLook::Decode                 // avatar/look block (@0x4e749a)   /*0x5f9a9e*/
//	    family = Decode1 (*v14)            // viewAll/family flag byte         /*0x5f9abd*/
//	    rankEnabled = Decode1              // 0 => zeros; else DecodeBuffer(16)/*0x5f9abf*/
//	        if rankEnabled: rank/rankMove/jobRank/jobRankMove (4x Decode4)    /*0x5f9ada*/
//	hasPic = Decode1 (m_bLoginOpt)         //                                 /*0x5f9b34*/
//	slots  = Decode4 (m_nSlotCount)        //                                 /*0x5f9b3a*/
//	// GMS major>87 reads an extra nBuyCharCount int; v83 does not.
//
// GW_CharacterStat::Decode (v83 @0x4e2a84, list path a3=0):
//	id=Decode4, name=DecodeBuffer(13), gender=Decode1, skin=Decode1,
//	face=Decode4, hair=Decode4, petLockerSN=DecodeBuffer(24)=3x long,
//	level=Decode1, then 10x Decode2 (job,str,dex,int,luk,hp,maxHp,mp,maxMp,ap),
//	sp=Decode2, exp=Decode4, fame=Decode2, gachaExp=Decode4, mapId=Decode4,
//	spawnPoint=Decode1, trailing Decode4 (GMS major>12).
//
// AvatarLook::Decode (v83 @0x4e749a): gender,skin,face,!mega,hair, equip loop
// (0xFF term), masked loop (0xFF term), cashWeapon=Decode4, pets=DecodeBuffer(12).
//
// packet-audit:verify packet=character/clientbound/CharacterList version=gms_v83 ida=0x5f9891
func TestCharacterListByteOutput(t *testing.T) {
	v83 := pt.Variants[1] // GMS v83
	ctx := pt.CreateContext(v83.Region, v83.MajorVersion, v83.MinorVersion)

	stats := model.NewCharacterStatistics(
		0x01020304,        // id
		"Hero",            // name (padded to 13)
		0,                 // gender
		0,                 // skinColor
		0x4D2,             // face
		0x7B,              // hair
		[3]uint64{0, 0, 0}, // petIds
		0x0A,              // level
		0x64,              // jobId
		4, 5, 6, 7,        // str, dex, int, luck
		0x64, 0x64, 0x32, 0x32, // hp, maxHp, mp, maxMp
		3,                 // ap
		false,             // hasSPTable (write sp short)
		2,                 // sp
		0,                 // experience
		8,                 // fame
		0,                 // gachaponExperience
		0x0BB8,            // mapId
		0,                 // spawnPoint
	)
	// Empty equipment/masked maps + nil pets -> deterministic avatar block.
	avatar := model.NewAvatar(0, 0, 0x4D2, false, 0x7B, nil, nil, nil)
	entry := model.NewCharacterListEntry(stats, avatar, false /*viewAll*/, false /*gm*/, 1, 2, 3, 4)

	input := NewCharacterList(0 /*status*/, []model.CharacterListEntry{entry}, false /*hasPic*/, 8 /*slots*/)
	got := input.Encode(nil, ctx)(nil)

	want := []byte{
		0x00, // status (Decode1)                                  /*0x5f98e9*/
		0x01, // count = 1 (Decode1)                               /*0x5f9a61*/

		// --- GW_CharacterStat block (entry 0) --- @0x4e2a84
		0x04, 0x03, 0x02, 0x01, // id = 0x01020304 (Decode4)       /*0x4e2aa7*/
		0x48, 0x65, 0x72, 0x6f, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // "Hero"+pad to 13 /*0x4e2ab8*/
		0x00,                   // gender (Decode1)                /*0x4e2acf*/
		0x00,                   // skin (Decode1)                  /*0x4e2ae4*/
		0xd2, 0x04, 0x00, 0x00, // face (Decode4)                  /*0x4e2af9*/
		0x7b, 0x00, 0x00, 0x00, // hair (Decode4)                  /*0x4e2b0e*/
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // pet long 0 (DecodeBuffer 24) /*0x4e2b19*/
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // pet long 1
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // pet long 2
		0x0a,                   // level (Decode1)                 /*0x4e2b20*/
		0x64, 0x00,             // jobId (Decode2)                 /*0x4e2b34*/
		0x04, 0x00,             // str (Decode2)                   /*0x4e2b4a*/
		0x05, 0x00,             // dex                             /*0x4e2b5e*/
		0x06, 0x00,             // int                             /*0x4e2b72*/
		0x07, 0x00,             // luck                            /*0x4e2b86*/
		0x64, 0x00,             // hp                              /*0x4e2b9a*/
		0x64, 0x00,             // maxHp                           /*0x4e2bae*/
		0x32, 0x00,             // mp                              /*0x4e2bc2*/
		0x32, 0x00,             // maxMp                           /*0x4e2bd6*/
		0x03, 0x00,             // ap (Decode2)                    /*0x4e2bea*/
		0x02, 0x00,             // sp (Decode2, !hasSPTable)       /*0x4e2c49*/
		0x00, 0x00, 0x00, 0x00, // experience (Decode4)            /*0x4e2c6e*/
		0x08, 0x00,             // fame (Decode2)                  /*0x4e2c88*/
		0x00, 0x00, 0x00, 0x00, // gachaExp (Decode4, GMS>28)      /*0x4e2ca2*/
		0xb8, 0x0b, 0x00, 0x00, // mapId (Decode4)                 /*0x4e2cbc*/
		0x00,                   // spawnPoint (Decode1)            /*0x4e2cdd*/
		0x00, 0x00, 0x00, 0x00, // trailing int (Decode4, GMS>12)  /*0x4e2ce3*/

		// --- AvatarLook block (entry 0) --- @0x4e749a
		0x00,                   // gender                          /*0x4e74ad*/
		0x00,                   // skin                            /*0x4e74ba*/
		0xd2, 0x04, 0x00, 0x00, // face                            /*0x4e74ce*/
		0x01,                   // !mega -> WriteBool(true)        /*0x4e74ea*/
		0x7b, 0x00, 0x00, 0x00, // hair                            /*0x4e74f6*/
		0xff,                   // equip terminator                /*0x4e74ff*/
		0xff,                   // masked terminator               /*0x4e7536*/
		0x00, 0x00, 0x00, 0x00, // cash weapon                     /*0x4e7572*/
		0x00, 0x00, 0x00, 0x00, // pet 0 (DecodeBuffer 12)         /*0x4e7585*/
		0x00, 0x00, 0x00, 0x00, // pet 1
		0x00, 0x00, 0x00, 0x00, // pet 2

		// --- entry trailer ---
		0x00,                   // family/viewAll flag (Decode1)   /*0x5f9abd*/
		0x01,                   // rankEnabled = !gm (Decode1)     /*0x5f9abf*/
		0x01, 0x00, 0x00, 0x00, // rank (Decode4)                  /*0x5f9ada DecodeBuffer 16*/
		0x02, 0x00, 0x00, 0x00, // rankMove
		0x03, 0x00, 0x00, 0x00, // jobRank
		0x04, 0x00, 0x00, 0x00, // jobRankMove

		// --- list trailer ---
		0x00,                   // hasPic (Decode1)                /*0x5f9b34*/
		0x08, 0x00, 0x00, 0x00, // slots (Decode4)                 /*0x5f9b3a*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("character-list bytes:\n got %x\nwant %x", got, want)
	}
}

func TestCharacterListRoundTrip(t *testing.T) {
	v83 := pt.Variants[1]
	ctx := pt.CreateContext(v83.Region, v83.MajorVersion, v83.MinorVersion)
	stats := model.NewCharacterStatistics(
		0x01020304, "Hero", 0, 0, 0x4D2, 0x7B, [3]uint64{0, 0, 0},
		0x0A, 0x64, 4, 5, 6, 7, 0x64, 0x64, 0x32, 0x32, 3, false, 2, 0, 8, 0, 0x0BB8, 0,
	)
	avatar := model.NewAvatar(0, 0, 0x4D2, false, 0x7B, nil, nil, nil)
	entry := model.NewCharacterListEntry(stats, avatar, false, false, 1, 2, 3, 4)
	input := NewCharacterList(0, []model.CharacterListEntry{entry}, false, 8)
	output := CharacterList{}
	pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
	if output.Status() != input.Status() {
		t.Errorf("status: got %v want %v", output.Status(), input.Status())
	}
	if len(output.Characters()) != len(input.Characters()) {
		t.Errorf("characters: got %d want %d", len(output.Characters()), len(input.Characters()))
	}
	if output.CharacterSlots() != input.CharacterSlots() {
		t.Errorf("slots: got %v want %v", output.CharacterSlots(), input.CharacterSlots())
	}
}
