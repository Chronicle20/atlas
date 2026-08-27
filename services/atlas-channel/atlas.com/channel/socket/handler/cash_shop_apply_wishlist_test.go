package handler

import (
	"atlas-channel/socket/writer"
	"context"
	"encoding/binary"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"

	cashcb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	swriter "github.com/Chronicle20/atlas/libs/atlas-socket/writer"
)

// TestCashShopWishListBodyEncodesStoredSerials pins the wire layout
// CashShopWishListUpdateBody produces for the APPLY_WISHLIST answer
// (derivation.md D2b: RESOLVED but flagged INFERENTIAL — the UPDATE_WISHLIST
// arm, mode 98, not LOAD_WISHLIST). The mode byte is followed by exactly 10
// little-endian uint32 slots (WishListUpdate.Encode's fixed-width pad),
// unused slots zero-filled — never a variable-length or empty payload.
func TestCashShopWishListBodyEncodesStoredSerials(t *testing.T) {
	options := map[string]interface{}{
		"operations": map[string]interface{}{
			cashcb.CashShopOperationUpdateWishlist: float64(98),
		},
	}

	tests := []struct {
		name string
		sns  []uint32
	}{
		{"full wishlist", []uint32{10000, 20000, 30000}},
		{"empty wishlist", []uint32{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := cashcb.CashShopWishListUpdateBody(tt.sns)(logrus.New(), context.Background())(options)

			want := make([]byte, 1+10*4)
			want[0] = 98
			for i := 0; i < 10; i++ {
				var v uint32
				if i < len(tt.sns) {
					v = tt.sns[i]
				}
				binary.LittleEndian.PutUint32(want[1+i*4:1+i*4+4], v)
			}

			if len(body) != len(want) {
				t.Fatalf("body length = %d, want %d (mode + 10 fixed uint32 slots)", len(body), len(want))
			}
			if body[0] != 98 {
				t.Fatalf("mode byte = %d, want 98", body[0])
			}
			for i := range want {
				if body[i] != want[i] {
					t.Fatalf("body = %#v, want %#v", body, want)
				}
			}
		})
	}
}

// applyWishlistFailureRecorder is a fake writer.Producer that records every
// announced writer name + body, resolving the "operations"/"errors" tables
// itself (independent of readerOptions) so the test can pin the exact mode
// byte the handler's error branch emits.
type applyWishlistFailureRecorder struct {
	announced []struct {
		writer string
		body   []byte
	}
}

func (r *applyWishlistFailureRecorder) producer() writer.Producer {
	return func(name string) (swriter.BodyFunc, error) {
		return func(l logrus.FieldLogger, ctx context.Context) func(encoder packet.Encode) []byte {
			return func(encoder packet.Encode) []byte {
				b := encoder(l, ctx)(applyWishlistFailureOptions())
				r.announced = append(r.announced, struct {
					writer string
					body   []byte
				}{writer: name, body: b})
				return b
			}
		}, nil
	}
}

func applyWishlistFailureOptions() map[string]interface{} {
	return map[string]interface{}{
		"operations": map[string]interface{}{
			cashcb.CashShopOperationLoadWishFailed: float64(93),
			cashcb.CashShopOperationSetWishFailed:  float64(99),
		},
		"errors": map[string]interface{}{
			"unknown_error": float64(1),
		},
	}
}

// applyWishlistDispatchOptions is the readerOptions table the dispatcher
// needs to route the raw op byte 33 to the APPLY_WISHLIST arm.
func applyWishlistDispatchOptions() map[string]interface{} {
	return map[string]interface{}{
		"operations": map[string]interface{}{
			CashShopOperationApplyWishlist: float64(33),
		},
	}
}

// applyWishlistPacket is D2a's resolved wire shape: the mode byte alone,
// with nothing after it.
func applyWishlistPacket(t *testing.T) *request.Reader {
	t.Helper()
	w := response.NewWriter(logrus.New())
	w.WriteByte(33)
	req := request.Request(w.Bytes())
	reader := request.NewRequestReader(&req, 0)
	return &reader
}

// TestApplyWishlistReadErrorAnswersSetWishFailedNotLoadWishFailed pins fix
// round 1 (review-task-8.md): a wishlist read failure must answer with
// SET_WISH_FAILED (mode 99), because that is the arm paired with the
// latch-clearing UPDATE_WISHLIST success answer above it
// (derivation.md D2b evidence 1 -- OnCashItemResSetWishFailed clears
// m_bCashShopRequestSent, OnCashItemResLoadWishFailed does not). Answering
// with LOAD_WISH_FAILED (mode 93) -- what the plan brief's Step 5
// instruction named, in conflict with the derivation it cites -- would leave
// the client's cash shop wedged on every read error.
func TestApplyWishlistReadErrorAnswersSetWishFailedNotLoadWishFailed(t *testing.T) {
	const characterId = uint32(778899)

	// GetByCharacterId (wishlist.ByCharacterIdProvider -> requests.DrainProvider)
	// fails whenever the upstream cash-shop service returns a non-2xx status.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	t.Setenv("CASHSHOP_SERVICE_URL", srv.URL+"/")

	s, ctx, cleanup := newCashItemUseTestSession(t, characterId)
	defer cleanup()

	rec := &applyWishlistFailureRecorder{}
	CashShopOperationHandleFunc(logrus.New(), ctx, rec.producer())(s, applyWishlistPacket(t), applyWishlistDispatchOptions())

	if len(rec.announced) != 1 {
		t.Fatalf("announced %d packets, want 1", len(rec.announced))
	}
	got := rec.announced[0]
	if got.writer != cashcb.CashShopOperationWriter {
		t.Fatalf("announced writer = %q, want %q", got.writer, cashcb.CashShopOperationWriter)
	}
	if len(got.body) == 0 {
		t.Fatal("announced body is empty")
	}
	if got.body[0] == 93 {
		t.Fatalf("mode byte = %d (LOAD_WISH_FAILED) -- does not clear the client's request-in-flight latch; want 99 (SET_WISH_FAILED)", got.body[0])
	}
	if got.body[0] != 99 {
		t.Fatalf("mode byte = %d, want 99 (SET_WISH_FAILED)", got.body[0])
	}
}
