package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/serverbound/FieldAdminCommand version=gms_v72 ida=0x50bf7c
// packet-audit:verify packet=field/serverbound/FieldAdminCommand version=gms_v79 ida=0x51803a
// packet-audit:verify packet=field/serverbound/FieldAdminCommand version=gms_v83 ida=0x52c958
// packet-audit:verify packet=field/serverbound/FieldAdminCommand version=gms_v84 ida=0x53891a
// packet-audit:verify packet=field/serverbound/FieldAdminCommand version=gms_v87 ida=0x5531b8
// packet-audit:verify packet=field/serverbound/FieldAdminCommand version=gms_v95 ida=0x540fbe
// packet-audit:verify packet=field/serverbound/FieldAdminCommand version=jms_v185 ida=0x568ac2
// TestAdminCommandByteOutputV79 pins the gms_v79 ADMIN_COMMAND (op 0x7D)
// serverbound wire. IDA: CField::SendChatMsgSlash send-site @0x51803a
// (GMS_v79_1_DEVM.exe) — COutPacket(0x7D) @0x51803f then Encode1(subCommand)
// @0x518044 (the leading sub-command byte; remaining payload is per-subcommand
// and modeled decode-and-log).
func TestAdminCommandByteOutputV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	input := NewAdminCommand(0x23)
	expected := []byte{0x23}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v79 admin_command golden mismatch: got %v want %v", actual, expected)
	}
}

// TestAdminCommandByteOutputV72 pins the gms_v72 ADMIN_COMMAND (op 0x7E)
// serverbound wire. IDA: CField::SendChatMsgSlash send-site @0x50bf7c
// (GMS_v72.1_U_DEVM.exe) — one of many COutPacket(0x7E) send-sites, each leading
// with a single sub-command byte: push 0x7E; ctor @0x50bf84 then Encode1
// @0x50bf95 (the leading sub-command byte; remaining payload is per-subcommand
// and modeled decode-and-log). Body == v79 legacy wire.
func TestAdminCommandByteOutputV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	input := NewAdminCommand(0x23)
	expected := []byte{0x23}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v72 admin_command golden mismatch: got %v want %v", actual, expected)
	}
}

// TestAdminCommandByteOutputV61 pins the gms_v61 ADMIN_COMMAND (op 0x7E = 126)
// serverbound wire. IDA: CField::SendChatMsgSlash#AdminCommand = sub_80C896
// @0x80c896 (GMS_v61.1_U_DEVM.exe) builds COutPacket(126) + Encode1(subCommand)
// (this send-site uses subcommand 8; the codec models the leading sub-command byte
// decode-and-log). GM slash-command opcode = v72 ADMIN_COMMAND=126 (Δ0).
// packet-audit:verify packet=field/serverbound/FieldAdminCommand version=gms_v61 ida=0x80c896
func TestAdminCommandByteOutputV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	input := NewAdminCommand(0x23)
	expected := []byte{0x23}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v61 admin_command golden mismatch: got %v want %v", actual, expected)
	}
}

func TestAdminCommandGolden(t *testing.T) {
	input := NewAdminCommand(0x05)
	ctx := pt.CreateContext("GMS", 83, 1)
	expected := []byte{0x05}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestAdminCommandRoundTrip(t *testing.T) {
	input := NewAdminCommand(0x05)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := AdminCommand{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.SubCommand() != input.SubCommand() {
				t.Errorf("round-trip mismatch: got %+v want %+v", output, input)
			}
		})
	}
}
