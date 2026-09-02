package producer

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

// Provider resolves a topic token to a ready-to-use MessageProducer.
type Provider func(token topic.Token) MessageProducer

// ProviderImpl is the canonical provider: span + tenant header decorators
// over the manager-owned writer for the token's topic.
func ProviderImpl(l logrus.FieldLogger) func(ctx context.Context) Provider {
	return func(ctx context.Context) Provider {
		sd := SpanHeaderDecorator(ctx)
		td := TenantHeaderDecorator(ctx)
		ed := EnvHeaderDecorator(ctx)
		return func(token topic.Token) MessageProducer {
			return Produce(l)(ManagerWriterProvider(l)(token))(sd, td, ed)
		}
	}
}
