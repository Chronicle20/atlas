package parcel

import (
	parcelmsg "atlas-channel/kafka/message/parcel"
	"atlas-channel/socket/writer"
	"context"
	"testing"

	"github.com/sirupsen/logrus"

	parcelcb "github.com/Chronicle20/atlas/libs/atlas-packet/parcel/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	socketwriter "github.com/Chronicle20/atlas/libs/atlas-socket/writer"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// alarmNamedOperations resolves ALARM_NAMED to the gms_v83 mode from
// docs/packets/dispatchers/parcel.yaml (0x19), matching
// libs/atlas-packet/parcel/clientbound/parcel_notify_test.go:82's fixture —
// the only place a literal mode byte is legitimate.
var alarmNamedOperations = map[string]interface{}{
	parcelcb.ParcelOperationAlarmNamed: float64(0x19),
}

// arrivedWp resolves ParcelWriter against alarmNamedOperations and captures
// the encoded bytes, mirroring consumer_test.go's wp helper.
func arrivedWp(captured *[][]byte) writer.Producer {
	return func(name string) (socketwriter.BodyFunc, error) {
		if name != parcelcb.ParcelWriter {
			return nil, nil
		}
		return func(_ logrus.FieldLogger, _ context.Context) func(packet.Encode) []byte {
			return func(encode packet.Encode) []byte {
				b := encode(nullLogger(), context.Background())(map[string]interface{}{"operations": alarmNamedOperations})
				*captured = append(*captured, b)
				return b
			}
		}, nil
	}
}

// decodeAlarmNamed reads the mode byte, sender name (MapleString-encoded:
// uint16 length prefix, little-endian) and hasItem bool from an ALARM_NAMED
// body, per libs/atlas-packet/parcel/clientbound/parcel_notify_test.go's
// fixture layout.
func decodeAlarmNamed(t *testing.T, b []byte) (senderName string, hasItem bool) {
	t.Helper()
	if len(b) < 3 {
		t.Fatalf("alarm named body too short: % x", b)
	}
	length := int(b[1]) | int(b[2])<<8
	if len(b) < 3+length+1 {
		t.Fatalf("alarm named body too short for name length %d: % x", length, b)
	}
	senderName = string(b[3 : 3+length])
	hasItem = b[3+length] != 0
	return
}

// TestParcelArrivedEvent pins handleParcelArrivedEvent's behavior against
// atlas-parcel's producer-side StatusEvent[StatusEventParcelArrivedBody]
// (task-241 Task 24) — field names must agree exactly since the two
// packages are separate Go modules.
func TestParcelArrivedEvent(t *testing.T) {
	t.Run("online recipient", func(t *testing.T) {
		tm := newTestTenant(t)
		ctx := tenant.WithContext(context.Background(), tm)
		s, cleanup := newRealSession(tm, ctx, 100)
		defer cleanup()
		sc := newTestServer(t, tm)
		_ = s

		var captured [][]byte
		h := handleParcelArrivedEvent(sc, arrivedWp(&captured))
		h(nullLogger(), ctx, parcelmsg.StatusEvent[parcelmsg.StatusEventParcelArrivedBody]{
			CharacterId: 100,
			Type:        parcelmsg.StatusEventParcelArrived,
			Body:        parcelmsg.StatusEventParcelArrivedBody{SenderName: "Alice", HasItem: true},
		})

		if len(captured) != 1 {
			t.Fatalf("announces = %d, want 1", len(captured))
		}
		senderName, hasItem := decodeAlarmNamed(t, captured[0])
		if senderName != "Alice" {
			t.Errorf("senderName = %q, want %q", senderName, "Alice")
		}
		if !hasItem {
			t.Error("hasItem = false, want true")
		}
	})

	t.Run("no item", func(t *testing.T) {
		tm := newTestTenant(t)
		ctx := tenant.WithContext(context.Background(), tm)
		s, cleanup := newRealSession(tm, ctx, 100)
		defer cleanup()
		sc := newTestServer(t, tm)
		_ = s

		var captured [][]byte
		h := handleParcelArrivedEvent(sc, arrivedWp(&captured))
		h(nullLogger(), ctx, parcelmsg.StatusEvent[parcelmsg.StatusEventParcelArrivedBody]{
			CharacterId: 100,
			Type:        parcelmsg.StatusEventParcelArrived,
			Body:        parcelmsg.StatusEventParcelArrivedBody{SenderName: "Alice", HasItem: false},
		})

		if len(captured) != 1 {
			t.Fatalf("announces = %d, want 1", len(captured))
		}
		_, hasItem := decodeAlarmNamed(t, captured[0])
		if hasItem {
			t.Error("hasItem = true, want false")
		}
	})

	t.Run("offline recipient", func(t *testing.T) {
		tm := newTestTenant(t)
		ctx := tenant.WithContext(context.Background(), tm)
		sc := newTestServer(t, tm)

		var captured [][]byte
		h := handleParcelArrivedEvent(sc, arrivedWp(&captured))
		h(nullLogger(), ctx, parcelmsg.StatusEvent[parcelmsg.StatusEventParcelArrivedBody]{
			CharacterId: 999,
			Type:        parcelmsg.StatusEventParcelArrived,
			Body:        parcelmsg.StatusEventParcelArrivedBody{SenderName: "Alice", HasItem: true},
		})

		if len(captured) != 0 {
			t.Errorf("announces = %d, want 0", len(captured))
		}
	})

	t.Run("wrong tenant", func(t *testing.T) {
		tm := newTestTenant(t)
		other := newTestTenant(t)
		ctx := tenant.WithContext(context.Background(), other)
		s, cleanup := newRealSession(tm, ctx, 100)
		defer cleanup()
		sc := newTestServer(t, tm)
		_ = s

		var captured [][]byte
		h := handleParcelArrivedEvent(sc, arrivedWp(&captured))
		h(nullLogger(), ctx, parcelmsg.StatusEvent[parcelmsg.StatusEventParcelArrivedBody]{
			CharacterId: 100,
			Type:        parcelmsg.StatusEventParcelArrived,
			Body:        parcelmsg.StatusEventParcelArrivedBody{SenderName: "Alice", HasItem: true},
		})

		if len(captured) != 0 {
			t.Errorf("announces = %d, want 0", len(captured))
		}
	})

	t.Run("wrong event type", func(t *testing.T) {
		tm := newTestTenant(t)
		ctx := tenant.WithContext(context.Background(), tm)
		s, cleanup := newRealSession(tm, ctx, 100)
		defer cleanup()
		sc := newTestServer(t, tm)
		_ = s

		var captured [][]byte
		h := handleParcelArrivedEvent(sc, arrivedWp(&captured))
		h(nullLogger(), ctx, parcelmsg.StatusEvent[parcelmsg.StatusEventParcelArrivedBody]{
			CharacterId: 100,
			Type:        "SOME_OTHER_EVENT",
			Body:        parcelmsg.StatusEventParcelArrivedBody{SenderName: "Alice", HasItem: true},
		})

		if len(captured) != 0 {
			t.Errorf("announces = %d, want 0", len(captured))
		}
	})
}
