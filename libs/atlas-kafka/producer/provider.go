package producer

import (
	"context"

	"github.com/sirupsen/logrus"
)

// Provider resolves a topic token to a ready-to-use MessageProducer.
type Provider func(token string) MessageProducer

// ProviderImpl is the canonical provider: span + tenant header decorators
// over the manager-owned writer for the token's topic.
func ProviderImpl(l logrus.FieldLogger) func(ctx context.Context) Provider {
	return func(ctx context.Context) Provider {
		sd := SpanHeaderDecorator(ctx)
		td := TenantHeaderDecorator(ctx)
		return func(token string) MessageProducer {
			return Produce(l)(ManagerWriterProvider(l)(token))(sd, td)
		}
	}
}
