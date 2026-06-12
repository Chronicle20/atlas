package clientbound

import (
	"context"
	"encoding/binary"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
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
func TestSayImageConversationDetailEncode(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			d := &SayImageConversationDetail{Images: []string{"img/a", "img/b"}}
			got := d.Encode(l, ctx)(nil)

			want := []byte{byte(2)}
			want = append(want, asciiBytes("img/a")...)
			want = append(want, asciiBytes("img/b")...)
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
func TestAskMemberShopAvatarConversationDetailEncode(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			d := &AskMemberShopAvatarConversationDetail{Message: "pick one", Candidates: []uint32{0x11223344, 0x55667788}}
			got := d.Encode(l, ctx)(nil)

			want := asciiBytes("pick one")
			want = append(want, byte(2))
			want = append(want, 0x44, 0x33, 0x22, 0x11)
			want = append(want, 0x88, 0x77, 0x66, 0x55)
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
