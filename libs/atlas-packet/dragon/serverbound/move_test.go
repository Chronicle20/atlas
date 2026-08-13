package serverbound

import (
	"bytes"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// CVecCtrlDragon::EndUpdateActive (GMS v95.0 @0x996570, v83 @0x9b7b9c) writes
// COutPacket(op) then CMovePath::Flush(...) and NOTHING else. There is no
// leading identity field — unlike CVecCtrlSummoned::EndUpdateActive, which
// writes Encode4 summonId first.
//
// packet-audit:verify packet=dragon/serverbound/Move version=gms_v95 ida=0x996570
func TestServerboundMoveHasNoLeadingIdentityField(t *testing.T) {
	ctx := test.CreateContext("GMS", 95, 1)
	l, _ := testlog.NewNullLogger()

	// startX=100 (0x64 0x00), startY=-200 (0x38 0xFF), then payload
	blob := []byte{0x64, 0x00, 0x38, 0xFF, 0x01, 0x00, 0x07}
	req := request.Request(blob)
	reader := request.NewRequestReader(&req, 0)

	var m Move
	m.Decode(l, ctx)(&reader, nil)

	if !bytes.Equal(m.RawMovement(), blob) {
		t.Fatalf("rawMovement must be the WHOLE body, got % X", m.RawMovement())
	}
	if m.StartX() != 100 || m.StartY() != -200 {
		t.Fatalf("start position = %d,%d, want 100,-200", m.StartX(), m.StartY())
	}

	got := test.Encode(t, ctx, m.Encode, nil)
	if !bytes.Equal(got, blob) {
		t.Fatalf("encode must be byte-faithful, got % X, want % X", got, blob)
	}
}

// v83 send site: CVecCtrlDragon::EndUpdateActive @0x9b7b9c
// (MapleStory_dump.exe.i64 v83, session 41f13e0d) writes COutPacket(0xB5=181)
// then CMovePath::Flush(...) and NOTHING else — no leading identity field,
// same shape as v95.
//
// packet-audit:verify packet=dragon/serverbound/Move version=gms_v83 ida=0x9b7b9c
func TestServerboundMoveHasNoLeadingIdentityField_v83(t *testing.T) {
	ctx := test.CreateContext("GMS", 83, 1)
	l, _ := testlog.NewNullLogger()

	blob := []byte{0x64, 0x00, 0x38, 0xFF, 0x01, 0x00, 0x07}
	req := request.Request(blob)
	reader := request.NewRequestReader(&req, 0)

	var m Move
	m.Decode(l, ctx)(&reader, nil)

	if !bytes.Equal(m.RawMovement(), blob) {
		t.Fatalf("v83 rawMovement must be the WHOLE body, got % X", m.RawMovement())
	}
	if m.StartX() != 100 || m.StartY() != -200 {
		t.Fatalf("v83 start position = %d,%d, want 100,-200", m.StartX(), m.StartY())
	}

	got := test.Encode(t, ctx, m.Encode, nil)
	if !bytes.Equal(got, blob) {
		t.Fatalf("v83 encode must be byte-faithful, got % X, want % X", got, blob)
	}
}

// The layout is uniform across all six applicable versions.
func TestServerboundMoveIdenticalAcrossVersions(t *testing.T) {
	blob := []byte{0x64, 0x00, 0x38, 0xFF, 0x01, 0x00, 0x07}
	l, _ := testlog.NewNullLogger()
	versions := []struct {
		region string
		major  uint16
	}{
		{"GMS", 83},
		{"GMS", 84},
		{"GMS", 87},
		{"GMS", 92},
		{"GMS", 95},
		{"JMS", 185},
	}
	for _, v := range versions {
		ctx := test.CreateContext(v.region, v.major, 1)
		req := request.Request(blob)
		reader := request.NewRequestReader(&req, 0)
		var m Move
		m.Decode(l, ctx)(&reader, nil)
		if !bytes.Equal(m.RawMovement(), blob) || m.StartX() != 100 || m.StartY() != -200 {
			t.Errorf("%s v%d: decode diverged: %+v", v.region, v.major, m)
		}
	}
}
