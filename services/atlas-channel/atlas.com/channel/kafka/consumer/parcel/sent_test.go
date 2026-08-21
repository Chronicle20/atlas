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

// successfullySentOperations resolves SUCCESSFULLY_SENT to the gms_v83 mode
// from docs/packets/dispatchers/parcel.yaml (0x12), matching
// libs/atlas-packet/parcel/clientbound/v83_test.go's fixture — the only
// place a literal mode byte is legitimate.
var successfullySentOperations = map[string]interface{}{
	parcelcb.ParcelOperationSuccessfullySent: float64(0x12),
}

// sentWp resolves ParcelWriter against successfullySentOperations and
// captures the encoded bytes, mirroring status_test.go's arrivedWp.
func sentWp(captured *[][]byte) writer.Producer {
	return func(name string) (socketwriter.BodyFunc, error) {
		if name != parcelcb.ParcelWriter {
			return nil, nil
		}
		return func(_ logrus.FieldLogger, _ context.Context) func(packet.Encode) []byte {
			return func(encode packet.Encode) []byte {
				b := encode(nullLogger(), context.Background())(map[string]interface{}{"operations": successfullySentOperations})
				*captured = append(*captured, b)
				return b
			}
		}, nil
	}
}

// TestParcelSentEvent pins handleParcelSentEvent against atlas-parcel's
// producer-side StatusEvent[StatusEventParcelSentBody] — field names must
// agree exactly since the two packages are separate Go modules.
//
// The announce is what closes the send loop: without it the client's send
// tab stays disabled after a successful parcel_send, because 0x12 is the arm
// that runs SetCtrlEnabled(1) plus ResetSendInfo/CloseParcelDlg
// (CParcelDlg::OnPacket default arm, v83 @0x6f579d).
func TestParcelSentEvent(t *testing.T) {
	t.Run("online sender", func(t *testing.T) {
		tm := newTestTenant(t)
		ctx := tenant.WithContext(context.Background(), tm)
		s, cleanup := newRealSession(tm, ctx, 200)
		defer cleanup()
		sc := newTestServer(t, tm)
		_ = s

		var captured [][]byte
		h := handleParcelSentEvent(sc, sentWp(&captured))
		h(nullLogger(), ctx, parcelmsg.StatusEvent[parcelmsg.StatusEventParcelSentBody]{
			CharacterId: 200,
			Type:        parcelmsg.StatusEventParcelSent,
			Body:        parcelmsg.StatusEventParcelSentBody{},
		})

		if len(captured) != 1 {
			t.Fatalf("announces = %d, want 1", len(captured))
		}
		// SUCCESSFULLY_SENT is a bare mode byte — no body follows it.
		if len(captured[0]) != 1 || captured[0][0] != 0x12 {
			t.Errorf("body = % x, want 12", captured[0])
		}
	})

	t.Run("offline sender", func(t *testing.T) {
		tm := newTestTenant(t)
		ctx := tenant.WithContext(context.Background(), tm)
		sc := newTestServer(t, tm)

		var captured [][]byte
		h := handleParcelSentEvent(sc, sentWp(&captured))
		h(nullLogger(), ctx, parcelmsg.StatusEvent[parcelmsg.StatusEventParcelSentBody]{
			CharacterId: 999,
			Type:        parcelmsg.StatusEventParcelSent,
			Body:        parcelmsg.StatusEventParcelSentBody{},
		})

		if len(captured) != 0 {
			t.Errorf("announces = %d, want 0", len(captured))
		}
	})

	t.Run("wrong tenant", func(t *testing.T) {
		tm := newTestTenant(t)
		other := newTestTenant(t)
		ctx := tenant.WithContext(context.Background(), other)
		s, cleanup := newRealSession(tm, ctx, 200)
		defer cleanup()
		sc := newTestServer(t, tm)
		_ = s

		var captured [][]byte
		h := handleParcelSentEvent(sc, sentWp(&captured))
		h(nullLogger(), ctx, parcelmsg.StatusEvent[parcelmsg.StatusEventParcelSentBody]{
			CharacterId: 200,
			Type:        parcelmsg.StatusEventParcelSent,
			Body:        parcelmsg.StatusEventParcelSentBody{},
		})

		if len(captured) != 0 {
			t.Errorf("announces = %d, want 0", len(captured))
		}
	})

	t.Run("wrong event type", func(t *testing.T) {
		tm := newTestTenant(t)
		ctx := tenant.WithContext(context.Background(), tm)
		s, cleanup := newRealSession(tm, ctx, 200)
		defer cleanup()
		sc := newTestServer(t, tm)
		_ = s

		var captured [][]byte
		h := handleParcelSentEvent(sc, sentWp(&captured))
		h(nullLogger(), ctx, parcelmsg.StatusEvent[parcelmsg.StatusEventParcelSentBody]{
			CharacterId: 200,
			Type:        parcelmsg.StatusEventParcelArrived,
			Body:        parcelmsg.StatusEventParcelSentBody{},
		})

		if len(captured) != 0 {
			t.Errorf("announces = %d, want 0", len(captured))
		}
	})
}
