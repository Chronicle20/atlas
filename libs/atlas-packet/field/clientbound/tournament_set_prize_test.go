package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TournamentSetPrize read order re-derived from the live IDBs and found
// identical in every version: Decode1(slot), Decode1(flag); flag!=0 gates
// two further Decode4 item ids (both fed to CItemInfo::GetItemName). The
// prior golden modelled the two item ids as unconditional, silently
// desyncing the client whenever flag==0 — a false pass; corrected here
// across all versions (task-181).
// packet-audit:verify packet=field/clientbound/FieldTournamentSetPrize version=gms_v79 ida=0x5587e3
// packet-audit:verify packet=field/clientbound/FieldTournamentSetPrize version=gms_v83 ida=0x57b815
// packet-audit:verify packet=field/clientbound/FieldTournamentSetPrize version=gms_v84 ida=0x58b326
// packet-audit:verify packet=field/clientbound/FieldTournamentSetPrize version=gms_v87 ida=0x5a9f62
// packet-audit:verify packet=field/clientbound/FieldTournamentSetPrize version=gms_v95 ida=0x5633a0
// packet-audit:verify packet=field/clientbound/FieldTournamentSetPrize version=jms_v185 ida=0x5cffa7
func TestTournamentSetPrizeGolden(t *testing.T) {
	// slot=1, flag=2 (nonzero -> item ids present)
	input := NewTournamentSetPrize(0x01, 0x02, 0x00000457, 0x00000005)
	ctx := test.CreateContext("GMS", 83, 1)
	expected := []byte{0x01, 0x02, 0x57, 0x04, 0x00, 0x00, 0x05, 0x00, 0x00, 0x00}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

// TestTournamentSetPrizeGoldenNoItems confirms flag==0 (failure/no-items
// branch) encodes to just the two leading bytes — no item ids follow.
func TestTournamentSetPrizeGoldenNoItems(t *testing.T) {
	input := NewTournamentSetPrize(0x00, 0x00, 0x00000457, 0x00000005)
	ctx := test.CreateContext("GMS", 83, 1)
	expected := []byte{0x00, 0x00}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("no-items golden mismatch: got %v want %v", actual, expected)
	}
}

// TestTournamentSetPrizeByteOutputV79 pins the gms_v79 TOURNAMENT_SET_PRIZE
// clientbound read. IDA: CField_Tournament::OnTournamentSetPrize @0x5587e3
// (GMS_v79_1_DEVM.exe) — Decode1(slot), Decode1(flag); flag!=0 gates two
// Decode4 item ids. Byte-identical to v83/v84/v87/v95/jms.
func TestTournamentSetPrizeByteOutputV79(t *testing.T) {
	input := NewTournamentSetPrize(0x01, 0x02, 0x00000457, 0x00000005)
	ctx := test.CreateContext("GMS", 79, 1)
	expected := []byte{0x01, 0x02, 0x57, 0x04, 0x00, 0x00, 0x05, 0x00, 0x00, 0x00}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v79 golden mismatch: got %v want %v", actual, expected)
	}
}

// TestTournamentSetPrizeByteOutputV79NoItems pins the gms_v79 flag==0 arm:
// no item ids are read.
func TestTournamentSetPrizeByteOutputV79NoItems(t *testing.T) {
	input := NewTournamentSetPrize(0x01, 0x00, 0x00000457, 0x00000005)
	ctx := test.CreateContext("GMS", 79, 1)
	expected := []byte{0x01, 0x00}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v79 no-items golden mismatch: got %v want %v", actual, expected)
	}
}

func TestTournamentSetPrizeRoundTrip(t *testing.T) {
	input := NewTournamentSetPrize(0x01, 0x02, 0x00000457, 0x00000005)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestTournamentSetPrizeRoundTripNoItems(t *testing.T) {
	input := NewTournamentSetPrize(0x01, 0x00, 0, 0)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
