package socket

import (
	"atlas-channel/listener"
	"atlas-channel/server"
	"context"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func nullLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)
	return l
}

func TestKickSessionRejectsANonSessionModel(t *testing.T) {
	tm := testTenant(t)
	ctx := NewListenerContext(context.Background(), tm)

	var sc server.Model

	err := KickSession(nullLogger(), ctx, nil, sc)(listener.Session("not-a-model"))
	if err == nil {
		t.Fatal("KickSession returned a nil error for a non-session.Model value")
	}
	if !strings.Contains(err.Error(), "unexpected session type") {
		t.Fatalf("err = %q, want it to contain \"unexpected session type\"", err.Error())
	}
}
