package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=pet/clientbound/PetChat version=gms_v83 ida=0x70476e
// packet-audit:verify packet=pet/clientbound/PetChat version=gms_v87 ida=0x74844b
// packet-audit:verify packet=pet/clientbound/PetChat version=gms_v95 ida=0x6a3860
// packet-audit:verify packet=pet/clientbound/PetChat version=jms_v185 ida=0x76a557
// packet-audit:verify packet=pet/clientbound/PetChat version=gms_v84 ida=0x720e91
func TestPetChat(t *testing.T) {
	input := NewPetChat(1234, 0, 1, 5, "Hello!", true)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// v79 PET_CHAT (cb op 0xAB=171) read order, verified GMS_v79_1_DEVM.exe (port
// 13340): CUserPool::OnUserCommonPacket@0x8c8c79 Decode4(ownerId)@0x8c8c84 →
// CUser::OnPetPacket@0x892474 Decode1(slot)@0x8924b6 → CPet::OnAction@0x690eec:
// Decode1(nType)@0x690f1d, Decode1(nAction)@0x690f25, DecodeStr(msg)@0x690f2e,
// Decode1(balloon)@0x690f41. Wire = ownerId(4)+slot(1)+nType(1)+nAction(1)+
// msg(2+len)+balloon(1); byte-identical to v83 (codec version-unconditional).
// packet-audit:verify packet=pet/clientbound/PetChat version=gms_v79 ida=0x690eec
func TestPetChatBytesV79(t *testing.T) {
	ctx := test.CreateContext("GMS", 79, 1)
	got := NewPetChat(0x01020304, 0x05, 0x06, 0x07, "Hi", true).Encode(nil, ctx)(nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01, // ownerId Decode4@0x8c8c84 (LE)
		0x05,       // slot Decode1@0x8924b6
		0x06,       // nType Decode1@0x690f1d
		0x07,       // nAction Decode1@0x690f25
		0x02, 0x00, // msg length DecodeStr@0x690f2e
		0x48, 0x69, // "Hi"
		0x01, // balloon Decode1@0x690f41
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v79 = % X, want % X", got, want)
	}
}
