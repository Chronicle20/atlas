package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func testBalloon() Balloon {
	return NewBalloon(5, 42, "CD", 1, 4, 0)
}

// employeeVersions enumerates the eight client versions that carry the
// hired-merchant feature. The wire layout is byte-identical across all of them
// (IDA-confirmed CEmployeePool::OnEmployeeEnterField/LeaveField/MiniRoomBalloon
// -> CEmployee::Init/SetBalloon on every IDB), so each golden test encodes under
// every version's tenant context and asserts the SAME bytes. gms_v48 has no
// hired-merchant feature (packet never routed there) and is excluded.
var employeeVersions = []struct {
	region string
	major  uint16
}{
	{"GMS", 61},
	{"GMS", 72},
	{"GMS", 79},
	{"GMS", 83},
	{"GMS", 84},
	{"GMS", 87},
	{"GMS", 95},
	{"JMS", 185},
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
// each byte tracing to the client read order:
//
//	CEmployeePool::OnEmployeeEnterField: Decode4 employeeId, Decode4 templateId,
//	then CEmployee::Init (Decode2 x, Decode2 y, Decode2 fh, DecodeStr ownerName),
//	then CEmployee::SetBalloon (Decode1 type; if type!=0: Decode4 sn, DecodeStr
//	title, Decode1, Decode1, Decode1).
//
// The layout is byte-identical on gms v61/72/79/83/84/87/95 and jms v185 (each
// address IDA-verified); gms v48 has no hired-merchant feature.
//
// packet-audit:verify packet=merchant/clientbound/EmployeeSpawn version=gms_v61 ida=0x4d3483
// packet-audit:verify packet=merchant/clientbound/EmployeeSpawn version=gms_v72 ida=0x4f4995
// packet-audit:verify packet=merchant/clientbound/EmployeeSpawn version=gms_v79 ida=0x4fd6b3
// packet-audit:verify packet=merchant/clientbound/EmployeeSpawn version=gms_v83 ida=0x510e83
// packet-audit:verify packet=merchant/clientbound/EmployeeSpawn version=gms_v84 ida=0x519e04
// packet-audit:verify packet=merchant/clientbound/EmployeeSpawn version=gms_v87 ida=0x533528
// packet-audit:verify packet=merchant/clientbound/EmployeeSpawn version=gms_v95 ida=0x518f70
// packet-audit:verify packet=merchant/clientbound/EmployeeSpawn version=jms_v185 ida=0x542a71
func TestEmployeeSpawnBytes(t *testing.T) {
	input := NewEmployeeSpawn(1000, 9000000, 100, -50, 7, "AB", NewBalloon(5, 42, "CD", 1, 4, 0))
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
	for _, v := range employeeVersions {
		ctx := pt.CreateContext(v.region, v.major, 1)
		b := pt.Encode(t, ctx, input.Encode, nil)
		if !bytes.Equal(b, want) {
			t.Fatalf("%s v%d bytes: got % x, want % x", v.region, v.major, b, want)
		}
	}
}

// TestEmployeeDestroyBytes pins DESTROY_HIRED_MERCHANT: a single u32 employeeId
// (CEmployeePool::OnEmployeeLeaveField reads Decode4 and nothing else).
//
// packet-audit:verify packet=merchant/clientbound/EmployeeDestroy version=gms_v61 ida=0x4d3520
// packet-audit:verify packet=merchant/clientbound/EmployeeDestroy version=gms_v72 ida=0x4f4a32
// packet-audit:verify packet=merchant/clientbound/EmployeeDestroy version=gms_v79 ida=0x4fd750
// packet-audit:verify packet=merchant/clientbound/EmployeeDestroy version=gms_v83 ida=0x510f20
// packet-audit:verify packet=merchant/clientbound/EmployeeDestroy version=gms_v84 ida=0x519ea1
// packet-audit:verify packet=merchant/clientbound/EmployeeDestroy version=gms_v87 ida=0x5335c5
// packet-audit:verify packet=merchant/clientbound/EmployeeDestroy version=gms_v95 ida=0x518d10
// packet-audit:verify packet=merchant/clientbound/EmployeeDestroy version=jms_v185 ida=0x542b0e
func TestEmployeeDestroyBytes(t *testing.T) {
	input := NewEmployeeDestroy(1000)
	want := []byte{0xE8, 0x03, 0x00, 0x00} // employeeId 1000, and nothing else
	for _, v := range employeeVersions {
		ctx := pt.CreateContext(v.region, v.major, 1)
		b := pt.Encode(t, ctx, input.Encode, nil)
		if !bytes.Equal(b, want) {
			t.Fatalf("%s v%d bytes: got % x, want % x", v.region, v.major, b, want)
		}
	}
}

// TestEmployeeUpdateBytes pins UPDATE_HIRED_MERCHANT: a u32 employeeId followed
// by the CEmployee::SetBalloon block (CEmployeePool::OnEmployeeMiniRoomBalloon).
//
// packet-audit:verify packet=merchant/clientbound/EmployeeUpdate version=gms_v61 ida=0x4d357e
// packet-audit:verify packet=merchant/clientbound/EmployeeUpdate version=gms_v72 ida=0x4f4a90
// packet-audit:verify packet=merchant/clientbound/EmployeeUpdate version=gms_v79 ida=0x4fd7ae
// packet-audit:verify packet=merchant/clientbound/EmployeeUpdate version=gms_v83 ida=0x510f7e
// packet-audit:verify packet=merchant/clientbound/EmployeeUpdate version=gms_v84 ida=0x519eff
// packet-audit:verify packet=merchant/clientbound/EmployeeUpdate version=gms_v87 ida=0x533623
// packet-audit:verify packet=merchant/clientbound/EmployeeUpdate version=gms_v95 ida=0x5187d0
// packet-audit:verify packet=merchant/clientbound/EmployeeUpdate version=jms_v185 ida=0x542b6c
func TestEmployeeUpdateBytes(t *testing.T) {
	input := NewEmployeeUpdate(1000, NewBalloon(5, 42, "CD", 1, 4, 0))
	want := []byte{
		0xE8, 0x03, 0x00, 0x00, // employeeId 1000
		0x05,                   // balloon miniRoomType 5
		0x2A, 0x00, 0x00, 0x00, // miniRoomSN 42
		0x02, 0x00, 0x43, 0x44, // title "CD"
		0x01, 0x04, 0x00, // cur, max, spec
	}
	for _, v := range employeeVersions {
		ctx := pt.CreateContext(v.region, v.major, 1)
		b := pt.Encode(t, ctx, input.Encode, nil)
		if !bytes.Equal(b, want) {
			t.Fatalf("%s v%d bytes: got % x, want % x", v.region, v.major, b, want)
		}
	}
}
