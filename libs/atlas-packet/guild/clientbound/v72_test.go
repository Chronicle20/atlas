package clientbound

import (
	"bytes"
	"testing"
)

// v72 GUILD foreign-change family verification (GMS_v72.1_U_DEVM.exe, port 13339).
//
//   - GUILD_NAME_CHANGED (CUserRemote::OnGuildNameChanged @0x88cd17):
//     DecodeStr(name) — the charId is read by the user-pool router before
//     dispatch, exactly as v79. Read order byte-identical to the v79 fixture.
//   - GUILD_MARK_CHANGED (CUserRemote::OnGuildMarkChanged @0x88cd62):
//     Decode2(bg)+Decode1(bgColor)+Decode2(logo)+Decode1(logoColor) — byte-
//     identical to v79.
//
// Both handlers are single, non-dispatcher functions; their v72 read orders were
// decompiled and match v79/v83 exactly (no MajorVersion gate in the codec), so the
// expected bytes are hand-computed from the v72 decompile and equal the v79 fixture.

// packet-audit:verify packet=guild/clientbound/GuildForeignNameChanged version=gms_v72 ida=0x88cd17
// packet-audit:verify packet=guild/clientbound/GuildForeignEmblemChanged version=gms_v72 ida=0x88cd62
func TestGuildForeignChangedV72(t *testing.T) {
	// ForeignNameChanged: WriteInt(charId) + WriteAsciiString(name).
	// charId=1001 (e9 03 00 00) + "Bob" (03 00 'B' 'o' 'b').
	// (@0x88cd17: DecodeStr(name); charId supplied by the user-pool router.)
	gotName := NewForeignNameChanged(1001, "Bob").Encode(nil, nil)(nil)
	wantName := []byte{0xE9, 0x03, 0x00, 0x00, 0x03, 0x00, 0x42, 0x6F, 0x62}
	if !bytes.Equal(gotName, wantName) {
		t.Errorf("v72 ForeignNameChanged: got % x want % x", gotName, wantName)
	}
	// ForeignEmblemChanged: WriteInt(charId)+WriteShort(bg)+WriteByte(bgColor)+WriteShort(logo)+WriteByte(logoColor).
	// charId=1001, logo=3,logoColor=2,bg=5,bgColor=4 → e9 03 00 00 | 05 00 | 04 | 03 00 | 02.
	// (@0x88cd62: Decode2(bg)+Decode1(bgColor)+Decode2(logo)+Decode1(logoColor).)
	gotEmblem := NewForeignEmblemChanged(1001, 3, 2, 5, 4).Encode(nil, nil)(nil)
	wantEmblem := []byte{0xE9, 0x03, 0x00, 0x00, 0x05, 0x00, 0x04, 0x03, 0x00, 0x02}
	if !bytes.Equal(gotEmblem, wantEmblem) {
		t.Errorf("v72 ForeignEmblemChanged: got % x want % x", gotEmblem, wantEmblem)
	}
}
