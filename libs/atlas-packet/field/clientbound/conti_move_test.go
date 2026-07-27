package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// ContiMove read order re-derived from the live IDBs and found identical in
// every version: Decode1(state) dispatches on (state-7) to one of six arms;
// arms 8/10/12 (OnStartShipMoveField/OnMoveField/OnEndShipMoveField) each
// Decode1 a second subState byte, arms 7/9/11 are nullsubs that read nothing
// further. The prior golden modelled a single unconditional state byte,
// silently dropping the subState byte for 8/10/12 — a false pass; corrected
// here across all versions (task-181).
// packet-audit:verify packet=field/clientbound/FieldContiMove version=gms_v79 ida=0x5374c1
// packet-audit:verify packet=field/clientbound/FieldContiMove version=gms_v83 ida=0x54dca3
// packet-audit:verify packet=field/clientbound/FieldContiMove version=gms_v84 ida=0x55a4e2
// packet-audit:verify packet=field/clientbound/FieldContiMove version=gms_v87 ida=0x577bbc
// packet-audit:verify packet=field/clientbound/FieldContiMove version=gms_v95 ida=0x54d680
// packet-audit:verify packet=field/clientbound/FieldContiMove version=jms_v185 ida=0x58e21b
func TestContiMoveGolden(t *testing.T) {
	// state=10 (OnMoveField), subState=4 (CShip::AppearShip)
	input := NewContiMove(10, 4)
	ctx := test.CreateContext("GMS", 83, 1)
	expected := []byte{0x0A, 0x04}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

// TestContiMoveGoldenNullsub confirms a nullsub state (7/9/11 — no
// sub-handler read) encodes to just the single state byte.
func TestContiMoveGoldenNullsub(t *testing.T) {
	input := NewContiMove(9, 0)
	ctx := test.CreateContext("GMS", 83, 1)
	expected := []byte{0x09}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("nullsub golden mismatch: got %v want %v", actual, expected)
	}
}

// TestContiMoveByteOutputV79 pins the gms_v79 CONTI_MOVE clientbound read.
// IDA: CField_ContiMove::OnContiMove @0x5374c1 (GMS_v79_1_DEVM.exe) —
// Decode1(state), then for state 8 dispatches to sub_5375AE
// (OnStartShipMoveField) which Decode1(subState); ==2 triggers
// CShip::LeaveShipMove via sub_5369DE. Byte-identical to v83/v84/v87/v95/jms.
func TestContiMoveByteOutputV79(t *testing.T) {
	// state=8 (OnStartShipMoveField), subState=2 (CShip::LeaveShipMove)
	input := NewContiMove(8, 2)
	ctx := test.CreateContext("GMS", 79, 1)
	expected := []byte{0x08, 0x02}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v79 golden mismatch: got %v want %v", actual, expected)
	}
}

// TestContiMoveByteOutputV79Nullsub pins the gms_v79 CONTI_MOVE nullsub arm
// (state=11 -> nullsub_9 @0x537606): no subState byte is read.
func TestContiMoveByteOutputV79Nullsub(t *testing.T) {
	input := NewContiMove(11, 0)
	ctx := test.CreateContext("GMS", 79, 1)
	expected := []byte{0x0B}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v79 nullsub golden mismatch: got %v want %v", actual, expected)
	}
}

func TestContiMoveRoundTrip(t *testing.T) {
	// state=12 (OnEndShipMoveField), subState=6 (CShip::EnterShipMove)
	input := NewContiMove(12, 6)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestContiMoveRoundTripNullsub(t *testing.T) {
	input := NewContiMove(7, 0)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
