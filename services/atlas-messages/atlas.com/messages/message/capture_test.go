package message

import (
	"atlas-messages/chat"
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestCaptureLineSwallowsRedisOutage proves that captureLine never surfaces a
// Redis failure to its caller: chat capture is best-effort and must not
// break message delivery. It simulates an outage by closing the backing
// miniredis instance out from under an already-initialized registry, then
// calls captureLine directly and asserts (a) it returns nothing the caller
// could observe as failure (its signature has no error return) and (b) the
// failure was logged at Warn so it isn't silently lost either.
func TestCaptureLineSwallowsRedisOutage(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run: %v", err)
	}
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	chat.InitRegistry(client)
	mr.Close() // simulate a Redis outage for all subsequent commands

	l, hook := test.NewNullLogger()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	ctx := tenant.WithContext(context.Background(), tm)
	p := &ProcessorImpl{l: l, ctx: ctx}
	f := field.NewBuilder(0, 1, 100000000).Build()

	// If capture failure were not swallowed, this would panic or the test
	// harness would otherwise surface it; completing normally is the point.
	p.captureLine(f, 1, "Alice", "GENERAL", "hello")

	foundWarn := false
	for _, e := range hook.AllEntries() {
		if e.Level == logrus.WarnLevel {
			foundWarn = true
		}
	}
	if !foundWarn {
		t.Errorf("expected a Warn-level log entry on redis outage, got: %+v", hook.AllEntries())
	}
}
