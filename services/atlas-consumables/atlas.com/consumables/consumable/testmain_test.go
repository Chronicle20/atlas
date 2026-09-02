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

// idleReader is a KafkaReader that never yields a message and never dials a
// broker. It exists so the package can register a real consumer for the
// compartment status topic: RequestItemConsume calls
// consumer.Manager.RegisterHandler, which hard-errors with "no consumer found
// for topic" unless one is registered, and an offline unit test has no broker
// to register against.
type idleReader struct{}

func (idleReader) FetchMessage(ctx context.Context) (kafka.Message, error) {
	<-ctx.Done()
	return kafka.Message{}, ctx.Err()
}

func (idleReader) CommitMessages(_ context.Context, _ ...kafka.Message) error { return nil }

func (idleReader) Close() error { return nil }

// registerCompartmentStatusConsumer wires the manager to idleReader and adds
// the one consumer RequestItemConsume needs. The context is cancelled before
// the consumer starts so its fetch loop exits immediately; only the map entry
// RegisterHandler looks up matters here. Returns the cancel func's waitgroup
// so TestMain can drain it before exiting.
func registerCompartmentStatusConsumer(t string) *sync.WaitGroup {
	l := logrus.New()
	l.SetOutput(os.Stderr)
	m := consumer.GetManager(
		consumer.ConfigEngine(consumer.EngineReader),
		consumer.ConfigReaderProducer(func(_ kafka.ReaderConfig) consumer.KafkaReader { return idleReader{} }),
	)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	wg := &sync.WaitGroup{}
	m.AddConsumer(l, ctx, wg)(consumer.NewConfig([]string{"localhost:9092"}, "compartment_status_event", t, "atlas-consumables-test"))
	return wg
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
	wg := registerCompartmentStatusConsumer(string(compartmentmsg.EnvEventTopicStatus))
	code := m.Run()
	wg.Wait()
	os.Exit(code)
}
