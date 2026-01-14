package producer_test

import (
	"atlas-reactors/kafka/producer"
	"context"
	"testing"

	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

func testLogger() logrus.FieldLogger {
	l, _ := test.NewNullLogger()
	return l
}

func testContext() context.Context {
	t, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return tenant.WithContext(context.Background(), t)
}

func TestProviderImpl_ReturnsFunction(t *testing.T) {
	l := testLogger()
	ctx := testContext()

	// ProviderImpl should return a function chain without panicking
	assert.NotPanics(t, func() {
		provider := producer.ProviderImpl(l)
		assert.NotNil(t, provider)

		ctxProvider := provider(ctx)
		assert.NotNil(t, ctxProvider)
	})
}

func TestProviderImpl_WithNilContext(t *testing.T) {
	l := testLogger()

	// ProviderImpl should handle nil context gracefully in the first call
	assert.NotPanics(t, func() {
		provider := producer.ProviderImpl(l)
		assert.NotNil(t, provider)
	})
}
