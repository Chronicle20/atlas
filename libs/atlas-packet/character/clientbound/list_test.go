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
// packet-audit:verify packet=character/clientbound/CharacterList version=gms_v84 ida=0x60e8c6
// packet-audit:verify packet=character/clientbound/CharacterList version=gms_v87 ida=0x63115a
// packet-audit:verify packet=character/clientbound/CharacterList version=gms_v95 ida=0x5dda00
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

// CharacterList v84 byte-fixture.
//
// Client read order — CLogin::OnSelectWorldResult (v84 @0x60e8c6), the
// world-select-success path (LABEL_33, v4==0 @0x60ea57). The read order is
// byte-identical to v83 (v84 body ≡ v83 below ~0x3D, IDA-confirmed):
//
//	status = Decode1                       // result/status byte (earlier path) /*0x60e91e*/
//	count  = Decode1 (v10)                 // number of avatar entries           /*0x60ea96*/
//	for each of count entries:             // loop @0x60eb58
//	    GW_CharacterStat::Decode           // statistics block (@0x4e9da4)       /*0x60eac5*/
//	    AvatarLook::Decode                 // avatar/look block (@0x4ef958)      /*0x60ead3*/
//	    family = Decode1 (*v12)            // viewAll/family flag byte            /*0x60eaf2*/
//	    rankEnabled = Decode1              // 0 => zeros; else DecodeBuffer(16)   /*0x60eaf4*/
//	        if rankEnabled: rank/rankMove/jobRank/jobRankMove (4x Decode4)        /*0x60eb0f*/
//	hasPic = Decode1 (this+392)            //                                    /*0x60eb69*/
//	slots  = Decode4 (this+400)            //                                    /*0x60eb6f*/
//	// GMS major>87 reads an extra nBuyCharCount int; v84 does not.
//
// GW_CharacterStat::Decode (v84 @0x4e9da4, list path a3=0): id=Decode4,
// name=DecodeBuffer(13), gender=Decode1, skin=Decode1, face=Decode4,
// hair=Decode4, petLockerSN=DecodeBuffer(24), level=Decode1, 10x Decode2
// (job,str,dex,int,luk,hp,maxHp,mp,maxMp,ap), sp=Decode2 (non-22xx job path),
// exp=Decode4, fame=Decode2, gachaExp=Decode4, mapId=Decode4, spawnPoint=Decode1,
// trailing Decode4.
//
// AvatarLook::Decode (v84 @0x4ef958): gender,skin,face,!mega,hair, equip loop
// (0xFF term), masked loop (0xFF term), cashWeapon=Decode4, pets=DecodeBuffer(12).
//
// The wire bytes mirror the v83 fixture exactly.
func TestCharacterListByteOutputV84(t *testing.T) {
	v84 := pt.Variants[5] // GMS v84
	ctx := pt.CreateContext(v84.Region, v84.MajorVersion, v84.MinorVersion)

	stats := model.NewCharacterStatistics(
		0x01020304,         // id
		"Hero",             // name (padded to 13)
		0,                  // gender
		0,                  // skinColor
		0x4D2,              // face
		0x7B,               // hair
		[3]uint64{0, 0, 0}, // petIds
		0x0A,               // level
		0x64,               // jobId
		4, 5, 6, 7,         // str, dex, int, luck
		0x64, 0x64, 0x32, 0x32, // hp, maxHp, mp, maxMp
		3,      // ap
		false,  // hasSPTable (write sp short)
		2,      // sp
		0,      // experience
		8,      // fame
		0,      // gachaponExperience
		0x0BB8, // mapId
		0,      // spawnPoint
	)
	avatar := model.NewAvatar(0, 0, 0x4D2, false, 0x7B, nil, nil, nil)
	entry := model.NewCharacterListEntry(stats, avatar, false /*viewAll*/, false /*gm*/, 1, 2, 3, 4)

	input := NewCharacterList(0 /*status*/, []model.CharacterListEntry{entry}, false /*hasPic*/, 8 /*slots*/)
	got := input.Encode(nil, ctx)(nil)

	want := []byte{
		0x00, // status (Decode1)                                  /*0x60e91e*/
		0x01, // count = 1 (Decode1)                               /*0x60ea96*/

		// --- GW_CharacterStat block (entry 0) --- @0x4e9da4
		0x04, 0x03, 0x02, 0x01, // id = 0x01020304 (Decode4)       /*0x4e9dc7*/
		0x48, 0x65, 0x72, 0x6f, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // "Hero"+pad to 13 /*0x4e9dd8*/
		0x00,                   // gender (Decode1)                /*0x4e9def*/
		0x00,                   // skin (Decode1)                  /*0x4e9e04*/
		0xd2, 0x04, 0x00, 0x00, // face (Decode4)                  /*0x4e9e19*/
		0x7b, 0x00, 0x00, 0x00, // hair (Decode4)                  /*0x4e9e2e*/
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // pet long 0 (DecodeBuffer 24) /*0x4e9e39*/
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // pet long 1
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // pet long 2
		0x0a,       // level (Decode1)                 /*0x4e9e48*/
		0x64, 0x00, // jobId (Decode2)                 /*0x4e9e54*/
		0x04, 0x00, // str (Decode2)                   /*0x4e9e6a*/
		0x05, 0x00, // dex                             /*0x4e9e7e*/
		0x06, 0x00, // int                             /*0x4e9e92*/
		0x07, 0x00, // luck                            /*0x4e9ea6*/
		0x64, 0x00, // hp                              /*0x4e9eba*/
		0x64, 0x00, // maxHp                           /*0x4e9ece*/
		0x32, 0x00, // mp                              /*0x4e9ee2*/
		0x32, 0x00, // maxMp                           /*0x4e9ef6*/
		0x03, 0x00, // ap (Decode2)                    /*0x4e9f0a*/
		0x02, 0x00, // sp (Decode2, non-22xx job path) /*0x4e9f69*/
		0x00, 0x00, 0x00, 0x00, // experience (Decode4)            /*0x4e9f8e*/
		0x08, 0x00,             // fame (Decode2)                  /*0x4e9fa8*/
		0x00, 0x00, 0x00, 0x00, // gachaExp (Decode4)              /*0x4e9fc2*/
		0xb8, 0x0b, 0x00, 0x00, // mapId (Decode4)                 /*0x4e9fdc*/
		0x00,                   // spawnPoint (Decode1)            /*0x4e9ffd*/
		0x00, 0x00, 0x00, 0x00, // trailing int (Decode4)          /*0x4ea003*/

		// --- AvatarLook block (entry 0) --- @0x4ef958
		0x00,                   // gender                          /*0x4ef96b*/
		0x00,                   // skin                            /*0x4ef978*/
		0xd2, 0x04, 0x00, 0x00, // face                            /*0x4ef98c*/
		0x01,                   // !mega -> WriteBool(true)        /*0x4ef9a8*/
		0x7b, 0x00, 0x00, 0x00, // hair                            /*0x4ef9b4*/
		0xff,                   // equip terminator                /*0x4ef9bd*/
		0xff,                   // masked terminator               /*0x4ef9f4*/
		0x00, 0x00, 0x00, 0x00, // cash weapon                     /*0x4efa30*/
		0x00, 0x00, 0x00, 0x00, // pet 0 (DecodeBuffer 12)         /*0x4efa43*/
		0x00, 0x00, 0x00, 0x00, // pet 1
		0x00, 0x00, 0x00, 0x00, // pet 2

		// --- entry trailer ---
		0x00,                   // family/viewAll flag (Decode1)   /*0x60eaf2*/
		0x01,                   // rankEnabled = !gm (Decode1)     /*0x60eaf4*/
		0x01, 0x00, 0x00, 0x00, // rank (Decode4)                  /*0x60eb0f DecodeBuffer 16*/
		0x02, 0x00, 0x00, 0x00, // rankMove
		0x03, 0x00, 0x00, 0x00, // jobRank
		0x04, 0x00, 0x00, 0x00, // jobRankMove

		// --- list trailer ---
		0x00,                   // hasPic (Decode1)                /*0x60eb69*/
		0x08, 0x00, 0x00, 0x00, // slots (Decode4)                 /*0x60eb6f*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("character-list v84 bytes:\n got %x\nwant %x", got, want)
	}
}

// CharacterList v87 byte-fixture.
//
// Client read order — CLogin::OnSelectWorldResult (v87 @0x63115a), the
// world-select-success path (LABEL_33, !v4 / v4==12 / v4==23 @0x6312eb):
//
//	status = Decode1                       // result/status byte (earlier path) /*0x6311b2*/
//	count  = Decode1 (v10)                 // number of avatar entries           /*0x63132a*/
//	for each of count entries:             // loop @0x6313ec, 15 slots, count decoded
//	    GW_CharacterStat::Decode           // statistics block (@0x501d0e)       /*0x631359*/
//	    AvatarLook::Decode                 // avatar/look block (@0x508277)      /*0x631367*/
//	    family = Decode1 (*v12)            // viewAll/family flag byte            /*0x631386*/
//	    rankEnabled = Decode1              // 0 => zeros; else DecodeBuffer(16)   /*0x631388*/
//	        if rankEnabled: rank/rankMove/jobRank/jobRankMove (4x Decode4)        /*0x6313a3*/
//	hasPic = Decode1 (m_bLoginOpt)         //                                    /*0x6313fd*/
//	slots  = Decode4 (m_nSlotCount)        //                                    /*0x631403*/
//	// GMS major>87 reads an extra nBuyCharCount int; v87 (==87) does NOT.
//
// GW_CharacterStat::Decode (v87 @0x501d0e, list path bBackwardUpdate=0):
// id=Decode4, name=DecodeBuffer(13), gender=Decode1, skin=Decode1, face=Decode4,
// hair=Decode4, petLockerSN=DecodeBuffer(24), level=Decode1, 10x Decode2
// (job,str,dex,int,luk,hp,maxHp,mp,maxMp,ap), sp=Decode2 (non-22xx job path),
// exp=Decode4, fame=Decode2, gachaExp=Decode4, mapId=Decode4, spawnPoint=Decode1,
// trailing Decode4 (this+227 @0x501f74), then trailing Decode2 (this+231 @0x501f80).
//
// The trailing Decode2 (nSubJob) is the ONLY structural delta vs v83/v84: v87
// reads an extra short after the trailing int. The Atlas codec writes it at
// MajorVersion()>=87 (character_statistics.go WriteShort(0) // nSubJob), so the
// v87 wire carries two extra 0x00 bytes in the GW_CharacterStat block.
//
// AvatarLook::Decode (v87 @0x508277): gender,skin,face,!mega,hair, equip loop
// (0xFF term), masked loop (0xFF term), cashWeapon=Decode4, pets=DecodeBuffer(12).
func TestCharacterListByteOutputV87(t *testing.T) {
	v87 := pt.Variants[2] // GMS v87
	ctx := pt.CreateContext(v87.Region, v87.MajorVersion, v87.MinorVersion)

	stats := model.NewCharacterStatistics(
		0x01020304,         // id
		"Hero",             // name (padded to 13)
		0,                  // gender
		0,                  // skinColor
		0x4D2,              // face
		0x7B,               // hair
		[3]uint64{0, 0, 0}, // petIds
		0x0A,               // level
		0x64,               // jobId
		4, 5, 6, 7,         // str, dex, int, luck
		0x64, 0x64, 0x32, 0x32, // hp, maxHp, mp, maxMp
		3,      // ap
		false,  // hasSPTable (write sp short)
		2,      // sp
		0,      // experience
		8,      // fame
		0,      // gachaponExperience
		0x0BB8, // mapId
		0,      // spawnPoint
	)
	avatar := model.NewAvatar(0, 0, 0x4D2, false, 0x7B, nil, nil, nil)
	entry := model.NewCharacterListEntry(stats, avatar, false /*viewAll*/, false /*gm*/, 1, 2, 3, 4)

	input := NewCharacterList(0 /*status*/, []model.CharacterListEntry{entry}, false /*hasPic*/, 8 /*slots*/)
	got := input.Encode(nil, ctx)(nil)

	want := []byte{
		0x00, // status (Decode1)                                  /*0x6311b2*/
		0x01, // count = 1 (Decode1)                               /*0x63132a*/

		// --- GW_CharacterStat block (entry 0) --- @0x501d0e
		0x04, 0x03, 0x02, 0x01, // id = 0x01020304 (Decode4)       /*0x501d31*/
		0x48, 0x65, 0x72, 0x6f, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // "Hero"+pad to 13 /*0x501d42*/
		0x00,                   // gender (Decode1)                /*0x501d59*/
		0x00,                   // skin (Decode1)                  /*0x501d6e*/
		0xd2, 0x04, 0x00, 0x00, // face (Decode4)                  /*0x501d83*/
		0x7b, 0x00, 0x00, 0x00, // hair (Decode4)                  /*0x501d98*/
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // pet long 0 (DecodeBuffer 24) /*0x501da3*/
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // pet long 1
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // pet long 2
		0x0a,       // level (Decode1)                 /*0x501db2*/
		0x64, 0x00, // jobId (Decode2)                 /*0x501dbe*/
		0x04, 0x00, // str (Decode2)                   /*0x501dd4*/
		0x05, 0x00, // dex                             /*0x501de8*/
		0x06, 0x00, // int                             /*0x501dfc*/
		0x07, 0x00, // luck                            /*0x501e10*/
		0x64, 0x00, // hp                              /*0x501e24*/
		0x64, 0x00, // maxHp                           /*0x501e38*/
		0x32, 0x00, // mp                              /*0x501e4c*/
		0x32, 0x00, // maxMp                           /*0x501e60*/
		0x03, 0x00, // ap (Decode2)                    /*0x501e74*/
		0x02, 0x00, // sp (Decode2, non-22xx job path) /*0x501ed3*/
		0x00, 0x00, 0x00, 0x00, // experience (Decode4)            /*0x501ef8*/
		0x08, 0x00,             // fame (Decode2)                  /*0x501f12*/
		0x00, 0x00, 0x00, 0x00, // gachaExp (Decode4)              /*0x501f2c*/
		0xb8, 0x0b, 0x00, 0x00, // mapId (Decode4)                 /*0x501f46*/
		0x00,                   // spawnPoint (Decode1)            /*0x501f67*/
		0x00, 0x00, 0x00, 0x00, // trailing int (Decode4)          /*0x501f74*/
		0x00, 0x00,             // nSubJob (Decode2, GMS>=87)       /*0x501f80*/

		// --- AvatarLook block (entry 0) --- @0x508277
		0x00,                   // gender                          /*0x50828a*/
		0x00,                   // skin                            /*0x508297*/
		0xd2, 0x04, 0x00, 0x00, // face                            /*0x5082ab*/
		0x01,                   // !mega -> WriteBool(true)        /*0x5082c7*/
		0x7b, 0x00, 0x00, 0x00, // hair                            /*0x5082d3*/
		0xff,                   // equip terminator                /*0x5082dc*/
		0xff,                   // masked terminator               /*0x508313*/
		0x00, 0x00, 0x00, 0x00, // cash weapon                     /*0x50834f*/
		0x00, 0x00, 0x00, 0x00, // pet 0 (DecodeBuffer 12)         /*0x50835d*/
		0x00, 0x00, 0x00, 0x00, // pet 1
		0x00, 0x00, 0x00, 0x00, // pet 2

		// --- entry trailer ---
		0x00,                   // family/viewAll flag (Decode1)   /*0x631386*/
		0x01,                   // rankEnabled = !gm (Decode1)     /*0x631388*/
		0x01, 0x00, 0x00, 0x00, // rank (Decode4)                  /*0x6313a3 DecodeBuffer 16*/
		0x02, 0x00, 0x00, 0x00, // rankMove
		0x03, 0x00, 0x00, 0x00, // jobRank
		0x04, 0x00, 0x00, 0x00, // jobRankMove

		// --- list trailer ---
		0x00,                   // hasPic (Decode1)                /*0x6313fd*/
		0x08, 0x00, 0x00, 0x00, // slots (Decode4)                 /*0x631403*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("character-list v87 bytes:\n got %x\nwant %x", got, want)
	}
}

// CharacterList v95 byte-fixture.
//
// Client read order — CLogin::OnSelectWorldResult (v95 @0x5dda00), the
// world-select-success path (v4==0 / v4==12 / v4==23 fallthrough @0x5ddd10):
//
//	status = Decode1                       // result/status byte (earlier path) /*0x5dda5f*/
//	count  = Decode1 (nCount)              // number of avatar entries           /*0x5ddd4b*/
//	for each of count entries:             // loop @0x5ddd66, 15 slots, count decoded
//	    GW_CharacterStat::Decode           // statistics block (@0x4f9d40)       /*0x5ddd7d*/
//	    AvatarLook::Decode                 // avatar/look block (@0x4f2c00)      /*0x5ddd8d*/
//	    family = Decode1 (*v28)            // viewAll/family flag byte            /*0x5ddda5*/
//	    rankEnabled = Decode1              // 0 => zeros; else DecodeBuffer(16)   /*0x5dddb5*/
//	        if rankEnabled: rank/rankMove/jobRank/jobRankMove (DecodeBuffer 16)   /*0x5dddd0*/
//	hasPic = Decode1 (m_bLoginOpt)         //                                    /*0x5dde34*/
//	slots  = Decode4 (m_nSlotCount)        //                                    /*0x5dde41*/
//	nBuyCharCount = Decode4 (m_nBuyCharCount) // read UNCONDITIONALLY in v95     /*0x5dde4c*/
//
// Two structural deltas vs v87:
//   (1) GW_CharacterStat reads HP/MaxHP/MP/MaxMP as Decode4 (int), not Decode2
//       (short) — IDA @0x4f9e56/0x4f9e6a/0x4f9e7e/0x4f9e95. The codec widens
//       these at MajorVersion()>=95 (character_statistics.go), adding 8 bytes.
//   (2) OnSelectWorldResult reads the trailing nBuyCharCount int unconditionally
//       (@0x5dde4c). The codec emits it at MajorVersion()>87 (list.go), adding 4
//       bytes. The nSubJob short (Decode2 @0x4f9fe2, GMS>=87) is also present.
//
// GW_CharacterStat::Decode (v95 @0x4f9d40, list path bBackwardUpdate=0):
// id=Decode4, name=DecodeBuffer(13), gender=Decode1, skin=Decode1, face=Decode4,
// hair=Decode4, petLockerSN=DecodeBuffer(24), level=Decode1, job/str/dex/int/luk
// (5x Decode2), hp/maxHp/mp/maxMp (4x Decode4), ap=Decode2, sp=Decode2 (non-3xxx/
// 22xx/2001 job path), exp=Decode4, fame=Decode2, gachaExp=Decode4, mapId=Decode4,
// spawnPoint=Decode1, nPlaytime=Decode4, nSubJob=Decode2.
//
// AvatarLook::Decode (v95 @0x4f2c00): gender,skin,face,!mega,hair, equip loop
// (0xFF term), masked loop (0xFF term), cashWeapon=Decode4, pets=DecodeBuffer(12).
func TestCharacterListByteOutputV95(t *testing.T) {
	v95 := pt.Variants[3] // GMS v95
	ctx := pt.CreateContext(v95.Region, v95.MajorVersion, v95.MinorVersion)

	stats := model.NewCharacterStatistics(
		0x01020304,         // id
		"Hero",             // name (padded to 13)
		0,                  // gender
		0,                  // skinColor
		0x4D2,              // face
		0x7B,               // hair
		[3]uint64{0, 0, 0}, // petIds
		0x0A,               // level
		0x64,               // jobId
		4, 5, 6, 7,         // str, dex, int, luck
		0x64, 0x64, 0x32, 0x32, // hp, maxHp, mp, maxMp
		3,      // ap
		false,  // hasSPTable (write sp short)
		2,      // sp
		0,      // experience
		8,      // fame
		0,      // gachaponExperience
		0x0BB8, // mapId
		0,      // spawnPoint
	)
	avatar := model.NewAvatar(0, 0, 0x4D2, false, 0x7B, nil, nil, nil)
	entry := model.NewCharacterListEntry(stats, avatar, false /*viewAll*/, false /*gm*/, 1, 2, 3, 4)

	input := NewCharacterList(0 /*status*/, []model.CharacterListEntry{entry}, false /*hasPic*/, 8 /*slots*/)
	got := input.Encode(nil, ctx)(nil)

	want := []byte{
		0x00, // status (Decode1)                                  /*0x5dda5f*/
		0x01, // count = 1 (Decode1)                               /*0x5ddd4b*/

		// --- GW_CharacterStat block (entry 0) --- @0x4f9d40
		0x04, 0x03, 0x02, 0x01, // id = 0x01020304 (Decode4)       /*0x4f9d71*/
		0x48, 0x65, 0x72, 0x6f, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // "Hero"+pad to 13 /*0x4f9d7b*/
		0x00,                   // gender (Decode1)                /*0x4f9da9*/
		0x00,                   // skin (Decode1)                  /*0x4f9db3*/
		0xd2, 0x04, 0x00, 0x00, // face (Decode4)                  /*0x4f9dbd*/
		0x7b, 0x00, 0x00, 0x00, // hair (Decode4)                  /*0x4f9dc5*/
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // pet long 0 (DecodeBuffer 24) /*0x4f9dd0*/
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // pet long 1
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // pet long 2
		0x0a,                   // level (Decode1)                 /*0x4f9dd7*/
		0x64, 0x00,             // jobId (Decode2)                 /*0x4f9deb*/
		0x04, 0x00,             // str (Decode2)                   /*0x4f9e02*/
		0x05, 0x00,             // dex                             /*0x4f9e17*/
		0x06, 0x00,             // int                             /*0x4f9e2c*/
		0x07, 0x00,             // luck                            /*0x4f9e41*/
		0x64, 0x00, 0x00, 0x00, // hp (Decode4, v95-widened)       /*0x4f9e56*/
		0x64, 0x00, 0x00, 0x00, // maxHp (Decode4)                 /*0x4f9e6a*/
		0x32, 0x00, 0x00, 0x00, // mp (Decode4)                    /*0x4f9e7e*/
		0x32, 0x00, 0x00, 0x00, // maxMp (Decode4)                 /*0x4f9e95*/
		0x03, 0x00,             // ap (Decode2)                    /*0x4f9eaf*/
		0x02, 0x00,             // sp (Decode2, non-3xxx/22xx job) /*0x4f9f2f*/
		0x00, 0x00, 0x00, 0x00, // experience (Decode4)            /*0x4f9f55*/
		0x08, 0x00,             // fame (Decode2)                  /*0x4f9f6f*/
		0x00, 0x00, 0x00, 0x00, // gachaExp (Decode4)              /*0x4f9f8a*/
		0xb8, 0x0b, 0x00, 0x00, // mapId (Decode4)                 /*0x4f9fa4*/
		0x00,                   // spawnPoint (Decode1)            /*0x4f9fc5*/
		0x00, 0x00, 0x00, 0x00, // nPlaytime trailing int (Decode4)/*0x4f9fd2*/
		0x00, 0x00,             // nSubJob (Decode2, GMS>=87)      /*0x4f9fe2*/

		// --- AvatarLook block (entry 0) --- @0x4f2c00
		0x00,                   // gender                          /*0x4f2c13*/
		0x00,                   // skin                            /*0x4f2c20*/
		0xd2, 0x04, 0x00, 0x00, // face                            /*0x4f2c33*/
		0x01,                   // !mega -> WriteBool(true)        /*0x4f2c53*/
		0x7b, 0x00, 0x00, 0x00, // hair                            /*0x4f2c61*/
		0xff,                   // equip terminator                /*0x4f2c6d*/
		0xff,                   // masked terminator               /*0x4f2cb3*/
		0x00, 0x00, 0x00, 0x00, // cash weapon                     /*0x4f2cf6*/
		0x00, 0x00, 0x00, 0x00, // pet 0 (DecodeBuffer 12)         /*0x4f2d04*/
		0x00, 0x00, 0x00, 0x00, // pet 1
		0x00, 0x00, 0x00, 0x00, // pet 2

		// --- entry trailer ---
		0x00,                   // family/viewAll flag (Decode1)   /*0x5ddda5*/
		0x01,                   // rankEnabled = !gm (Decode1)     /*0x5dddb5*/
		0x01, 0x00, 0x00, 0x00, // rank (Decode4)                  /*0x5dddd0 DecodeBuffer 16*/
		0x02, 0x00, 0x00, 0x00, // rankMove
		0x03, 0x00, 0x00, 0x00, // jobRank
		0x04, 0x00, 0x00, 0x00, // jobRankMove

		// --- list trailer ---
		0x00,                   // hasPic (Decode1)                /*0x5dde34*/
		0x08, 0x00, 0x00, 0x00, // slots (Decode4)                 /*0x5dde41*/
		0x00, 0x00, 0x00, 0x00, // nBuyCharCount (Decode4, GMS>87) /*0x5dde4c*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("character-list v95 bytes:\n got %x\nwant %x", got, want)
	}
}

// CharacterList jms byte-fixture.
//
// Client read order — CLogin::OnSelectWorldResult (jms v185 @0x66f3d8,
// MapleStory_dump_SCY.exe), the world-select-success path (v34==0 || v34==12):
//
//	status = Decode1                       // result/status byte              /*0x66f411*/
//	_      = DecodeStr                      // JMS leading ASCII string (empty)/*0x66f72e*/
//	count  = Decode1 (v29)                 // number of avatar entries        /*0x66f73d*/
//	for each of count entries:             // loop, 18 slots, count decoded
//	    GW_CharacterStat::Decode(_,_,0)    // statistics block (@0x50ec17)    /*0x66f76c*/
//	    AvatarLook::Decode                 // avatar/look block (@0x51517e)   /*0x66f77a*/
//	    family = Decode1 (*v21)            // viewAll/family flag byte         /*0x66f78e*/
//	    rankEnabled = Decode1              // 0 => zeros; else DecodeBuffer(16)/*0x66f79b*/
//	        if rankEnabled: rank/rankMove/jobRank/jobRankMove (DecodeBuffer 16)/*0x66f7b6*/
//	hasPic = Decode1 (m_bLoginOpt)         //                                 /*0x66f815*/
//	_      = Decode1 (m_bQuerySSN...)      // JMS extra byte                  /*0x66f822*/
//	slots  = Decode4 (m_nSlotCount)        //                                 /*0x66f832*/
//	nBuyCharCount = Decode4 (m_nBuyChar..) // read unconditionally in jms     /*0x66f83f*/
//
// JMS structural deltas vs GMS (all IDA-confirmed against the Atlas codec
// list.go JMS branch and character_statistics.go JMS branch):
//   (1) A leading empty ASCII string is read after the status byte (DecodeStr
//       @0x66f72e). list.go writes WriteAsciiString("") for JMS.
//   (2) An extra byte (m_bQuerySSNOnCreateNewCharacter) sits between hasPic and
//       slots (Decode1 @0x66f822). list.go writes WriteByte(0) for JMS.
//   (3) nBuyCharCount is read unconditionally (Decode4 @0x66f83f). list.go writes
//       WriteInt(0) for JMS.
//
// GW_CharacterStat::Decode (jms @0x50ec17, list path bBackwardUpdate=0):
// id=Decode4 /*0x50ec35*/, name=DecodeBuffer(13) /*0x50ec4b*/, gender=Decode1
// /*0x50ec5d*/, skin=Decode1 /*0x50ec72*/, face=Decode4 /*0x50ec87*/, hair=Decode4
// /*0x50ec9c*/, petLockerSN=DecodeBuffer(24) /*0x50ecac*/, level=Decode1 /*0x50ecb3*/,
// 5x Decode2 job/str/dex/int/luk /*0x50ecc7..0x50ed19*/, HP/MaxHP/MP/MaxMP = 4x
// Decode2 (int16, NOT v95-widened — jms reads shorts) /*0x50ed2d/41/55/69*/, ap=Decode2
// /*0x50ed7d*/, sp=Decode2 (non-extendSP job path) /*0x50edd2*/, exp=Decode4 /*0x50edf7*/,
// fame=Decode2 /*0x50ee11*/, gachaExp=Decode4 /*0x50ee2b*/, mapId=Decode4 /*0x50ee45*/,
// spawnPoint=Decode1 /*0x50ee58*/, then the jms-extra tail: Decode2 /*0x50ee65*/ +
// DecodeBuffer(8) /*0x50ee7c*/ + nPlaytime Decode4 /*0x50ee83*/ + Decode4 /*0x50ee90*/
// + Decode4 /*0x50ee9d*/. character_statistics.go JMS branch writes
// WriteShort(0)+WriteLong(0)+3xWriteInt(0) — byte-exact match (2+8+4+4+4).
//
// AvatarLook::Decode (jms @0x51517e): gender=Decode1 /*0x51518a*/, skin=Decode1
// /*0x515194*/, face=Decode4 /*0x5151a1*/, !mega=Decode1 /*0x5151ce*/, hair=Decode4
// /*0x5151d5*/, equip loop (key Decode1 0xFF term /*0x5151de*/; value Decode4
// /*0x5151ec*/), masked loop (key Decode1 0xFF term /*0x515215*/; value Decode4
// /*0x515223*/), cashWeapon=Decode4 /*0x515251*/, pets=DecodeBuffer(12) /*0x515264*/.
//
// The jms wire matches the Atlas codec exactly — no per-version codec delta. HP/MP
// stay int16 (jms is NOT the v95-widened path) and the jms tail is fully consumed.
//
// packet-audit:verify packet=character/clientbound/CharacterList version=jms_v185 ida=0x66f3d8
func TestCharacterListByteOutputJMS(t *testing.T) {
	jms := pt.Variants[4] // JMS v185
	ctx := pt.CreateContext(jms.Region, jms.MajorVersion, jms.MinorVersion)

	stats := model.NewCharacterStatistics(
		0x01020304,         // id
		"Hero",             // name (padded to 13)
		0,                  // gender
		0,                  // skinColor
		0x4D2,              // face
		0x7B,               // hair
		[3]uint64{0, 0, 0}, // petIds
		0x0A,               // level
		0x64,               // jobId
		4, 5, 6, 7,         // str, dex, int, luck
		0x64, 0x64, 0x32, 0x32, // hp, maxHp, mp, maxMp
		3,      // ap
		false,  // hasSPTable (write sp short)
		2,      // sp
		0,      // experience
		8,      // fame
		0,      // gachaponExperience
		0x0BB8, // mapId
		0,      // spawnPoint
	)
	avatar := model.NewAvatar(0, 0, 0x4D2, false, 0x7B, nil, nil, nil)
	entry := model.NewCharacterListEntry(stats, avatar, false /*viewAll*/, false /*gm*/, 1, 2, 3, 4)

	input := NewCharacterList(0 /*status*/, []model.CharacterListEntry{entry}, false /*hasPic*/, 8 /*slots*/)
	got := input.Encode(nil, ctx)(nil)

	want := []byte{
		0x00,       // status (Decode1)                                  /*0x66f411*/
		0x00, 0x00, // JMS leading ASCII string len = 0 (DecodeStr)      /*0x66f72e*/
		0x01,       // count = 1 (Decode1)                               /*0x66f73d*/

		// --- GW_CharacterStat block (entry 0) --- @0x50ec17
		0x04, 0x03, 0x02, 0x01, // id = 0x01020304 (Decode4)       /*0x50ec35*/
		0x48, 0x65, 0x72, 0x6f, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // "Hero"+pad to 13 /*0x50ec4b*/
		0x00,                   // gender (Decode1)                /*0x50ec5d*/
		0x00,                   // skin (Decode1)                  /*0x50ec72*/
		0xd2, 0x04, 0x00, 0x00, // face (Decode4)                  /*0x50ec87*/
		0x7b, 0x00, 0x00, 0x00, // hair (Decode4)                  /*0x50ec9c*/
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // pet long 0 (DecodeBuffer 24) /*0x50ecac*/
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // pet long 1
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // pet long 2
		0x0a,       // level (Decode1)                 /*0x50ecb3*/
		0x64, 0x00, // jobId (Decode2)                 /*0x50ecc7*/
		0x04, 0x00, // str (Decode2)                   /*0x50ecdd*/
		0x05, 0x00, // dex                             /*0x50ecf1*/
		0x06, 0x00, // int                             /*0x50ed05*/
		0x07, 0x00, // luck                            /*0x50ed19*/
		0x64, 0x00, // hp (Decode2, int16 — jms)       /*0x50ed2d*/
		0x64, 0x00, // maxHp (Decode2)                 /*0x50ed41*/
		0x32, 0x00, // mp (Decode2)                    /*0x50ed55*/
		0x32, 0x00, // maxMp (Decode2)                 /*0x50ed69*/
		0x03, 0x00, // ap (Decode2)                    /*0x50ed7d*/
		0x02, 0x00, // sp (Decode2, non-extendSP job)  /*0x50edd2*/
		0x00, 0x00, 0x00, 0x00, // experience (Decode4)            /*0x50edf7*/
		0x08, 0x00,             // fame (Decode2)                  /*0x50ee11*/
		0x00, 0x00, 0x00, 0x00, // gachaExp (Decode4)              /*0x50ee2b*/
		0xb8, 0x0b, 0x00, 0x00, // mapId (Decode4)                 /*0x50ee45*/
		0x00,                   // spawnPoint (Decode1)            /*0x50ee58*/
		0x00, 0x00,             // jms tail short (Decode2)        /*0x50ee65*/
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // jms tail buf8 (DecodeBuffer 8) /*0x50ee7c*/
		0x00, 0x00, 0x00, 0x00, // jms nPlaytime (Decode4)         /*0x50ee83*/
		0x00, 0x00, 0x00, 0x00, // jms extra int (Decode4)         /*0x50ee90*/
		0x00, 0x00, 0x00, 0x00, // jms extra int (Decode4)         /*0x50ee9d*/

		// --- AvatarLook block (entry 0) --- @0x51517e
		0x00,                   // gender                          /*0x51518a*/
		0x00,                   // skin                            /*0x515194*/
		0xd2, 0x04, 0x00, 0x00, // face                            /*0x5151a1*/
		0x01,                   // !mega -> WriteBool(true)        /*0x5151ce*/
		0x7b, 0x00, 0x00, 0x00, // hair                            /*0x5151d5*/
		0xff,                   // equip terminator                /*0x5151de*/
		0xff,                   // masked terminator               /*0x515215*/
		0x00, 0x00, 0x00, 0x00, // cash weapon                     /*0x515251*/
		0x00, 0x00, 0x00, 0x00, // pet 0 (DecodeBuffer 12)         /*0x515264*/
		0x00, 0x00, 0x00, 0x00, // pet 1
		0x00, 0x00, 0x00, 0x00, // pet 2

		// --- entry trailer ---
		0x00,                   // family/viewAll flag (Decode1)   /*0x66f78e*/
		0x01,                   // rankEnabled = !gm (Decode1)     /*0x66f79b*/
		0x01, 0x00, 0x00, 0x00, // rank (Decode4)                  /*0x66f7b6 DecodeBuffer 16*/
		0x02, 0x00, 0x00, 0x00, // rankMove
		0x03, 0x00, 0x00, 0x00, // jobRank
		0x04, 0x00, 0x00, 0x00, // jobRankMove

		// --- list trailer ---
		0x00,                   // hasPic (Decode1)                /*0x66f815*/
		0x00,                   // m_bQuerySSN... (Decode1, jms)   /*0x66f822*/
		0x08, 0x00, 0x00, 0x00, // slots (Decode4)                 /*0x66f832*/
		0x00, 0x00, 0x00, 0x00, // nBuyCharCount (Decode4, jms)    /*0x66f83f*/
	}
	if !bytes.Equal(got, want) {
		t.Errorf("character-list jms bytes:\n got %x\nwant %x", got, want)
	}
}

// CharacterList v48 byte-fixture (GMS_v48_1_DEVM.exe, port 13337).
//
// Client read order — legacy char-list decoder sub_5013ED @0x5013ed, the
// world-select-success path (!v4 || v4==12 || v4==23, LABEL_29 @0x501558):
//
//	status = Decode1                       // result/status byte (earlier @0x501424)
//	count  = Decode1 (v12)                 // number of avatar entries         /*0x50158a*/
//	for each of 3 fixed slots (v16<48):    // loop @0x501626; entries<count decoded
//	    GW_CharacterStat::Decode(a2,0)     // statistics block  sub_49B627     /*0x5015b0*/
//	    AvatarLook::Decode                 // avatar/look block sub_49E1E0     /*0x5015be*/
//	    rankEnabled = Decode1              // 0 => memset 16; else DecodeBuffer(16) /*0x5015c6*/
//	// loop ENDS and returns — v48 reads NO family byte, NO hasPic, NO trailing
//	// slot-count Decode4 (the slot count entered the char-list at v61,
//	// sub_56688D @0x566b02).
//
// GW_CharacterStat::Decode (v48 sub_49B627 @0x49b627, list path a3=0): id=Decode4
// /*0x49b64a*/, name=DecodeBuffer(13) /*0x49b65b*/, gender=Decode1 /*0x49b672*/,
// skin=Decode1 /*0x49b687*/, face=Decode4 /*0x49b69c*/, hair=Decode4 /*0x49b6b1*/,
// petLockerSN=DecodeBuffer(8) — a SINGLE 8-byte pet, NOT 3 longs /*0x49b6bc*/,
// level=Decode1 /*0x49b6cb*/, 11x Decode2 (job,str,dex,int,luk,hp,maxHp,mp,maxMp,
// ap,sp) /*0x49b6d7..0x49b7b0*/, exp=Decode4 /*0x49b7b3*/, fame=Decode2 /*0x49b7cd*/,
// mapId=Decode4 /*0x49b7e7*/, spawnPoint=Decode1 /*0x49b801*/. NO gachaExp, NO
// trailing int, NO nSubJob.
//
// AvatarLook::Decode (v48 sub_49E1E0 @0x49e1e0): gender=Decode1 /*0x49e1f3*/,
// skin=Decode1 /*0x49e1ff*/, face=Decode4 /*0x49e20f*/, !mega=Decode1 /*0x49e22c*/,
// hair=Decode4 /*0x49e238*/, equip loop (0xFF term) /*0x49e241*/, masked loop
// (0xFF term) /*0x49e278*/, cashWeapon=Decode4 /*0x49e2b6*/, pet=Decode4 — a
// SINGLE 4-byte pet int, NOT DecodeBuffer(12) /*0x49e2b9*/.
//
// packet-audit:verify packet=character/clientbound/CharacterList version=gms_v48 ida=0x5013ed
func TestCharacterListByteOutputV48(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)

	stats := model.NewCharacterStatistics(
		0x01020304,         // id
		"Hero",             // name (padded to 13)
		0,                  // gender
		0,                  // skinColor
		0x4D2,              // face
		0x7B,               // hair
		[3]uint64{0, 0, 0}, // petIds
		0x0A,               // level
		0x64,               // jobId
		4, 5, 6, 7,         // str, dex, int, luck
		0x64, 0x64, 0x32, 0x32, // hp, maxHp, mp, maxMp
		3,      // ap
		false,  // hasSPTable (write sp short)
		2,      // sp
		0,      // experience
		8,      // fame
		0,      // gachaponExperience
		0x0BB8, // mapId
		0,      // spawnPoint
	)
	avatar := model.NewAvatar(0, 0, 0x4D2, false, 0x7B, nil, nil, nil)
	entry := model.NewCharacterListEntry(stats, avatar, false /*viewAll*/, false /*gm*/, 1, 2, 3, 4)

	input := NewCharacterList(0 /*status*/, []model.CharacterListEntry{entry}, false /*hasPic*/, 8 /*slots*/)
	got := input.Encode(nil, ctx)(nil)

	want := []byte{
		0x00, // status (Decode1)                                  /*0x501424*/
		0x01, // count = 1 (Decode1)                               /*0x50158a*/

		// --- GW_CharacterStat block (entry 0) --- sub_49B627 @0x49b627
		0x04, 0x03, 0x02, 0x01, // id = 0x01020304 (Decode4)       /*0x49b64a*/
		0x48, 0x65, 0x72, 0x6f, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // "Hero"+pad to 13 /*0x49b65b*/
		0x00,                   // gender (Decode1)                /*0x49b672*/
		0x00,                   // skin (Decode1)                  /*0x49b687*/
		0xd2, 0x04, 0x00, 0x00, // face (Decode4)                  /*0x49b69c*/
		0x7b, 0x00, 0x00, 0x00, // hair (Decode4)                  /*0x49b6b1*/
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // SINGLE pet long (DecodeBuffer 8) /*0x49b6bc*/
		0x0a,       // level (Decode1)                 /*0x49b6cb*/
		0x64, 0x00, // jobId (Decode2)                 /*0x49b6d7*/
		0x04, 0x00, // str (Decode2)                   /*0x49b6eb*/
		0x05, 0x00, // dex                             /*0x49b6ff*/
		0x06, 0x00, // int                             /*0x49b713*/
		0x07, 0x00, // luck                            /*0x49b727*/
		0x64, 0x00, // hp                              /*0x49b73b*/
		0x64, 0x00, // maxHp                           /*0x49b74f*/
		0x32, 0x00, // mp                              /*0x49b763*/
		0x32, 0x00, // maxMp                           /*0x49b777*/
		0x03, 0x00, // ap (Decode2)                    /*0x49b78b*/
		0x02, 0x00, // sp (Decode2, !hasSPTable)       /*0x49b79f*/
		0x00, 0x00, 0x00, 0x00, // experience (Decode4)            /*0x49b7b3*/
		0x08, 0x00,             // fame (Decode2)                  /*0x49b7cd*/
		0xb8, 0x0b, 0x00, 0x00, // mapId (Decode4)                 /*0x49b7e7*/
		0x00,                   // spawnPoint (Decode1)            /*0x49b801*/

		// --- AvatarLook block (entry 0) --- sub_49E1E0 @0x49e1e0
		0x00,                   // gender                          /*0x49e1f3*/
		0x00,                   // skin                            /*0x49e1ff*/
		0xd2, 0x04, 0x00, 0x00, // face                            /*0x49e20f*/
		0x01,                   // !mega -> WriteBool(true)        /*0x49e22c*/
		0x7b, 0x00, 0x00, 0x00, // hair                            /*0x49e238*/
		0xff,                   // equip terminator                /*0x49e241*/
		0xff,                   // masked terminator               /*0x49e278*/
		0x00, 0x00, 0x00, 0x00, // cash weapon                     /*0x49e2b6*/
		0x00, 0x00, 0x00, 0x00, // SINGLE pet int (Decode4)        /*0x49e2b9*/

		// --- entry trailer (no family byte: v48<73) ---
		0x01,                   // rankEnabled = !gm (Decode1)     /*0x5015c6*/
		0x01, 0x00, 0x00, 0x00, // rank (Decode4)                  /*0x5015e1 DecodeBuffer 16*/
		0x02, 0x00, 0x00, 0x00, // rankMove
		0x03, 0x00, 0x00, 0x00, // jobRank
		0x04, 0x00, 0x00, 0x00, // jobRankMove

		// --- NO list trailer: v48 reads no hasPic (<83), no slot count (<61) ---
	}
	if !bytes.Equal(got, want) {
		t.Errorf("character-list v48 bytes:\n got %x\nwant %x", got, want)
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
