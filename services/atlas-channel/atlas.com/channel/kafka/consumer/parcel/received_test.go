package parcel

import (
	parcelmsg "atlas-channel/kafka/message/parcel"
	dueyparcel "atlas-channel/parcel"
	"atlas-channel/socket/writer"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	parcelcb "github.com/Chronicle20/atlas/libs/atlas-packet/parcel/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	socketwriter "github.com/Chronicle20/atlas/libs/atlas-socket/writer"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// parcelRemovedOperations resolves PARCEL_REMOVED to the gms_v83 mode from
// docs/packets/dispatchers/parcel.yaml (0x17), matching
// libs/atlas-packet/parcel/clientbound/parcel_notify_test.go's fixture — the
// only place a literal mode byte is legitimate.
var parcelRemovedOperations = map[string]interface{}{
	parcelcb.ParcelOperationParcelRemoved: float64(0x17),
}

// receivedWp resolves ParcelWriter against parcelRemovedOperations and
// captures the encoded bytes, mirroring sent_test.go's sentWp.
func receivedWp(captured *[][]byte) writer.Producer {
	return func(name string) (socketwriter.BodyFunc, error) {
		if name != parcelcb.ParcelWriter {
			return nil, nil
		}
		return func(_ logrus.FieldLogger, _ context.Context) func(packet.Encode) []byte {
			return func(encode packet.Encode) []byte {
				b := encode(nullLogger(), context.Background())(map[string]interface{}{"operations": parcelRemovedOperations})
				*captured = append(*captured, b)
				return b
			}
		}, nil
	}
}

// TestParcelReceivedEvent pins handleParcelReceivedEvent against atlas-
// parcel's producer-side StatusEvent[StatusEventParcelReceivedBody] — field
// names must agree exactly since the two packages are separate Go modules.
//
// The announce is what closes the receive loop: without it the client's
// CParcelDlg stays disabled forever, because PARCEL_REMOVED's case 23 is the
// arm that runs RemoveParcel then SetCtrlEnabled(1) itself
// (CParcelDlg::OnPacket, v83 @0x6f56ea).
func TestParcelReceivedEvent(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{name: "online recipient", run: func(t *testing.T) {
			tm := newTestTenant(t)
			ctx := tenant.WithContext(context.Background(), tm)
			s, cleanup := newRealSession(tm, ctx, 100)
			defer cleanup()
			sc := newTestServer(t, tm)
			_ = s

			parcelId := uuid.New()
			var captured [][]byte
			h := handleParcelReceivedEvent(sc, receivedWp(&captured))
			h(nullLogger(), ctx, parcelmsg.StatusEvent[parcelmsg.StatusEventParcelReceivedBody]{
				CharacterId: 100,
				Type:        parcelmsg.StatusEventParcelReceived,
				Body:        parcelmsg.StatusEventParcelReceivedBody{ParcelId: parcelId},
			})

			if len(captured) != 1 {
				t.Fatalf("announces = %d, want 1", len(captured))
			}
			got := captured[0]
			want := []byte{
				0x17,
				byte(dueyparcel.WireId(parcelId)),
				byte(dueyparcel.WireId(parcelId) >> 8),
				byte(dueyparcel.WireId(parcelId) >> 16),
				byte(dueyparcel.WireId(parcelId) >> 24),
				parcelcb.ParcelRemovedKindClaimed,
			}
			if len(got) != len(want) {
				t.Fatalf("body = % x, want % x", got, want)
			}
			for i := range want {
				if got[i] != want[i] {
					t.Errorf("body = % x, want % x", got, want)
					break
				}
			}
		}},
		{name: "offline recipient", run: func(t *testing.T) {
			tm := newTestTenant(t)
			ctx := tenant.WithContext(context.Background(), tm)
			sc := newTestServer(t, tm)

			var captured [][]byte
			h := handleParcelReceivedEvent(sc, receivedWp(&captured))
			h(nullLogger(), ctx, parcelmsg.StatusEvent[parcelmsg.StatusEventParcelReceivedBody]{
				CharacterId: 999,
				Type:        parcelmsg.StatusEventParcelReceived,
				Body:        parcelmsg.StatusEventParcelReceivedBody{ParcelId: uuid.New()},
			})

			if len(captured) != 0 {
				t.Errorf("announces = %d, want 0", len(captured))
			}
		}},
		{name: "wrong tenant", run: func(t *testing.T) {
			tm := newTestTenant(t)
			other := newTestTenant(t)
			ctx := tenant.WithContext(context.Background(), other)
			s, cleanup := newRealSession(tm, ctx, 100)
			defer cleanup()
			sc := newTestServer(t, tm)
			_ = s

			var captured [][]byte
			h := handleParcelReceivedEvent(sc, receivedWp(&captured))
			h(nullLogger(), ctx, parcelmsg.StatusEvent[parcelmsg.StatusEventParcelReceivedBody]{
				CharacterId: 100,
				Type:        parcelmsg.StatusEventParcelReceived,
				Body:        parcelmsg.StatusEventParcelReceivedBody{ParcelId: uuid.New()},
			})

			if len(captured) != 0 {
				t.Errorf("announces = %d, want 0", len(captured))
			}
		}},
		{name: "wrong event type", run: func(t *testing.T) {
			tm := newTestTenant(t)
			ctx := tenant.WithContext(context.Background(), tm)
			s, cleanup := newRealSession(tm, ctx, 100)
			defer cleanup()
			sc := newTestServer(t, tm)
			_ = s

			var captured [][]byte
			h := handleParcelReceivedEvent(sc, receivedWp(&captured))
			h(nullLogger(), ctx, parcelmsg.StatusEvent[parcelmsg.StatusEventParcelReceivedBody]{
				CharacterId: 100,
				Type:        parcelmsg.StatusEventParcelSent,
				Body:        parcelmsg.StatusEventParcelReceivedBody{ParcelId: uuid.New()},
			})

			if len(captured) != 0 {
				t.Errorf("announces = %d, want 0", len(captured))
			}
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}
