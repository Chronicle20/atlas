package test

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Encode runs an Encode closure under the given tenant context with a null
// logger and returns the produced bytes. It lets version-boundary tests assert
// byte-equality across tenant versions (e.g. v84 == v83) for the same packet.
func Encode(t *testing.T, ctx context.Context, encode func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte, options map[string]interface{}) []byte {
	t.Helper()
	l, _ := testlog.NewNullLogger()
	return encode(l, ctx)(options)
}

func RoundTrip(t *testing.T, ctx context.Context, encode func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte, decode func(logrus.FieldLogger, context.Context) func(*request.Reader, map[string]interface{}), options map[string]interface{}) {
	t.Helper()
	l, _ := testlog.NewNullLogger()

	bytes := encode(l, ctx)(options)
	req := request.Request(bytes)
	reader := request.NewRequestReader(&req, 0)
	decode(l, ctx)(&reader, options)

	if reader.Available() > 0 {
		t.Errorf("reader has %d unconsumed bytes after decode", reader.Available())
	}
}

// MovementRoundTrip is RoundTrip for a CLIENTBOUND packet whose body embeds
// model.Movement.
//
// It exists because model.Movement's per-element XOffset/YOffset pair is
// directional on GMS v87: CMovePath::Encode @0x6c70fe writes it, CMovePath::
// Decode @0x6c6e86 never reads it. Atlas therefore reads the pair off a v87
// client and does not write it back, so on v87 Encode is deliberately NOT the
// inverse of Decode for a clientbound packet.
//
// A plain RoundTrip does not catch that, and worse, does not fail either.
// request.Reader silently returns 0 for a read past the end without advancing
// the position, so decoding a v87 clientbound movement blob over-reads, corrupts
// the tail it decoded into, and STILL leaves Available() == 0. The assertion
// passes while asserting nothing.
//
// So on those tenants this helper skips the identity check rather than letting a
// vacuous one stand. The width of the emitted blob is not left unchecked: it is
// pinned per-version and per-direction by the move-path byte oracle in
// libs/atlas-packet/model (TestNormalElementMovementVersionBoundary and
// TestNormalElementOffsetsAreDirectional), which is the same OPAQUE_LEDGER
// arrangement the individual movement packets already rely on for their blob.
//
// Serverbound movement packets still use RoundTrip directly, but NOT because
// the identity is safe there in general — model.Movement.Encode is shared, so it
// omits the pair on a v87 tenant in either direction. It is safe only for the
// serverbound movement tests as they stand, which pass nil options and so build
// no NORMAL element for the gate to apply to. A serverbound test that adds a
// NORMAL element on a v87 tenant must come through here instead.
func MovementRoundTrip(t *testing.T, ctx context.Context, encode func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte, decode func(logrus.FieldLogger, context.Context) func(*request.Reader, map[string]interface{}), options map[string]interface{}) {
	t.Helper()

	if movementIsDirectional(ctx) {
		return
	}
	RoundTrip(t, ctx, encode, decode, options)
}

// movementIsDirectional reports whether the tenant is one where a clientbound
// model.Movement encode is not the inverse of its decode. GMS v87 only: v83/v84
// carry the pair on neither side, v92+ and JMS carry it on both.
func movementIsDirectional(ctx context.Context) bool {
	tm := tenant.MustFromContext(ctx)
	return tm.IsRegion("GMS") && tm.MajorAtLeast(87) && !tm.MajorAtLeast(92)
}
