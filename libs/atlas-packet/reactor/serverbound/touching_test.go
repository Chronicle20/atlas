package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestTouchingRoundTrip verifies TouchingRequest round-trips both fields on
// every tenant variant. Unlike HitRequest, the touch body is version-invariant
// (see opcode-derivation.md: every in-scope version writes
// COutPacket(N); Encode4(dwID); Encode1(touching), no gate anywhere), so no
// version-conditional assertions are needed.
func TestTouchingRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)

			enteringInput := TouchingRequest{oid: 100, touching: true}
			enteringOutput := TouchingRequest{}
			pt.RoundTrip(t, ctx, enteringInput.Encode, enteringOutput.Decode, nil)
			if enteringOutput.Oid() != enteringInput.Oid() {
				t.Errorf("entering oid: got %v, want %v", enteringOutput.Oid(), enteringInput.Oid())
			}
			if enteringOutput.Touching() != true {
				t.Errorf("entering touching: got %v, want true", enteringOutput.Touching())
			}

			leavingInput := TouchingRequest{oid: 100, touching: false}
			leavingOutput := TouchingRequest{}
			pt.RoundTrip(t, ctx, leavingInput.Encode, leavingOutput.Decode, nil)
			if leavingOutput.Oid() != leavingInput.Oid() {
				t.Errorf("leaving oid: got %v, want %v", leavingOutput.Oid(), leavingInput.Oid())
			}
			if leavingOutput.Touching() != false {
				t.Errorf("leaving touching: got %v, want false", leavingOutput.Touching())
			}
		})
	}
}

// TestTouchingBytes pins the exact wire bytes for the touch-notification body.
// Per docs/tasks/task-249-touch-activated-reactors/opcode-derivation.md, every
// in-scope version's CReactorPool::FindTouchReactorAroundLocalUser send site
// is COutPacket(N); Encode4(dwID); Encode1(touching) with the opcode N being
// the only per-version difference (196/198/206/212/219/243/250/217 across
// gms_v72/79/83/84/87/92/95 and jms_v185 respectively). The body bytes below
// (oid + touching flag) are therefore identical for every variant; this table
// pins them on a representative pair plus a legacy/JMS spot-check.
//
// packet-audit:verify packet=reactor/serverbound/ReactorTouchingRequest version=gms_v72 ida=0x692bb0
// packet-audit:verify packet=reactor/serverbound/ReactorTouchingRequest version=gms_v79 ida=0x6b8362
// packet-audit:verify packet=reactor/serverbound/ReactorTouchingRequest version=gms_v83 ida=0x735D90
// packet-audit:verify packet=reactor/serverbound/ReactorTouchingRequest version=gms_v84 ida=0x753378
// packet-audit:verify packet=reactor/serverbound/ReactorTouchingRequest version=gms_v87 ida=0x77bca7
// packet-audit:verify packet=reactor/serverbound/ReactorTouchingRequest version=gms_v92 ida=0x6C1630
// packet-audit:verify packet=reactor/serverbound/ReactorTouchingRequest version=gms_v95 ida=0x6cded0
// packet-audit:verify packet=reactor/serverbound/ReactorTouchingRequest version=jms_v185 ida=0x79f0aa
func TestTouchingBytes(t *testing.T) {
	t.Run("entering", func(t *testing.T) {
		ctx := pt.CreateContext("GMS", 83, 1)
		input := TouchingRequest{oid: 100, touching: true}
		got := pt.Encode(t, ctx, input.Encode, nil)
		want := []byte{0x64, 0x00, 0x00, 0x00, 0x01}
		if !bytes.Equal(got, want) {
			t.Errorf("entering touching bytes:\n got % x\nwant % x", got, want)
		}
	})

	t.Run("leaving", func(t *testing.T) {
		ctx := pt.CreateContext("GMS", 83, 1)
		input := TouchingRequest{oid: 100, touching: false}
		got := pt.Encode(t, ctx, input.Encode, nil)
		want := []byte{0x64, 0x00, 0x00, 0x00, 0x00}
		if !bytes.Equal(got, want) {
			t.Errorf("leaving touching bytes:\n got % x\nwant % x", got, want)
		}
	})

	t.Run("entering_v95", func(t *testing.T) {
		ctx := pt.CreateContext("GMS", 95, 1)
		input := TouchingRequest{oid: 100, touching: true}
		got := pt.Encode(t, ctx, input.Encode, nil)
		want := []byte{0x64, 0x00, 0x00, 0x00, 0x01}
		if !bytes.Equal(got, want) {
			t.Errorf("entering_v95 touching bytes:\n got % x\nwant % x", got, want)
		}
	})

	t.Run("leaving_jms", func(t *testing.T) {
		ctx := pt.CreateContext("JMS", 185, 1)
		input := TouchingRequest{oid: 100, touching: false}
		got := pt.Encode(t, ctx, input.Encode, nil)
		want := []byte{0x64, 0x00, 0x00, 0x00, 0x00}
		if !bytes.Equal(got, want) {
			t.Errorf("leaving_jms touching bytes:\n got % x\nwant % x", got, want)
		}
	})
}
