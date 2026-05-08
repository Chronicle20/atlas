package data

import (
	"context"
	"io"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func nullLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

func TestHandleDataUpdated_UnknownTypeSkipped(t *testing.T) {
	handleDataUpdated(nullLogger(), context.Background(), event[dataUpdatedEventBody]{
		Type: "FUTURE",
		Body: dataUpdatedEventBody{TenantId: uuid.New().String(), Worker: WorkerMap},
	})
}

func TestHandleDataUpdated_WorkerMonsterSkipped(t *testing.T) {
	handleDataUpdated(nullLogger(), context.Background(), event[dataUpdatedEventBody]{
		Type: EventTypeDataUpdated,
		Body: dataUpdatedEventBody{TenantId: uuid.New().String(), Worker: "MONSTER"},
	})
}

func TestHandleDataUpdated_WorkerNPCSkipped(t *testing.T) {
	handleDataUpdated(nullLogger(), context.Background(), event[dataUpdatedEventBody]{
		Type: EventTypeDataUpdated,
		Body: dataUpdatedEventBody{TenantId: uuid.New().String(), Worker: "NPC"},
	})
}

func TestHandleDataUpdated_MalformedTenantId(t *testing.T) {
	handleDataUpdated(nullLogger(), context.Background(), event[dataUpdatedEventBody]{
		Type: EventTypeDataUpdated,
		Body: dataUpdatedEventBody{TenantId: "not-a-uuid", Worker: WorkerMap},
	})
}

func TestConsumerEnabled_Default(t *testing.T) {
	if v, ok := os.LookupEnv("DATA_EVENTS_CONSUMER_ENABLED"); ok {
		defer os.Setenv("DATA_EVENTS_CONSUMER_ENABLED", v)
	} else {
		defer os.Unsetenv("DATA_EVENTS_CONSUMER_ENABLED")
	}
	os.Unsetenv("DATA_EVENTS_CONSUMER_ENABLED")
	if !consumerEnabled() {
		t.Fatal("expected default true")
	}
	t.Setenv("DATA_EVENTS_CONSUMER_ENABLED", "false")
	if consumerEnabled() {
		t.Fatal("expected false")
	}
}
