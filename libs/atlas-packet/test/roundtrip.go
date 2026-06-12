package test

import (
	"context"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
	testlog "github.com/sirupsen/logrus/hooks/test"
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
