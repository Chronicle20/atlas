package producer

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func TestProviderImplComposesSpanAndTenantHeaders(t *testing.T) {
	ResetInstance()
	t.Cleanup(ResetInstance)

	mw := &MockWriter{topic: "provider-test-topic"}
	GetManager(ConfigWriterFactory(func(topicName string) Writer { return mw }))
	t.Setenv("EVENT_TOPIC_PROVIDER_TEST", "provider-test-topic")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)

	l, _ := test.NewNullLogger()

	var p Provider = ProviderImpl(l)(ctx) // compile-time: returns the named Provider type
	if err := p("EVENT_TOPIC_PROVIDER_TEST")(model.FixedProvider([]kafka.Message{{Value: []byte("v")}})); err != nil {
		t.Fatalf("produce: %v", err)
	}

	if len(mw.writtenMessages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(mw.writtenMessages))
	}
	headers := map[string]string{}
	for _, h := range mw.writtenMessages[0].Headers {
		headers[h.Key] = string(h.Value)
	}
	if headers[tenant.ID] != ten.Id().String() {
		t.Errorf("missing/wrong tenant id header: %q", headers[tenant.ID])
	}
	if headers[tenant.Region] != "GMS" {
		t.Errorf("missing/wrong region header: %q", headers[tenant.Region])
	}
}
