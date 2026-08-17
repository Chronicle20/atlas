package consumer

import (
	"context"
	"encoding/binary"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func testTenant(t *testing.T) tenant.Model {
	t.Helper()
	tn, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("unable to create test tenant: %s", err.Error())
	}
	return tn
}

func tenantHeaders(t *testing.T, tn tenant.Model) []kafka.Header {
	t.Helper()
	major := make([]byte, 2)
	binary.BigEndian.PutUint16(major, tn.MajorVersion())
	minor := make([]byte, 2)
	binary.BigEndian.PutUint16(minor, tn.MinorVersion())
	return []kafka.Header{
		{Key: tenant.ID, Value: []byte(tn.Id().String())},
		{Key: tenant.Region, Value: []byte(tn.Region())},
		{Key: tenant.MajorVersion, Value: major},
		{Key: tenant.MinorVersion, Value: minor},
	}
}

func TestEnvHeaderParserPutsTheHeaderOnTheContext(t *testing.T) {
	ctx := EnvHeaderParser(context.Background(), []kafka.Header{
		{Key: env.Key, Value: []byte("pr-123")},
	})
	if got := env.MustFromContext(ctx); got != env.Id("pr-123") {
		t.Fatalf("got %q, want \"pr-123\"", got)
	}
}

func TestEnvHeaderParserWithNoHeaderIsTheLegacyValue(t *testing.T) {
	ctx := EnvHeaderParser(context.Background(), nil)
	if got := env.MustFromContext(ctx); got != env.Id("") {
		t.Fatalf("got %q, want the empty id", got)
	}
}

func TestEnvHeaderParserDerivesFromTenantWhenNoHeaderIsPresent(t *testing.T) {
	tn := testTenant(t)
	reg := env.NewMapRegistry(env.Id("main"), time.Now)
	reg.ApplyTenant(tn.Id().String(), env.Id("pr-123"))
	env.SetRegistry(reg)
	t.Cleanup(func() { env.SetRegistry(nil) })

	base := tenant.WithContext(context.Background(), tn)
	ctx := EnvHeaderParser(base, nil)
	if got := env.MustFromContext(ctx); got != env.Id("pr-123") {
		t.Fatalf("got %q, want \"pr-123\" derived from the tenant", got)
	}
}

func TestEnvHeaderParserMarksAMismatch(t *testing.T) {
	// FR-7.7. The parser cannot return an error (its signature is
	// context-in/context-out), so it records the mismatch on the context
	// and the gate drops the message.
	tn := testTenant(t)
	reg := env.NewMapRegistry(env.Id("main"), time.Now)
	reg.ApplyTenant(tn.Id().String(), env.Id("pr-123"))
	env.SetRegistry(reg)
	t.Cleanup(func() { env.SetRegistry(nil) })

	base := tenant.WithContext(context.Background(), tn)
	ctx := EnvHeaderParser(base, []kafka.Header{{Key: env.Key, Value: []byte("main")}})
	if !env.Mismatched(ctx) {
		t.Fatal("mismatch not recorded; the gate would process the message")
	}
}

// TestSetHeaderParsersReconcilesInOrder drives headers through the actual
// SetHeaderParsers composition — the same config.headerParsers slice
// processMessage iterates in manager.go — rather than calling
// EnvHeaderParser directly with a hand-built context. This proves the
// registered order (TenantHeaderParser before EnvHeaderParser) is what
// makes reconciliation possible. If EnvHeaderParser ran first, the tenant
// would be absent from the context, tenantId would silently become "", and
// Reconcile would never detect the mismatch below.
func TestSetHeaderParsersReconcilesInOrder(t *testing.T) {
	tn := testTenant(t)
	reg := env.NewMapRegistry(env.Id("main"), time.Now)
	reg.ApplyTenant(tn.Id().String(), env.Id("pr-123"))
	env.SetRegistry(reg)
	t.Cleanup(func() { env.SetRegistry(nil) })

	config := SetHeaderParsers(TenantHeaderParser, EnvHeaderParser)(
		NewConfig([]string{"localhost:9092"}, "test", "topic", "group"),
	)

	headers := append(tenantHeaders(t, tn), kafka.Header{Key: env.Key, Value: []byte("main")})

	ctx := context.Background()
	for _, p := range config.headerParsers {
		ctx = p(ctx, headers)
	}

	if !env.Mismatched(ctx) {
		t.Fatal("expected the mismatch to be detected when parsers run in the registered order")
	}
}
