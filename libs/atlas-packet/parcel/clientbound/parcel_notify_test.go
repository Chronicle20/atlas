package clientbound

import (
	"bytes"
	"testing"
	"time"

	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-packet/parcel"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// TestParcelNotifyArms pins the four body-carrying PARCEL notify arms
// (task-241 Task 9): CParcelDlg::OnPacket @0x6F56EA (v83) explicit cases
// 23 (PARCEL_REMOVED), 24 (PARCEL_ARRIVED), 25 (ALARM_NAMED), and 27
// (ALARM_GENERIC). Each body func resolves its mode from the tenant
// `operations` table (never a hard-coded literal). gms_v83 modes per
// docs/packets/dispatchers/parcel.yaml (Task 6). All four keys carry a
// jms_v185 value in that table (Ruling 5's 7 populated keys).
//
// No `packet-audit:verify` marker: that requires the per-version IDA
// export + evidence/audit-report pair (DISPATCHER_FAMILY.md step 5,
// VERIFYING_A_PACKET.md), which is out of scope for this task (assigned
// to Task 28).
func TestParcelNotifyArms(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)

	t.Run("removed claimed", func(t *testing.T) {
		l, _ := testlog.NewNullLogger()
		options := map[string]interface{}{
			"operations": map[string]interface{}{
				ParcelOperationParcelRemoved: float64(0x17),
			},
		}
		got := ParcelRemovedBody(7, ParcelRemovedKindClaimed)(l, ctx)(options)
		want := []byte{0x17, 0x07, 0x00, 0x00, 0x00, ParcelRemovedKindClaimed}
		if !bytes.Equal(got, want) {
			t.Errorf("removed claimed: got % x want % x", got, want)
		}
	})

	t.Run("removed discarded", func(t *testing.T) {
		l, _ := testlog.NewNullLogger()
		options := map[string]interface{}{
			"operations": map[string]interface{}{
				ParcelOperationParcelRemoved: float64(0x17),
			},
		}
		got := ParcelRemovedBody(7, ParcelRemovedKindDiscarded)(l, ctx)(options)
		want := []byte{0x17, 0x07, 0x00, 0x00, 0x00, 0x03}
		if !bytes.Equal(got, want) {
			t.Errorf("removed discarded: got % x want % x", got, want)
		}
	})

	t.Run("arrived", func(t *testing.T) {
		l, _ := testlog.NewNullLogger()
		sentAt := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
		p := parcel.NewParcel(7, "Alice", 1000, sentAt, "hi")
		pBytes := p.Encode(nil, ctx)(nil)

		options := map[string]interface{}{
			"operations": map[string]interface{}{
				ParcelOperationParcelArrived: float64(0x18),
			},
		}
		got := ParcelArrivedBody(p)(l, ctx)(options)
		var want []byte
		want = append(want, 0x18)
		want = append(want, pBytes...)
		if !bytes.Equal(got, want) {
			t.Errorf("arrived: got % x want % x", got, want)
		}
	})

	t.Run("alarm named with item", func(t *testing.T) {
		l, _ := testlog.NewNullLogger()
		options := map[string]interface{}{
			"operations": map[string]interface{}{
				ParcelOperationAlarmNamed: float64(0x19),
			},
		}
		got := ParcelAlarmNamedBody("Alice", true)(l, ctx)(options)
		var want []byte
		want = append(want, 0x19, 0x05, 0x00)
		want = append(want, []byte("Alice")...)
		want = append(want, 0x01)
		if !bytes.Equal(got, want) {
			t.Errorf("alarm named with item: got % x want % x", got, want)
		}
	})

	t.Run("alarm named no item", func(t *testing.T) {
		l, _ := testlog.NewNullLogger()
		options := map[string]interface{}{
			"operations": map[string]interface{}{
				ParcelOperationAlarmNamed: float64(0x19),
			},
		}
		got := ParcelAlarmNamedBody("Alice", false)(l, ctx)(options)
		var want []byte
		want = append(want, 0x19, 0x05, 0x00)
		want = append(want, []byte("Alice")...)
		want = append(want, 0x00)
		if !bytes.Equal(got, want) {
			t.Errorf("alarm named no item: got % x want % x", got, want)
		}
	})

	t.Run("alarm generic", func(t *testing.T) {
		l, _ := testlog.NewNullLogger()
		options := map[string]interface{}{
			"operations": map[string]interface{}{
				ParcelOperationAlarmGeneric: float64(0x1B),
			},
		}
		got := ParcelAlarmGenericBody(true)(l, ctx)(options)
		want := []byte{0x1B, 0x01}
		if !bytes.Equal(got, want) {
			t.Errorf("alarm generic: got % x want % x", got, want)
		}
	})
}

// TestParcelOpenQuickDecode pins OpenQuick.Decode against its own Encode
// output — added silently by Task 8 (untested at the time; task-9 brief
// asks for coverage alongside the new arms).
func TestParcelOpenQuickDecode(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	encoded := NewParcelOpenQuick(0x1A).Encode(nil, ctx)(nil)

	req := request.Request(encoded)
	reader := request.NewRequestReader(&req, 0)

	var m OpenQuick
	m.Decode(nil, ctx)(&reader, nil)

	if m.Mode() != 0x1A {
		t.Errorf("OpenQuick.Decode: got mode %#x want %#x", m.Mode(), 0x1A)
	}
	if reader.Available() > 0 {
		t.Errorf("OpenQuick.Decode: reader has %d unconsumed bytes", reader.Available())
	}
}
