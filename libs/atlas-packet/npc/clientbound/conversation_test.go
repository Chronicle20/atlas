package clientbound

import (
	"context"
	"encoding/binary"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// asciiBytes returns the on-wire encoding of an ASCII string: a 2-byte
// little-endian length prefix followed by the raw bytes. Mirrors
// response.Writer.WriteAsciiString for plain-ASCII inputs.
func asciiBytes(s string) []byte {
	out := make([]byte, 2+len(s))
	binary.LittleEndian.PutUint16(out[:2], uint16(len(s)))
	copy(out[2:], s)
	return out
}

// packet-audit:verify packet=npc/clientbound/NpcAskSlideMenuConversationDetail version=gms_v83 ida=0x76b5c8
// packet-audit:verify packet=npc/clientbound/NpcAskSlideMenuConversationDetail version=gms_v87 ida=0x792bb4
// packet-audit:verify packet=npc/clientbound/NpcAskSlideMenuConversationDetail version=gms_v95 ida=0x6dbe50
// packet-audit:verify packet=npc/clientbound/NpcAskAvatarConversationDetail version=gms_v83 ida=0x74713d
// packet-audit:verify packet=npc/clientbound/NpcAskBoxTextConversationDetail version=gms_v83 ida=0x746c46
// packet-audit:verify packet=npc/clientbound/NpcAskMemberShopAvatarConversationDetail version=gms_v83 ida=0x74730b
// packet-audit:verify packet=npc/clientbound/NpcAskMenuConversationDetail version=gms_v83 ida=0x746fad
// packet-audit:verify packet=npc/clientbound/NpcAskNumberConversationDetail version=gms_v83 ida=0x746dff
// packet-audit:verify packet=npc/clientbound/NpcAskPetAllConversationDetail version=gms_v83 ida=0x74775c
// packet-audit:verify packet=npc/clientbound/NpcAskPetConversationDetail version=gms_v83 ida=0x7474a2
// packet-audit:verify packet=npc/clientbound/NpcAskQuizConversationDetail version=gms_v83 ida=0xa26b09
// packet-audit:verify packet=npc/clientbound/NpcAskSpeedQuizConversationDetail version=gms_v83 ida=0xa26c66
// packet-audit:verify packet=npc/clientbound/NpcAskTextConversationDetail version=gms_v83 ida=0x746a8b
// packet-audit:verify packet=npc/clientbound/NpcAskYesNoConversationDetail version=gms_v83 ida=0x74692c
// packet-audit:verify packet=npc/clientbound/NpcNpcConversation version=gms_v83 ida=0x74660a
// packet-audit:verify packet=npc/clientbound/NpcSayConversationDetail version=gms_v83 ida=0x7467ab
// packet-audit:verify packet=npc/clientbound/NpcSayImageConversationDetail version=gms_v83 ida=0x961275
// packet-audit:verify packet=npc/clientbound/NpcAskAvatarConversationDetail version=gms_v87 ida=0x792330
// packet-audit:verify packet=npc/clientbound/NpcAskBoxTextConversationDetail version=gms_v87 ida=0x791e79
// packet-audit:verify packet=npc/clientbound/NpcAskMemberShopAvatarConversationDetail version=gms_v87 ida=0x7924cc
// packet-audit:verify packet=npc/clientbound/NpcAskMenuConversationDetail version=gms_v87 ida=0x7921a8
// packet-audit:verify packet=npc/clientbound/NpcAskNumberConversationDetail version=gms_v87 ida=0x792020
// packet-audit:verify packet=npc/clientbound/NpcAskPetAllConversationDetail version=gms_v87 ida=0x7928f1
// packet-audit:verify packet=npc/clientbound/NpcAskPetConversationDetail version=gms_v87 ida=0x792663
// packet-audit:verify packet=npc/clientbound/NpcAskQuizConversationDetail version=gms_v87 ida=0x792b90
// packet-audit:verify packet=npc/clientbound/NpcAskSpeedQuizConversationDetail version=gms_v87 ida=0x792ba2
// packet-audit:verify packet=npc/clientbound/NpcAskTextConversationDetail version=gms_v87 ida=0x791cd0
// packet-audit:verify packet=npc/clientbound/NpcAskYesNoConversationDetail version=gms_v87 ida=0x791b70
// packet-audit:verify packet=npc/clientbound/NpcNpcConversation version=gms_v87 ida=0x791666
// packet-audit:verify packet=npc/clientbound/NpcSayConversationDetail version=gms_v87 ida=0x791828
// packet-audit:verify packet=npc/clientbound/NpcSayImageConversationDetail version=gms_v87 ida=0x7919a9
// packet-audit:verify packet=npc/clientbound/NpcAskAvatarConversationDetail version=gms_v95 ida=0x6dcff0
// packet-audit:verify packet=npc/clientbound/NpcAskBoxTextConversationDetail version=gms_v95 ida=0x6dc9c0
// packet-audit:verify packet=npc/clientbound/NpcAskMenuConversationDetail version=gms_v95 ida=0x6dce00
// packet-audit:verify packet=npc/clientbound/NpcAskNumberConversationDetail version=gms_v95 ida=0x6dcc00
// packet-audit:verify packet=npc/clientbound/NpcAskPetAllConversationDetail version=gms_v95 ida=0x6ddbe0
// packet-audit:verify packet=npc/clientbound/NpcAskPetConversationDetail version=gms_v95 ida=0x6dd6e0
// packet-audit:verify packet=npc/clientbound/NpcAskQuizConversationDetail version=gms_v95 ida=0x9ffad0
// packet-audit:verify packet=npc/clientbound/NpcAskSpeedQuizConversationDetail version=gms_v95 ida=0x9f1d50
// packet-audit:verify packet=npc/clientbound/NpcAskTextConversationDetail version=gms_v95 ida=0x6dc790
// packet-audit:verify packet=npc/clientbound/NpcAskYesNoConversationDetail version=gms_v95 ida=0x6dc5a0
// packet-audit:verify packet=npc/clientbound/NpcNpcConversation version=gms_v95 ida=0x6de0f0
// packet-audit:verify packet=npc/clientbound/NpcSayConversationDetail version=gms_v95 ida=0x6dc110
// packet-audit:verify packet=npc/clientbound/NpcAskAvatarConversationDetail version=jms_v185 ida=0x7b7e1d
// packet-audit:verify packet=npc/clientbound/NpcAskBoxTextConversationDetail version=jms_v185 ida=0x7b7966
// packet-audit:verify packet=npc/clientbound/NpcAskMenuConversationDetail version=jms_v185 ida=0x7b7c95
// packet-audit:verify packet=npc/clientbound/NpcAskNumberConversationDetail version=jms_v185 ida=0x7b7b0d
// packet-audit:verify packet=npc/clientbound/NpcAskPetAllConversationDetail version=jms_v185 ida=0x7b8250
// packet-audit:verify packet=npc/clientbound/NpcAskPetConversationDetail version=jms_v185 ida=0x7b7fc2
// packet-audit:verify packet=npc/clientbound/NpcAskQuizConversationDetail version=jms_v185 ida=0x7b84ef
// packet-audit:verify packet=npc/clientbound/NpcAskSpeedQuizConversationDetail version=jms_v185 ida=0x7b8501
// packet-audit:verify packet=npc/clientbound/NpcAskTextConversationDetail version=jms_v185 ida=0x7b77bd
// packet-audit:verify packet=npc/clientbound/NpcAskYesNoConversationDetail version=jms_v185 ida=0x7b765d
// packet-audit:verify packet=npc/clientbound/NpcNpcConversation version=jms_v185 ida=0x7b7160
// packet-audit:verify packet=npc/clientbound/NpcSayConversationDetail version=jms_v185 ida=0x7b7315
// packet-audit:verify packet=npc/clientbound/NpcSayImageConversationDetail version=jms_v185 ida=0x7b7496
// packet-audit:verify packet=npc/clientbound/NpcNpcConversation version=gms_v84 ida=0x76850a
// packet-audit:verify packet=npc/clientbound/NpcAskAvatarConversationDetail version=gms_v84 ida=0x76921d
// packet-audit:verify packet=npc/clientbound/NpcAskBoxTextConversationDetail version=gms_v84 ida=0x768d26
// packet-audit:verify packet=npc/clientbound/NpcAskMemberShopAvatarConversationDetail version=gms_v84 ida=0x7693eb
// packet-audit:verify packet=npc/clientbound/NpcAskNumberConversationDetail version=gms_v84 ida=0x768edf
// packet-audit:verify packet=npc/clientbound/NpcAskPetAllConversationDetail version=gms_v84 ida=0x76983c
// packet-audit:verify packet=npc/clientbound/NpcAskPetConversationDetail version=gms_v84 ida=0x769582
// packet-audit:verify packet=npc/clientbound/NpcAskQuizConversationDetail version=gms_v84 ida=0xa722bf
// packet-audit:verify packet=npc/clientbound/NpcAskSlideMenuConversationDetail version=gms_v84 ida=0x769b26
// packet-audit:verify packet=npc/clientbound/NpcAskSpeedQuizConversationDetail version=gms_v84 ida=0xa7241c
// packet-audit:verify packet=npc/clientbound/NpcAskYesNoConversationDetail version=gms_v84 ida=0x768a0b
// packet-audit:verify packet=npc/clientbound/NpcSayImageConversationDetail version=gms_v84 ida=0x768844
func TestNpcConversationSay(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	detail := &SayConversationDetail{Message: "Hello adventurer!", Next: true, Previous: false}
	detailBytes := detail.Encode(l, context.Background())(nil)

	input := NewNpcConversation(0, 2100, 0, 0, 0, detailBytes)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestNpcConversationSayWithSecondaryNpc(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	detail := &SayConversationDetail{Message: "Two NPCs talking!", Next: true, Previous: true}
	detailBytes := detail.Encode(l, context.Background())(nil)

	// param = 4 triggers writing secondaryNpcTemplateId
	input := NewNpcConversation(0, 2100, 0, 4, 9999, detailBytes)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestNpcConversationAskMenu(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	detail := &AskMenuConversationDetail{Message: "#L0#Option 1#l\r\n#L1#Option 2#l"}
	detailBytes := detail.Encode(l, context.Background())(nil)

	input := NewNpcConversation(0, 2100, 5, 0, 0, detailBytes)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// TestSayImageConversationDetailEncode verifies the image-count prefix is a
// single byte, matching CScriptMan::OnSayImage@0x6dc310 which reads the count
// via CInPacket::Decode1 (line 61, 0x6dc3d9) before looping DecodeStr.
// packet-audit:verify packet=npc/clientbound/NpcSayImageConversationDetail version=gms_v95 ida=0x6dc310
func TestSayImageConversationDetailEncode(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			d := &SayImageConversationDetail{Images: []string{"img/a", "img/b"}}
			got := d.Encode(l, ctx)(nil)

			var want []byte
			// GMS <83 (v79 etc.): CScriptMan::OnSayImage @0x6c8052 reads a single
			// DecodeStr (one image, no count). v83+/JMS read Decode1 count + list.
			if v.Region == "GMS" && v.MajorVersion < 83 {
				want = asciiBytes("img/a")
			} else {
				want = []byte{byte(2)}
				want = append(want, asciiBytes("img/a")...)
				want = append(want, asciiBytes("img/b")...)
			}
			if !bytesEqual(got, want) {
				t.Errorf("SayImage encode mismatch\n got=%v\nwant=%v", got, want)
			}
		})
	}
}

// TestAskMemberShopAvatarConversationDetailEncode verifies the candidate-count
// prefix is a single byte, matching CScriptMan::OnAskMembershopAvatar@0x6dd340
// (case 9) which reads the count via CInPacket::Decode1 (line 55, 0x6dd394)
// before looping Decode4 per candidate.
// packet-audit:verify packet=npc/clientbound/NpcAskMemberShopAvatarConversationDetail version=gms_v95 ida=0x6dd340
func TestAskMemberShopAvatarConversationDetailEncode(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			d := &AskMemberShopAvatarConversationDetail{Message: "pick one", Candidates: []uint32{0x11223344, 0x55667788}}
			got := d.Encode(l, ctx)(nil)

			want := asciiBytes("pick one")
			// GMS <83 (v79 etc.): the client reads count + (int64 SN + byte) per
			// entry (CScriptMan::OnAskMembershopAvatar @0x6c8bc8), incompatible
			// with the v83+ int32 style-id list — Atlas gates count=0. v83+ keeps
			// the int32 candidate list.
			if v.Region == "GMS" && v.MajorVersion < 83 {
				want = append(want, byte(0))
			} else {
				want = append(want, byte(2))
				want = append(want, 0x44, 0x33, 0x22, 0x11)
				want = append(want, 0x88, 0x77, 0x66, 0x55)
			}
			if !bytesEqual(got, want) {
				t.Errorf("AskMemberShopAvatar encode mismatch\n got=%v\nwant=%v", got, want)
			}
		})
	}
}

// TestAskSlideMenuConversationDetailEncode verifies the leading slideDlgType
// int is written for GMS v87+ and for JMS185, and omitted for GMS v83..86
// (v84..86 == v83, off-by-one fix, delta §3.2).
// JMS185 sub_7E2A97@0x7e2a97 reads two leading Decode4s (slideDlgType + menuType)
// then DecodeStr(message) unconditionally; GMS v83 reads a single Decode4.
// packet-audit:verify packet=npc/clientbound/NpcAskSlideMenuConversationDetail version=jms_v185 ida=0x7b8513
func TestAskSlideMenuConversationDetailEncode(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	intBytes := func(v uint32) []byte {
		out := make([]byte, 4)
		binary.LittleEndian.PutUint32(out, v)
		return out
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			d := &AskSlideMenuConversationDetail{Unknown: true, MenuType: 0x000000AA, Message: "slide"}
			got := d.Encode(l, ctx)(nil)

			leadingPresent := (v.Region == "GMS" && v.MajorVersion >= 87) || v.Region == "JMS"
			var want []byte
			if leadingPresent {
				want = append(want, intBytes(1)...) // slideDlgType (Unknown=true)
			}
			want = append(want, intBytes(0x000000AA)...) // menuType
			want = append(want, asciiBytes("slide")...)
			if !bytesEqual(got, want) {
				t.Errorf("AskSlideMenu encode mismatch (leading=%v)\n got=%v\nwant=%v", leadingPresent, got, want)
			}
		})
	}
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestNpcConversationAccessors(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	detail := &SayConversationDetail{Message: "test", Next: true, Previous: false}
	detailBytes := detail.Encode(l, context.Background())(nil)

	m := NewNpcConversation(1, 2100, 3, 4, 5000, detailBytes)
	if m.SpeakerTypeId() != 1 {
		t.Errorf("expected SpeakerTypeId 1, got %d", m.SpeakerTypeId())
	}
	if m.SpeakerTemplateId() != 2100 {
		t.Errorf("expected SpeakerTemplateId 2100, got %d", m.SpeakerTemplateId())
	}
	if m.MsgType() != 3 {
		t.Errorf("expected MsgType 3, got %d", m.MsgType())
	}
	if m.Param() != 4 {
		t.Errorf("expected Param 4, got %d", m.Param())
	}
	if m.SecondaryNpcTemplateId() != 5000 {
		t.Errorf("expected SecondaryNpcTemplateId 5000, got %d", m.SecondaryNpcTemplateId())
	}
	if m.Operation() != NpcConversationWriter {
		t.Errorf("expected Operation %s, got %s", NpcConversationWriter, m.Operation())
	}
}
