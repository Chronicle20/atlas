package clientbound

import (
	"bytes"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// packet-audit:verify packet=character/clientbound/BuffGive version=gms_v72 ida=0x918b24
// packet-audit:verify packet=character/clientbound/BuffGiveForeign version=gms_v72 ida=0x88cb10
// packet-audit:verify packet=character/clientbound/BuffGive version=gms_v79 ida=0x96a6d1
// packet-audit:verify packet=character/clientbound/BuffGiveForeign version=gms_v79 ida=0x8d9a03
// packet-audit:verify packet=character/clientbound/BuffGive version=gms_v87 ida=0xab77ff
// packet-audit:verify packet=character/clientbound/BuffGive version=gms_v95 ida=0xa02fc0
// packet-audit:verify packet=character/clientbound/BuffGive version=gms_v83 ida=0xa202be
// packet-audit:verify packet=character/clientbound/BuffGiveForeign version=gms_v83 ida=0x98385d
// packet-audit:verify packet=character/clientbound/BuffGiveForeign version=gms_v87 ida=0xa092e7
// packet-audit:verify packet=character/clientbound/BuffGiveForeign version=gms_v95 ida=0xb13200
// packet-audit:verify packet=character/clientbound/BuffGive version=gms_v84 ida=0xa6b6c3
// packet-audit:verify packet=character/clientbound/BuffGiveForeign version=gms_v84 ida=0x9c3bfb
// packet-audit:verify packet=character/clientbound/BuffGive version=jms_v185 ida=0xb0701f
// packet-audit:verify packet=character/clientbound/BuffGiveForeign version=jms_v185 ida=0xa57431
func TestBuffGiveEmptyRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			cts := model.NewCharacterTemporaryStat()
			input := NewBuffGive(*cts)
			output := BuffGive{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

// TestBuffGiveDiseaseTrailer pins that a BuffGive carrying a mob-applied
// disease (here SLOW) ends with the debuff trailer
// (Short(900) + Byte(1)) instead of the buff trailer (Short(0) + Byte(0)).
// Without this branch the v83 client gets the raw stat but skips the
// debuff icon and the flag-gated effects (WEAKEN jump-block, etc.).
func TestBuffGiveDiseaseTrailer(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	tn, _ := tenant.Create([16]byte{}, "GMS", 83, 1)
	cts := model.NewCharacterTemporaryStat()
	cts.AddStat(nil)(tn)(string(character.TemporaryStatTypeSlow), 126, 80, 2, time.Now().Add(15*time.Second))

	got := NewBuffGive(*cts).Encode(nil, ctx)(nil)
	if len(got) < 3 {
		t.Fatalf("encoded payload too short: %d bytes", len(got))
	}
	// Last 3 bytes: Short(900) + Byte(1) → 84 03 01.
	wantTail := []byte{0x84, 0x03, 0x01}
	tail := got[len(got)-3:]
	if !bytes.Equal(tail, wantTail) {
		t.Errorf("disease trailer: got %x want %x (full payload tail: %x)", tail, wantTail, got[max(0, len(got)-8):])
	}
}

// TestBuffGiveBuffTrailer pins that a BuffGive with only player buffs
// keeps the legacy trailer (Short(0) + Byte(0)) — guards against the
// disease branch accidentally swallowing buffs.
func TestBuffGiveBuffTrailer(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	tn, _ := tenant.Create([16]byte{}, "GMS", 83, 1)
	cts := model.NewCharacterTemporaryStat()
	cts.AddStat(nil)(tn)(string(character.TemporaryStatTypeInvincible), 2301003, 30, 20, time.Now().Add(5*time.Minute))

	got := NewBuffGive(*cts).Encode(nil, ctx)(nil)
	if len(got) < 3 {
		t.Fatalf("encoded payload too short: %d bytes", len(got))
	}
	wantTail := []byte{0x00, 0x00, 0x00}
	tail := got[len(got)-3:]
	if !bytes.Equal(tail, wantTail) {
		t.Errorf("buff trailer: got %x want %x", tail, wantTail)
	}
}

func TestBuffGiveForeignEmptyRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			cts := model.NewCharacterTemporaryStat()
			input := NewBuffGiveForeign(12345, *cts)
			output := BuffGiveForeign{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.CharacterId() != 12345 {
				t.Errorf("characterId: got %v, want %v", output.CharacterId(), 12345)
			}
		})
	}
}

// jmsEmptyMask is the jms_v185 SecondaryStat flag word that BuffGive /
// BuffGiveForeign emit with no active per-stat buffs. The TwoState/base group
// (EnergyCharge..Undead) occupies jms shifts 110-116, so bits 110-116 are set
// unconditionally. The mask is written H>>32, H&L, L>>32, L&L (4 ints LE).
// Bits 110-116 fall in H bits 46-52 → first int = 1<<14..1<<20 = 0x001FC000.
// jms client read: SecondaryStat::DecodeForLocal @0x7fcc73 / DecodeForRemote —
// 4× CInPacket::Decode4 for the UINT128 flag word, then per-set-bit blocks.
// This word is jms-distinct from v83 (0x0000FC01 in the L words) and is the
// load-bearing version delta these packets carry.
var jmsEmptyMask = []byte{
	0x00, 0xc0, 0x1f, 0x00, // int0 = 0x001FC000 (bits 110-116)
	0x00, 0x00, 0x00, 0x00, // int1
	0x00, 0x00, 0x00, 0x00, // int2
	0x00, 0x00, 0x00, 0x00, // int3
}

// TestBuffGiveJMSMask pins the jms_v185 empty-CTS SecondaryStat flag word and
// the giveBuff trailer for the local (own-player) BuffGive. The first 16 bytes
// of the body are the flag word read by SecondaryStat::DecodeForLocal; the
// trailing 3 bytes are the buff trailer (Short(0)+Byte(0)).
func TestBuffGiveJMSMask(t *testing.T) {
	v := pt.Variants[4] // JMS v185
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	got := NewBuffGive(*model.NewCharacterTemporaryStat()).Encode(nil, ctx)(nil)
	if !bytes.Equal(got[:16], jmsEmptyMask) {
		t.Errorf("jms BuffGive flag word: got %x want %x", got[:16], jmsEmptyMask)
	}
	// Empty CTS → no per-stat value blocks → mask immediately followed by
	// nDefenseAtt/nDefenseState (00 00) before the base-stat blocks.
	if got[16] != 0x00 || got[17] != 0x00 {
		t.Errorf("jms BuffGive defense bytes: got %x want 0000", got[16:18])
	}
	// giveBuff trailer: Short(0) + Byte(0).
	wantTail := []byte{0x00, 0x00, 0x00}
	if !bytes.Equal(got[len(got)-3:], wantTail) {
		t.Errorf("jms BuffGive trailer: got %x want %x", got[len(got)-3:], wantTail)
	}
}

// TestBuffGiveForeignJMSMask pins the jms_v185 wire for the remote BuffGiveForeign:
// Int(characterId) prefix, then the SecondaryStat flag word (DecodeForRemote),
// then the Short(0)+Byte(0) trailer.
func TestBuffGiveForeignJMSMask(t *testing.T) {
	v := pt.Variants[4] // JMS v185
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
	got := NewBuffGiveForeign(12345, *model.NewCharacterTemporaryStat()).Encode(nil, ctx)(nil)
	wantPrefix := []byte{0x39, 0x30, 0x00, 0x00} // Int(12345) LE
	if !bytes.Equal(got[:4], wantPrefix) {
		t.Errorf("jms BuffGiveForeign characterId: got %x want %x", got[:4], wantPrefix)
	}
	if !bytes.Equal(got[4:20], jmsEmptyMask) {
		t.Errorf("jms BuffGiveForeign flag word: got %x want %x", got[4:20], jmsEmptyMask)
	}
	wantTail := []byte{0x00, 0x00, 0x00}
	if !bytes.Equal(got[len(got)-3:], wantTail) {
		t.Errorf("jms BuffGiveForeign trailer: got %x want %x", got[len(got)-3:], wantTail)
	}
}

// v79EmptyMask is the GMS v79 SecondaryStat flag word BuffGive / BuffGiveForeign
// emit with no active per-stat buffs. The v79 CTS registry path is byte-identical
// to v83 (no version gate fires below 87), so the two-state/base group
// (EnergyCharge..Undead) occupies shifts 82-88 and those bits are set
// unconditionally. Shifts 82-88 → H-word bits 18-24 → int1 = 0x01FC0000, so the
// wire (H>>32, H&L, L>>32, L&L, 4 ints LE) is 00000000 0001FC00... i.e. bytes
// 00 00 00 00 | 00 00 FC 01 | 00 00 00 00 | 00 00 00 00. The v79 client reads this
// 16-byte mask as an opaque UINT128 via SecondaryStat::DecodeForLocal /
// DecodeBuffer(16) (CWvsContext::OnTemporaryStatSet @0x96a6d1), then a trailing
// Decode2 tDelay (§5 opaque caveat — blob absorbed by the trailing opaque buffer).
var v79EmptyMask = []byte{
	0x00, 0x00, 0x00, 0x00, // int0 = H>>32 = 0
	0x00, 0x00, 0xFC, 0x01, // int1 = H&L = 0x01FC0000 (bits 82-88)
	0x00, 0x00, 0x00, 0x00, // int2 = L>>32 = 0
	0x00, 0x00, 0x00, 0x00, // int3 = L&L = 0
}

// TestBuffGiveV79Mask pins the v79 empty-CTS SecondaryStat flag word and the
// giveBuff trailer for the local (own-player) BuffGive. The first 16 bytes are the
// flag word read by SecondaryStat::DecodeForLocal (client @0x96a6d1); the trailing
// 3 bytes are the buff trailer Short(0)+Byte(0). The trailing Decode2 tDelay is the
// u16; the client only reads the trailing MovementAffectingStat byte when the mask
// carries a movement stat (none here) — the emitted byte is harmless over-write.
func TestBuffGiveV79Mask(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	got := NewBuffGive(*model.NewCharacterTemporaryStat()).Encode(nil, ctx)(nil)
	if !bytes.Equal(got[:16], v79EmptyMask) {
		t.Errorf("v79 BuffGive flag word: got %x want %x", got[:16], v79EmptyMask)
	}
	if got[16] != 0x00 || got[17] != 0x00 {
		t.Errorf("v79 BuffGive defense bytes: got %x want 0000", got[16:18])
	}
	wantTail := []byte{0x00, 0x00, 0x00}
	if !bytes.Equal(got[len(got)-3:], wantTail) {
		t.Errorf("v79 BuffGive trailer: got %x want %x", got[len(got)-3:], wantTail)
	}
}

// TestBuffGiveForeignV79Mask pins the v79 wire for the remote BuffGiveForeign:
// Int(characterId) prefix, then the SecondaryStat flag word (DecodeForRemote,
// client @0x8d9a03), then the Short(0)+Byte(0) trailer. charId is consumed by the
// remote-packet dispatcher before the handler body.
func TestBuffGiveForeignV79Mask(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	got := NewBuffGiveForeign(12345, *model.NewCharacterTemporaryStat()).Encode(nil, ctx)(nil)
	wantPrefix := []byte{0x39, 0x30, 0x00, 0x00} // Int(12345) LE
	if !bytes.Equal(got[:4], wantPrefix) {
		t.Errorf("v79 BuffGiveForeign characterId: got %x want %x", got[:4], wantPrefix)
	}
	if !bytes.Equal(got[4:20], v79EmptyMask) {
		t.Errorf("v79 BuffGiveForeign flag word: got %x want %x", got[4:20], v79EmptyMask)
	}
	wantTail := []byte{0x00, 0x00, 0x00}
	if !bytes.Equal(got[len(got)-3:], wantTail) {
		t.Errorf("v79 BuffGiveForeign trailer: got %x want %x", got[len(got)-3:], wantTail)
	}
}

// TestBuffGiveV72Mask pins the legacy GMS v72 empty-CTS wire. v72 < 87 so the CTS
// model's only version gates (87 / 95) do not fire: the SecondaryStat flag word is
// byte-identical to v79 (same v79EmptyMask). IDA-verified: CWvsContext::
// OnTemporaryStatSet @0x918b24 (GMS_v72.1_U_DEVM.exe, port 13339) reads the mask via
// SecondaryStat::DecodeForLocal @0x918b79 into a UINT128 (0x80 bits), then Decode2
// tDelay @0x918b95, then a trailing Decode1 only when IsMovementAffectingStat (none
// here) — the same opaque structure as v79 (§5 caveat).
func TestBuffGiveV72Mask(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	got := NewBuffGive(*model.NewCharacterTemporaryStat()).Encode(nil, ctx)(nil)
	if !bytes.Equal(got[:16], v79EmptyMask) {
		t.Errorf("v72 BuffGive flag word: got %x want %x", got[:16], v79EmptyMask)
	}
	wantTail := []byte{0x00, 0x00, 0x00}
	if !bytes.Equal(got[len(got)-3:], wantTail) {
		t.Errorf("v72 BuffGive trailer: got %x want %x", got[len(got)-3:], wantTail)
	}
}

// TestBuffGiveForeignV72Mask pins the legacy GMS v72 remote wire. Int(characterId)
// prefix, then the SecondaryStat flag word via SecondaryStat::DecodeForRemote (client
// CUserRemote::OnSetTemporaryStat @0x88cb10, UINT128 0x80 bits), then the Short(0)+
// Byte(0) trailer. v72 < 87 so the mask is byte-identical to v79.
func TestBuffGiveForeignV72Mask(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	got := NewBuffGiveForeign(12345, *model.NewCharacterTemporaryStat()).Encode(nil, ctx)(nil)
	wantPrefix := []byte{0x39, 0x30, 0x00, 0x00} // Int(12345) LE
	if !bytes.Equal(got[:4], wantPrefix) {
		t.Errorf("v72 BuffGiveForeign characterId: got %x want %x", got[:4], wantPrefix)
	}
	if !bytes.Equal(got[4:20], v79EmptyMask) {
		t.Errorf("v72 BuffGiveForeign flag word: got %x want %x", got[4:20], v79EmptyMask)
	}
	wantTail := []byte{0x00, 0x00, 0x00}
	if !bytes.Equal(got[len(got)-3:], wantTail) {
		t.Errorf("v72 BuffGiveForeign trailer: got %x want %x", got[len(got)-3:], wantTail)
	}
}

// TestBuffGiveV61Mask pins the very-legacy GMS v61 empty-CTS wire. v61 < 87 so the
// CTS model's only version gates (87 / 95) do not fire → the SecondaryStat flag word
// is byte-identical to v72/v79 (v79EmptyMask, bits 82-88). IDA-verified: the real
// per-op handler CWvsContext::OnTemporaryStatSet @0x84311f (GMS_v61.1_U_DEVM.exe, port
// 13338 — registry's dispatcher note-address 0x830d6b is the CWvsContext switch, not
// the handler) reads the mask via SecondaryStat::DecodeForLocal @0x843174 into a UINT128
// (0x80 bits), Decode2 tDelay @0x843190, then a trailing Decode1 only when
// IsMovementAffectingStat (none here, @0x8433f6) — structurally identical to v72
// @0x918b24 (§5 opaque caveat).
// packet-audit:verify packet=character/clientbound/BuffGive version=gms_v61 ida=0x84311f
func TestBuffGiveV61Mask(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	got := NewBuffGive(*model.NewCharacterTemporaryStat()).Encode(nil, ctx)(nil)
	if !bytes.Equal(got[:16], v79EmptyMask) {
		t.Errorf("v61 BuffGive flag word: got %x want %x", got[:16], v79EmptyMask)
	}
	wantTail := []byte{0x00, 0x00, 0x00}
	if !bytes.Equal(got[len(got)-3:], wantTail) {
		t.Errorf("v61 BuffGive trailer: got %x want %x", got[len(got)-3:], wantTail)
	}
}

// TestBuffGiveForeignV61Mask pins the very-legacy GMS v61 remote wire. Int(characterId)
// prefix, then the SecondaryStat flag word via SecondaryStat::DecodeForRemote (client
// CUserRemote::OnSetTemporaryStat @0x7cbf5f: DecodeForRemote @0x7cbf7d + Decode2 tDelay
// @0x7cbf9b, UINT128 0x80 bits), then the Short(0)+Byte(0) trailer. v61 < 87 so the mask
// is byte-identical to v72/v79 (§5 opaque caveat).
// packet-audit:verify packet=character/clientbound/BuffGiveForeign version=gms_v61 ida=0x7cbf5f
func TestBuffGiveForeignV61Mask(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	got := NewBuffGiveForeign(12345, *model.NewCharacterTemporaryStat()).Encode(nil, ctx)(nil)
	wantPrefix := []byte{0x39, 0x30, 0x00, 0x00} // Int(12345) LE
	if !bytes.Equal(got[:4], wantPrefix) {
		t.Errorf("v61 BuffGiveForeign characterId: got %x want %x", got[:4], wantPrefix)
	}
	if !bytes.Equal(got[4:20], v79EmptyMask) {
		t.Errorf("v61 BuffGiveForeign flag word: got %x want %x", got[4:20], v79EmptyMask)
	}
	wantTail := []byte{0x00, 0x00, 0x00}
	if !bytes.Equal(got[len(got)-3:], wantTail) {
		t.Errorf("v61 BuffGiveForeign trailer: got %x want %x", got[len(got)-3:], wantTail)
	}
}

// TestBuffGiveV48Mask pins the very-legacy GMS v48 empty-CTS local GIVE_BUFF wire.
// Pre-v61 the SecondaryStat mask is a plain 8-byte little-endian value (NOT the
// 128-bit UINT128): CWvsContext::OnTemporaryStatSet @0x71af4b → sub_5CA524, whose
// first op is CInPacket::DecodeBuffer(&v150, 8) @0x5ca539 (GMS_v48_1_DEVM.exe). There
// are no nDefenseAtt/nDefenseState bytes and no trailing base-stat blocks — the
// handler reads only a Decode2 delay @0x71af82 then an optional Decode1. The two-state
// base bits (shifts 81-87) fall in the mask high word the client never reads, so an
// empty CTS is 8 zero bytes, then BuffGive's Short(0)+Byte(0) trailer → 11 bytes.
// packet-audit:verify packet=character/clientbound/BuffGive version=gms_v48 ida=0x71af4b
func TestBuffGiveV48Mask(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
	got := NewBuffGive(*model.NewCharacterTemporaryStat()).Encode(nil, ctx)(nil)
	want := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0x00, 0x00, 0x00} // 8-byte mask + Short(0)+Byte(0)
	if !bytes.Equal(got, want) {
		t.Errorf("v48 BuffGive wire: got %x want %x", got, want)
	}
}

// TestBuffGiveV48SingleStat pins the pre-v61 8-byte mask BIT ORDER and per-stat value
// block. Combo is registry shift 21 (IDA-cross-checked: the v48 foreign decoder
// sub_5CBA1F reads Decode1 for bit 21 = Combo's byte foreign shape), so its mask bit
// lands in wire byte 2 bit 5 → 0x20. The local value block (sub_5CA524, per set bit)
// is Decode2(value)+Decode4(reason)+Decode2(duration/500): short(3) + int(sourceId) +
// short(duration). The duration short is wall-clock-relative so only the mask, value,
// and reason are pinned deterministically.
func TestBuffGiveV48SingleStat(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
	tn, _ := tenant.Create([16]byte{}, "GMS", 48, 1)
	cts := model.NewCharacterTemporaryStat()
	cts.AddStat(nil)(tn)(string(character.TemporaryStatTypeCombo), 0x11223344, 3, 0, time.Now().Add(5*time.Minute))

	got := NewBuffGive(*cts).Encode(nil, ctx)(nil)
	if len(got) != 19 {
		t.Fatalf("v48 BuffGive single-stat length: got %d want 19 (8 mask + 8 value + 3 trailer)", len(got))
	}
	wantMask := []byte{0x00, 0x00, 0x20, 0x00, 0x00, 0x00, 0x00, 0x00} // Combo = shift 21 → byte2 bit5
	if !bytes.Equal(got[:8], wantMask) {
		t.Errorf("v48 BuffGive Combo mask: got %x want %x", got[:8], wantMask)
	}
	if !bytes.Equal(got[8:10], []byte{0x03, 0x00}) {
		t.Errorf("v48 BuffGive Combo value: got %x want 0300", got[8:10])
	}
	if !bytes.Equal(got[10:14], []byte{0x44, 0x33, 0x22, 0x11}) {
		t.Errorf("v48 BuffGive Combo reason: got %x want 44332211", got[10:14])
	}
	// got[14:16] duration short is time-dependent; not pinned.
	if !bytes.Equal(got[16:19], []byte{0x00, 0x00, 0x00}) {
		t.Errorf("v48 BuffGive trailer: got %x want 000000", got[16:19])
	}
}
