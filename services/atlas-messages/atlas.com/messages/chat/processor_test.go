package chat

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func setupRegistry(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(client)
}

func testTenantContext() context.Context {
	tm, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return tenant.WithContext(context.Background(), tm)
}

func testField() field.Model {
	return field.NewBuilder(0, 1, 100000000).Build()
}

func TestCaptureAndRecentInvolving(t *testing.T) {
	setupRegistry(t)
	l, _ := test.NewNullLogger()
	ctx := testTenantContext()
	p := NewProcessor(l, ctx)

	if err := p.Capture(testField(), 1, "Alice", "GENERAL", "first"); err != nil {
		t.Fatalf("Capture: %v", err)
	}
	time.Sleep(2 * time.Millisecond) // distinct unix-milli timestamps
	if err := p.Capture(testField(), 2, "Bob", "WHISPER", "second"); err != nil {
		t.Fatalf("Capture: %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	if err := p.Capture(testField(), 3, "Carol", "GENERAL", "uninvolved"); err != nil {
		t.Fatalf("Capture: %v", err)
	}

	lines, err := p.RecentInvolving([]uint32{1, 2})
	if err != nil {
		t.Fatalf("RecentInvolving: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %+v", len(lines), lines)
	}
	if lines[0].Text != "first" || lines[1].Text != "second" {
		t.Errorf("expected timestamp-ascending merge, got %+v", lines)
	}
	if lines[0].SenderName != "Alice" || lines[0].ChatType != "GENERAL" || lines[0].MapId != 100000000 {
		t.Errorf("line fields mismatch: %+v", lines[0])
	}
}

func TestRecentInvolvingEmptyBuffer(t *testing.T) {
	setupRegistry(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, testTenantContext())

	lines, err := p.RecentInvolving([]uint32{99})
	if err != nil {
		t.Fatalf("RecentInvolving: %v", err)
	}
	if len(lines) != 0 {
		t.Fatalf("expected empty, got %+v", lines)
	}
}

func TestConfigDefaults(t *testing.T) {
	if got := envInt("CHAT_CAPTURE_TEST_UNSET_VAR", 900); got != 900 {
		t.Errorf("default: got %d", got)
	}
	t.Setenv("CHAT_CAPTURE_TEST_SET_VAR", "42")
	if got := envInt("CHAT_CAPTURE_TEST_SET_VAR", 900); got != 42 {
		t.Errorf("env override: got %d", got)
	}
	t.Setenv("CHAT_CAPTURE_TEST_BAD_VAR", "notanint")
	if got := envInt("CHAT_CAPTURE_TEST_BAD_VAR", 900); got != 900 {
		t.Errorf("bad value falls back to default: got %d", got)
	}
}
