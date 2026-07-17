package tv

import (
	"context"
	"io"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/sirupsen/logrus"
)

func testLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

func testTvSenderLook() model.Avatar {
	equip := map[slot.Position]uint32{5: 1040002}
	masked := map[slot.Position]uint32{}
	pets := map[int8]uint32{}
	return model.NewAvatar(0, 1, 20000, false, 30000, equip, masked, pets)
}

// TestTvSetMessageBodyResolvesMessageType verifies the DOM-25 contract:
// TvSetMessageBody never emits a literal wire byte for messageType — it is
// resolved per tenant from the TvSetMessage writer's "messageTypes" table.
func TestTvSetMessageBodyResolvesMessageType(t *testing.T) {
	l := testLogger()
	ctx := pt.CreateContext("GMS", 83, 1)

	// A v83-shaped writer config: NORMAL=0, STAR=1, HEART=2.
	options := map[string]interface{}{
		"messageTypes": map[string]interface{}{
			"NORMAL": float64(0),
			"STAR":   float64(1),
			"HEART":  float64(2),
		},
	}

	senderLook := testTvSenderLook()
	lines := [5]string{"a", "b", "c", "d", "e"}

	cases := []struct {
		msgType  TvMessageType
		wantByte byte
	}{
		{TvMessageNormal, 0},
		{TvMessageStar, 1},
		{TvMessageHeart, 2},
	}
	for _, c := range cases {
		t.Run(string(c.msgType), func(t *testing.T) {
			got := TvSetMessageBody(c.msgType, senderLook, "Sender", "", lines, 60, nil)(l, ctx)(options)
			if len(got) < 2 {
				t.Fatalf("payload too short: got %d bytes", len(got))
			}
			// byte[0] = flag (1, no receiver), byte[1] = resolved messageType.
			if got[0] != 1 {
				t.Errorf("flag byte: got 0x%02X, want 0x01", got[0])
			}
			if got[1] != c.wantByte {
				t.Errorf("messageType byte: got 0x%02X, want 0x%02X", got[1], c.wantByte)
			}
		})
	}
}

// TestTvSetMessageBodyUnconfiguredMessageTypeDegrades confirms a messageType
// missing from the tenant "messageTypes" table degrades to 99 rather than
// panicking or silently sending 0.
func TestTvSetMessageBodyUnconfiguredMessageTypeDegrades(t *testing.T) {
	l := testLogger()
	ctx := pt.CreateContext("GMS", 83, 1)
	options := map[string]interface{}{
		"messageTypes": map[string]interface{}{
			"NORMAL": float64(0),
		},
	}
	senderLook := testTvSenderLook()
	lines := [5]string{"a", "b", "c", "d", "e"}
	got := TvSetMessageBody(TvMessageHeart, senderLook, "Sender", "", lines, 60, nil)(l, ctx)(options)
	if len(got) < 2 {
		t.Fatalf("payload too short: got %d bytes", len(got))
	}
	if got[1] != 99 {
		t.Errorf("messageType byte: got 0x%02X, want 0x63 (99)", got[1])
	}
}

// TestTvSendMessageResultErrorBodyResolvesErrorCode verifies the DOM-25
// contract for the error-result body: the notice selector is resolved per
// tenant from the TvSendMessageResult writer's "errorCodes" table.
func TestTvSendMessageResultErrorBodyResolvesErrorCode(t *testing.T) {
	l := testLogger()
	ctx := context.Background()

	options := map[string]interface{}{
		"errorCodes": map[string]interface{}{
			"GM_MESSAGE":     float64(1),
			"QUEUE_TOO_LONG": float64(2),
			"WRONG_USER":     float64(3),
		},
	}

	cases := []struct {
		reason   TvResultReason
		wantCode byte
	}{
		{TvResultGmMessage, 1},
		{TvResultQueueTooLong, 2},
		{TvResultWrongUser, 3},
	}
	for _, c := range cases {
		t.Run(string(c.reason), func(t *testing.T) {
			got := TvSendMessageResultErrorBody(c.reason)(l, ctx)(options)
			want := []byte{1, c.wantCode}
			if len(got) != len(want) {
				t.Fatalf("byte count: got %d, want %d", len(got), len(want))
			}
			if got[0] != want[0] || got[1] != want[1] {
				t.Fatalf("payload: got % X, want % X", got, want)
			}
		})
	}
}

// TestTvSendMessageResultErrorBodyUnconfiguredReasonDegrades confirms a
// reason missing from the tenant "errorCodes" table degrades to 99.
func TestTvSendMessageResultErrorBodyUnconfiguredReasonDegrades(t *testing.T) {
	l := testLogger()
	ctx := context.Background()
	options := map[string]interface{}{
		"errorCodes": map[string]interface{}{
			"GM_MESSAGE": float64(1),
		},
	}
	got := TvSendMessageResultErrorBody(TvResultQueueTooLong)(l, ctx)(options)
	want := []byte{1, 99}
	if len(got) != len(want) {
		t.Fatalf("byte count: got %d, want %d", len(got), len(want))
	}
	if got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("payload: got % X, want % X", got, want)
	}
}

// TestTvClearMessageBodyEmptyPayload confirms no resolution is applied and
// the payload is empty.
func TestTvClearMessageBodyEmptyPayload(t *testing.T) {
	l := testLogger()
	ctx := context.Background()
	got := TvClearMessageBody()(l, ctx)(map[string]interface{}{})
	if len(got) != 0 {
		t.Errorf("payload length: got %d, want 0", len(got))
	}
}

// TestTvSendMessageResultSuccessBodyPayload confirms no resolution is applied
// and the payload is a bare 00.
func TestTvSendMessageResultSuccessBodyPayload(t *testing.T) {
	l := testLogger()
	ctx := context.Background()
	got := TvSendMessageResultSuccessBody()(l, ctx)(map[string]interface{}{})
	if len(got) != 1 || got[0] != 0 {
		t.Errorf("payload: got % X, want [00]", got)
	}
}
