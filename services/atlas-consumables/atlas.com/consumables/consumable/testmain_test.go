package consumable

import (
	"atlas-consumables/kafka/message/consumable"
	"atlas-consumables/kafka/message/monsterbook"
	"context"
	"log"
	"os"
	"sync"
	"testing"

	assetMsg "atlas-consumables/kafka/message/asset"
	compartmentmsg "atlas-consumables/kafka/message/compartment"

	foodmsg "atlas-consumables/kafka/message/food"
	monsterMsg "atlas-consumables/kafka/message/monster"

	sagamsg "atlas-consumables/kafka/message/saga"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

// emitted captures everything this package's tests produce to Kafka. Installed
// once for the package; individual tests call emitted.Reset() rather than
// reinstalling the manager.
var emitted *producertest.Capture

// stoppedReader is the KafkaReader handed to the consumer manager in tests. It
// never fetches: the consumers registered below exist purely so
// Manager.RegisterHandler finds a consumer for the topic, which the
// once-handler registration inside RequestItemConsume and friends requires.
type stoppedReader struct{}

func (stoppedReader) FetchMessage(ctx context.Context) (kafka.Message, error) {
	<-ctx.Done()
	return kafka.Message{}, ctx.Err()
}

func (stoppedReader) CommitMessages(_ context.Context, _ ...kafka.Message) error { return nil }

func (stoppedReader) Close() error { return nil }

// registerConsumers puts a consumer in the manager's map for each topic whose
// once-handlers this package's tests register. The context handed in is
// already cancelled, so every consumer goroutine returns on its first loop
// iteration and nothing dials a broker.
func registerConsumers(tokens ...topic.Token) {
	consumer.ResetInstance()
	m := consumer.GetManager(
		consumer.ConfigReaderProducer(func(_ kafka.ReaderConfig) consumer.KafkaReader { return stoppedReader{} }),
		consumer.ConfigEngine(consumer.EngineReader),
	)

	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	wg := &sync.WaitGroup{}
	for _, tk := range tokens {
		t, err := topic.EnvProvider(l)(tk)()
		if err != nil {
			log.Fatalf("failed to resolve %s: %v", tk, err)
		}
		m.AddConsumer(l, ctx, wg)(consumer.NewConfig(nil, "test", t, "test-group"))
	}
	wg.Wait()
}

func TestMain(m *testing.M) {
	setEnv := func(k topic.Token) {
		if err := os.Setenv(string(k), string(k)); err != nil {
			log.Fatalf("failed to set %s: %v", k, err)
		}
	}
	setEnv(assetMsg.EnvEventTopicStatus)
	setEnv(compartmentmsg.EnvCommandTopic)
	setEnv(compartmentmsg.EnvEventTopicStatus)
	setEnv(consumable.EnvEventTopic)
	setEnv(foodmsg.EnvEventTopic)
	setEnv(monsterMsg.EnvCommandTopic)
	setEnv(monsterMsg.EnvEventTopicCatch)
	setEnv(monsterbook.EnvCommandTopic)
	setEnv(sagamsg.EnvStatusEventTopic)
	emitted = producertest.InstallCapturing()
	registerConsumers(compartmentmsg.EnvEventTopicStatus, assetMsg.EnvEventTopicStatus)
	os.Exit(m.Run())
}
