package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func testBalloon() Balloon {
	return NewBalloon(5, 42, "CD", 1, 4, 0)
}

func TestEmployeeSpawnRoundTrip(t *testing.T) {
	input := NewEmployeeSpawn(1000, 9000000, 100, -50, 7, "AB", testBalloon())
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := &EmployeeSpawn{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.EmployeeId() != 1000 || output.TemplateId() != 9000000 {
				t.Errorf("ids: got employee %d template %d", output.EmployeeId(), output.TemplateId())
			}
			if output.X() != 100 || output.Y() != -50 || output.Foothold() != 7 {
				t.Errorf("pos: got x %d y %d fh %d", output.X(), output.Y(), output.Foothold())
			}
			if output.OwnerName() != "AB" {
				t.Errorf("ownerName: got %q", output.OwnerName())
			}
			if output.Balloon().MiniRoomType() != 5 || output.Balloon().Title() != "CD" {
				t.Errorf("balloon: got type %d title %q", output.Balloon().MiniRoomType(), output.Balloon().Title())
			}
		})
	}
}

// TestEmployeeSpawnEmptyBalloonRoundTrip covers the MiniRoomType==0 branch, where
// the client reads nothing after the type byte.
func TestEmployeeSpawnEmptyBalloonRoundTrip(t *testing.T) {
	input := NewEmployeeSpawn(1, 9000000, 0, 0, 0, "X", NewEmptyBalloon())
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := &EmployeeSpawn{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Balloon().MiniRoomType() != 0 {
				t.Errorf("balloon type: got %d, want 0", output.Balloon().MiniRoomType())
			}
		})
	}
}

func TestEmployeeDestroyRoundTrip(t *testing.T) {
	input := NewEmployeeDestroy(1000)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := &EmployeeDestroy{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.EmployeeId() != 1000 {
				t.Errorf("employeeId: got %d, want 1000", output.EmployeeId())
			}
		})
	}
}

func TestEmployeeUpdateRoundTrip(t *testing.T) {
	input := NewEmployeeUpdate(1000, testBalloon())
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := &EmployeeUpdate{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.EmployeeId() != 1000 || output.Balloon().Title() != "CD" {
				t.Errorf("update: got id %d title %q", output.EmployeeId(), output.Balloon().Title())
			}
		})
	}
}

// TestEmployeeSpawnBytes pins the SPAWN_HIRED_MERCHANT wire layout byte-for-byte,
// each byte tracing to the v83 read order:
//
//	CEmployeePool::OnEmployeeEnterField (v83 @0x510e83): Decode4 employeeId, Decode4
//	templateId, then CEmployee::Init (@0x50d56c: Decode2 x, Decode2 y, Decode2 fh,
//	DecodeStr ownerName), then CEmployee::SetBalloon (@0x50d897: Decode1 type; if
//	type!=0: Decode4 sn, DecodeStr title, Decode1, Decode1, Decode1).
//
// The layout is byte-identical on gms v61/72/79/83/84/87/95 and jms v185; gms v48
// has no hired-merchant feature (packet never routed there).
//
// v83 read fn CEmployeePool::OnEmployeeEnterField @0x510e83 (formal matrix promotion pending: evidence pin + per-version cells)
func TestEmployeeSpawnBytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := NewEmployeeSpawn(1000, 9000000, 100, -50, 7, "AB", NewBalloon(5, 42, "CD", 1, 4, 0))
	ctx := pt.CreateContext("GMS", 83, 1)
	b := input.Encode(l, ctx)(nil)
	want := []byte{
		0xE8, 0x03, 0x00, 0x00, // employeeId 1000
		0x40, 0x54, 0x89, 0x00, // templateId 9000000
		0x64, 0x00, // x 100
		0xCE, 0xFF, // y -50
		0x07, 0x00, // foothold 7
		0x02, 0x00, 0x41, 0x42, // ownerName "AB"
		0x05,                   // balloon miniRoomType 5
		0x2A, 0x00, 0x00, 0x00, // miniRoomSN 42
		0x02, 0x00, 0x43, 0x44, // title "CD"
		0x01, 0x04, 0x00, // curVisitors 1, maxVisitors 4, spec 0
	}
	if !bytes.Equal(b, want) {
		t.Fatalf("bytes: got % x, want % x", b, want)
	}
}

// v83 read fn CEmployeePool::OnEmployeeLeaveField @0x510f20 (formal matrix promotion pending)
func TestEmployeeDestroyBytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := NewEmployeeDestroy(1000)
	ctx := pt.CreateContext("GMS", 83, 1)
	b := input.Encode(l, ctx)(nil)
	want := []byte{0xE8, 0x03, 0x00, 0x00} // employeeId 1000, and nothing else
	if !bytes.Equal(b, want) {
		t.Fatalf("bytes: got % x, want % x", b, want)
	}
}

// v83 read fn CEmployeePool::OnEmployeeMiniRoomBalloon @0x510f7e (formal matrix promotion pending)
func TestEmployeeUpdateBytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := NewEmployeeUpdate(1000, NewBalloon(5, 42, "CD", 1, 4, 0))
	ctx := pt.CreateContext("GMS", 83, 1)
	b := input.Encode(l, ctx)(nil)
	want := []byte{
		0xE8, 0x03, 0x00, 0x00, // employeeId 1000
		0x05,                   // balloon miniRoomType 5
		0x2A, 0x00, 0x00, 0x00, // miniRoomSN 42
		0x02, 0x00, 0x43, 0x44, // title "CD"
		0x01, 0x04, 0x00, // cur, max, spec
	}
	if !bytes.Equal(b, want) {
		t.Fatalf("bytes: got % x, want % x", b, want)
	}
}
