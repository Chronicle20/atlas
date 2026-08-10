package handler

import (
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	cashcb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	swriter "github.com/Chronicle20/atlas/libs/atlas-socket/writer"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// The two wire codes the fake writer's options table resolves. They are
// deliberately NOT the values any real template uses: the point is to prove the
// handler goes through the config-resolved path (DOM-25) rather than emitting a
// hard-coded byte.
const (
	couponTestModeUseCouponFailed = byte(0x40)
	couponTestErrInvalidCode      = byte(0x51)
)

// couponTestWriterOptions is what a real tenant socket-config template supplies
// to the CASHSHOP_OPERATION writer. Supplying it here is what makes the encoded
// bytes decodable back to their symbolic keys.
func couponTestWriterOptions() map[string]interface{} {
	return map[string]interface{}{
		"operations": map[string]interface{}{
			cashcb.CashShopOperationUseCouponFailed: float64(couponTestModeUseCouponFailed),
		},
		"errors": map[string]interface{}{
			cashcb.CashShopOperationErrorInvalidCouponCode: float64(couponTestErrInvalidCode),
		},
	}
}

// The session needs a live, harmless connection because
// session.Model.announceEncrypted writes straight to it — discardConn is
// declared in character_damage_test.go and reused here.

type couponRedemptionCall struct {
	characterId uint32
	code        string
}

type couponHandlerEnv struct {
	t         *testing.T
	ctx       context.Context
	s         session.Model
	l         logrus.FieldLogger
	logs      *bytes.Buffer
	wp        writer.Producer
	published []couponRedemptionCall
	announced [][]byte
}

const couponTestCharacterId = uint32(4242)

// newCouponHandlerEnv builds a GMS v83 session + tenant ctx, a log sink, a
// capturing writer.Producer, and swaps the coupon-redemption seam so nothing
// touches Kafka. Precedent for the seam swap: installCashItemInSlotSeam in
// character_cash_item_use_test.go.
func newCouponHandlerEnv(t *testing.T) *couponHandlerEnv {
	t.Helper()

	ten := mustTenant(t, "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), ten)

	sessionId := uuid.New()
	s := session.NewSession(sessionId, ten, 0, discardConn{})
	session.AddSessionToRegistry(ten.Id(), s)
	t.Cleanup(func() { session.ClearRegistryForTenant(ten.Id()) })

	logs := &bytes.Buffer{}
	l := logrus.New()
	l.SetOutput(logs)
	l.SetLevel(logrus.DebugLevel)

	sp := session.NewProcessor(l, ctx)
	sp.SetCharacterId(sessionId, couponTestCharacterId)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).Build()
	updated := sp.SetField(sessionId, f)

	env := &couponHandlerEnv{t: t, ctx: ctx, s: updated, l: l, logs: logs}

	env.wp = func(name string) (swriter.BodyFunc, error) {
		if name != cashcb.CashShopOperationWriter {
			t.Errorf("announced via writer %q, want %q", name, cashcb.CashShopOperationWriter)
		}
		return func(bl logrus.FieldLogger, bctx context.Context) func(encoder packet.Encode) []byte {
			return func(encoder packet.Encode) []byte {
				b := encoder(bl, bctx)(couponTestWriterOptions())
				env.announced = append(env.announced, b)
				return b
			}
		}, nil
	}

	orig := couponRedemptionRequestFunc
	couponRedemptionRequestFunc = func(_ logrus.FieldLogger, _ context.Context, characterId uint32, code string) error {
		env.published = append(env.published, couponRedemptionCall{characterId: characterId, code: code})
		return nil
	}
	t.Cleanup(func() { couponRedemptionRequestFunc = orig })

	return env
}

func (e *couponHandlerEnv) handle(r *request.Reader) {
	e.t.Helper()
	CashShopCouponCodeHandleFunc(e.l, e.ctx, e.wp)(e.s, r, map[string]interface{}{})
}

func (e *couponHandlerEnv) commandsPublished() int { return len(e.published) }

func (e *couponHandlerEnv) lastPublishedCode() string {
	e.t.Helper()
	if len(e.published) == 0 {
		e.t.Fatal("no coupon redemption was published")
	}
	return e.published[len(e.published)-1].code
}

func (e *couponHandlerEnv) announcements() int { return len(e.announced) }

// lastAnnouncedErrorKey decodes the last announced CASHSHOP_OPERATION body
// (mode byte, error byte) and maps the error byte back through the same options
// table the writer resolved it with, so the assertion is on the symbolic key.
func (e *couponHandlerEnv) lastAnnouncedErrorKey() string {
	e.t.Helper()
	if len(e.announced) == 0 {
		e.t.Fatal("nothing was announced")
	}
	b := e.announced[len(e.announced)-1]
	if len(b) != 2 {
		e.t.Fatalf("announced body length %d, want 2 (mode + error)", len(b))
	}
	if b[0] != couponTestModeUseCouponFailed {
		e.t.Errorf("announced mode 0x%02X, want the resolved USE_COUPON_FAILED 0x%02X", b[0], couponTestModeUseCouponFailed)
	}
	if b[1] == couponTestErrInvalidCode {
		return cashcb.CashShopOperationErrorInvalidCouponCode
	}
	return "unresolved error byte 0x" + strings.ToUpper(hexByte(b[1]))
}

func hexByte(b byte) string {
	const digits = "0123456789abcdef"
	return string([]byte{digits[b>>4], digits[b&0x0F]})
}

func (e *couponHandlerEnv) logOutput() string { return e.logs.String() }

// couponPacket encodes a GMS v83 COUPON_CODE body: targetCharacter string, then
// the code string. (v83 has no nType byte, and the trailing guarded string is
// only emitted when targetCharacter is non-empty — see
// libs/atlas-packet/cash/serverbound/coupon_code.go.)
func couponPacket(t *testing.T, _ context.Context, targetCharacter string, code string) *request.Reader {
	t.Helper()
	raw := append(asciiString(targetCharacter), asciiString(code)...)
	if targetCharacter != "" {
		raw = append(raw, asciiString("")...)
	}
	req := request.Request(raw)
	reader := request.NewRequestReader(&req, 0)
	return &reader
}

func asciiString(s string) []byte {
	out := []byte{byte(len(s)), byte(len(s) >> 8)}
	return append(out, []byte(s)...)
}

// An empty or over-long code is answered locally with INVALID_COUPON_CODE and
// never reaches Kafka — FR-4.3, and the first line of brute-force defence.
// This gate is load-bearing rather than an optimization: gms_v48 has no
// client-side empty-code guard at all, so the server is the only thing
// stopping an empty submission there.
func TestCouponCodeHandlerShortCircuitsAnImplausibleCode(t *testing.T) {
	for _, c := range []struct{ name, code string }{
		{"empty", ""},
		{"whitespace only", "    "},
		{"over the column limit", strings.Repeat("A", 33)},
	} {
		t.Run(c.name, func(t *testing.T) {
			env := newCouponHandlerEnv(t)
			env.handle(couponPacket(t, env.ctx, "", c.code))
			if env.commandsPublished() != 0 {
				t.Errorf("published %d commands, want 0", env.commandsPublished())
			}
			if got := env.lastAnnouncedErrorKey(); got != cashcb.CashShopOperationErrorInvalidCouponCode {
				t.Errorf("announced %q, want %q", got, cashcb.CashShopOperationErrorInvalidCouponCode)
			}
		})
	}
}

// A plausible code is normalized ONCE, here, so the value on the wire to
// atlas-cashshop and the value in the database have the same shape.
func TestCouponCodeHandlerNormalizesBeforePublishing(t *testing.T) {
	env := newCouponHandlerEnv(t)
	env.handle(couponPacket(t, env.ctx, "", "  maple2026 "))
	if got := env.lastPublishedCode(); got != "MAPLE2026" {
		t.Errorf("published %q, want MAPLE2026", got)
	}
	if env.announcements() != 0 {
		t.Errorf("announced %d packets, want 0 — the reply comes from the status event", env.announcements())
	}
}

// The code is a secret: it must not appear in the logs.
func TestCouponCodeHandlerDoesNotLogTheCode(t *testing.T) {
	env := newCouponHandlerEnv(t)
	env.handle(couponPacket(t, env.ctx, "", "SECRETCODE"))
	if strings.Contains(env.logOutput(), "SECRETCODE") {
		t.Error("the coupon code leaked into the logs")
	}
}
