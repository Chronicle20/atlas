package pending_change

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	pendingchange2 "atlas-character/kafka/message/pending_change"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

func TestResolvedEventCarriesStatusAndReason(t *testing.T) {
	m := NewBuilder().
		SetId(uuid.New()).
		SetCharacterId(42).
		SetType(TypeNameChange).
		SetStatus(StatusRejected).
		SetReason("name_taken").
		SetRequestedName("Echo").
		SetSourceWorldId(world.Id(3)).
		SetTransactionId(uuid.New()).
		SetCreatedAt(time.Now()).
		SetExpiresAt(time.Now().Add(time.Hour)).
		Build()

	msgs, err := resolvedEventProvider(m)()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	var ev pendingchange2.StatusEvent[pendingchange2.ResolvedEventBody]
	if err := json.Unmarshal(msgs[0].Value, &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ev.Type != pendingchange2.EventTypeResolved {
		t.Fatalf("type = %s", ev.Type)
	}
	if ev.CharacterId != 42 || ev.WorldId != world.Id(3) {
		t.Fatalf("routing fields = %d / %d", ev.CharacterId, ev.WorldId)
	}
	if ev.Body.Status != StatusRejected || ev.Body.Reason != "name_taken" {
		t.Fatalf("body = %s / %s", ev.Body.Status, ev.Body.Reason)
	}
	if ev.Body.RequestedName != "Echo" {
		t.Fatalf("requestedName = %s", ev.Body.RequestedName)
	}
}

func TestCreatedEventCarriesExpiry(t *testing.T) {
	expiresAt := time.Now().Add(time.Hour).UTC()
	m := NewBuilder().
		SetId(uuid.New()).
		SetCharacterId(7).
		SetType(TypeWorldTransfer).
		SetStatus(StatusPending).
		SetDestinationWorldId(world.Id(2)).
		SetSourceWorldId(world.Id(1)).
		SetTransactionId(uuid.New()).
		SetCreatedAt(time.Now()).
		SetExpiresAt(expiresAt).
		Build()

	msgs, err := createdEventProvider(m)()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	var ev pendingchange2.StatusEvent[pendingchange2.CreatedEventBody]
	if err := json.Unmarshal(msgs[0].Value, &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ev.Type != pendingchange2.EventTypeCreated {
		t.Fatalf("type = %s", ev.Type)
	}
	if ev.CharacterId != 7 || ev.WorldId != world.Id(1) {
		t.Fatalf("routing fields = %d / %d", ev.CharacterId, ev.WorldId)
	}
	if ev.Body.ChangeType != TypeWorldTransfer {
		t.Fatalf("changeType = %s", ev.Body.ChangeType)
	}
	if ev.Body.DestinationWorldId != world.Id(2) {
		t.Fatalf("destinationWorldId = %d", ev.Body.DestinationWorldId)
	}
	if !ev.Body.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("expiresAt = %v, want %v", ev.Body.ExpiresAt, expiresAt)
	}
}
