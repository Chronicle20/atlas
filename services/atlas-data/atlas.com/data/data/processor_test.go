package data

import (
	"encoding/json"
	"testing"
	"time"
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
