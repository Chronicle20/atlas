package data

import (
	"context"
	"io"
	"os"
	"testing"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func nullLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

func TestHandleDataUpdated_UnknownTypeSkipped(t *testing.T) {
	e := event[dataUpdatedEventBody]{
		Type: "SOME_FUTURE_TYPE",
		Body: dataUpdatedEventBody{TenantId: uuid.New().String(), Worker: WorkerMonster},
	}
	handleDataUpdated(nullLogger(), context.Background(), e)
}

func TestHandleDataUpdated_UnrelatedWorkerSkipped(t *testing.T) {
	e := event[dataUpdatedEventBody]{
		Type: EventTypeDataUpdated,
		Body: dataUpdatedEventBody{TenantId: uuid.New().String(), Worker: "MAP"},
	}
	handleDataUpdated(nullLogger(), context.Background(), e)
}

func TestHandleDataUpdated_MalformedTenantId(t *testing.T) {
	e := event[dataUpdatedEventBody]{
		Type: EventTypeDataUpdated,
		Body: dataUpdatedEventBody{TenantId: "not-a-uuid", Worker: WorkerMonster},
	}
	handleDataUpdated(nullLogger(), context.Background(), e)
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
	t.Setenv("DATA_EVENTS_CONSUMER_ENABLED", "garbage")
	if !consumerEnabled() {
		t.Fatal("expected default true on unparseable")
	}
}

func TestHandleDataUpdated_HappyPath_Smoke(t *testing.T) {
	tid := uuid.New()
	tm, _ := tenant.Create(tid, "GMS", 0, 83)
	ctx := tenant.WithContext(context.Background(), tm)
	e := event[dataUpdatedEventBody]{
		Type: EventTypeDataUpdated,
		Body: dataUpdatedEventBody{TenantId: tid.String(), Worker: WorkerMonster},
	}
	handleDataUpdated(nullLogger(), ctx, e)
}
