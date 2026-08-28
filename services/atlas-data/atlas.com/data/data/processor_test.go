package data

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func TestDataUpdatedEventProvider_KeyIsTenantId(t *testing.T) {
	tenantId := "8b8d2bb0-2d1f-46b0-8c1c-1234567890ab"
	p := dataUpdatedEventProvider(tenantId, WorkerMonster, time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC))
	msgs, err := p()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("len(msgs) = %d, want 1", len(msgs))
	}
	if string(msgs[0].Key) != tenantId {
		t.Fatalf("key = %q, want %q", string(msgs[0].Key), tenantId)
	}
}

func TestDataUpdatedEventProvider_BodyShape(t *testing.T) {
	tenantId := "8b8d2bb0-2d1f-46b0-8c1c-1234567890ab"
	completedAt := time.Date(2026, 5, 8, 12, 30, 0, 0, time.UTC)
	p := dataUpdatedEventProvider(tenantId, WorkerMap, completedAt)
	msgs, _ := p()

	var ev event[dataUpdatedEventBody]
	if err := json.Unmarshal(msgs[0].Value, &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ev.Type != EventTypeDataUpdated {
		t.Fatalf("Type = %q, want %q", ev.Type, EventTypeDataUpdated)
	}
	if ev.Body.TenantId != tenantId {
		t.Fatalf("TenantId = %q", ev.Body.TenantId)
	}
	if ev.Body.Worker != WorkerMap {
		t.Fatalf("Worker = %q", ev.Body.Worker)
	}
	if ev.Body.CompletedAt != "2026-05-08T12:30:00Z" {
		t.Fatalf("CompletedAt = %q, want RFC3339 UTC", ev.Body.CompletedAt)
	}
}

func TestProducerEnabled_DefaultTrue(t *testing.T) {
	// Snapshot + restore env so other tests don't see our state.
	if v, ok := os.LookupEnv("DATA_EVENTS_PRODUCER_ENABLED"); ok {
		defer os.Setenv("DATA_EVENTS_PRODUCER_ENABLED", v)
	} else {
		defer os.Unsetenv("DATA_EVENTS_PRODUCER_ENABLED")
	}
	os.Unsetenv("DATA_EVENTS_PRODUCER_ENABLED")
	if !producerEnabled() {
		t.Fatal("expected default true when unset")
	}
}

func TestProducerEnabled_ExplicitFalse(t *testing.T) {
	t.Setenv("DATA_EVENTS_PRODUCER_ENABLED", "false")
	if producerEnabled() {
		t.Fatal("expected false when DATA_EVENTS_PRODUCER_ENABLED=false")
	}
}

func TestProducerEnabled_UnparseableTrue(t *testing.T) {
	t.Setenv("DATA_EVENTS_PRODUCER_ENABLED", "not-a-bool")
	if !producerEnabled() {
		t.Fatal("expected default true when unparseable")
	}
}

func TestWorkersIncludesItemMake(t *testing.T) {
	if WorkerItemMake != "ITEM_MAKE" {
		t.Fatalf("WorkerItemMake = %q, want %q", WorkerItemMake, "ITEM_MAKE")
	}
	count := 0
	for _, w := range Workers {
		if w == WorkerItemMake {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("Workers contains WorkerItemMake %d times, want 1", count)
	}
	if len(Workers) != 18 {
		t.Fatalf("len(Workers) = %d, want 18", len(Workers))
	}
}

func TestStartWorkerDispatchesItemMake(t *testing.T) {
	t.Setenv("DATA_EVENTS_PRODUCER_ENABLED", "false")
	tn, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	ctx := tenant.WithContext(context.Background(), tn)
	p := &ProcessorImpl{l: l, ctx: ctx, db: nil}
	tmp := t.TempDir()
	if err = p.StartWorker(WorkerItemMake, tmp); err != nil {
		t.Fatalf("StartWorker(WorkerItemMake, ...) returned error, want nil: %v", err)
	}
}
