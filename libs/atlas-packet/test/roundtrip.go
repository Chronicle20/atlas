package test

import (
	"context"
	"testing"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

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
