package consumable

import (
	"atlas-consumables/kafka/message/consumable"
	"atlas-consumables/kafka/message/monsterbook"
	"log"
	"os"
	"testing"

	assetMsg "atlas-consumables/kafka/message/asset"
	compartmentmsg "atlas-consumables/kafka/message/compartment"

	foodmsg "atlas-consumables/kafka/message/food"
	monsterMsg "atlas-consumables/kafka/message/monster"

	sagamsg "atlas-consumables/kafka/message/saga"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

// emitted captures everything this package's tests produce to Kafka. Installed
// once for the package; individual tests call emitted.Reset() rather than
// reinstalling the manager.
var emitted *producertest.Capture

func TestMain(m *testing.M) {
	setEnv := func(k topic.Token) {
		if err := os.Setenv(string(k), string(k)); err != nil {
			log.Fatalf("failed to set %s: %v", k, err)
		}
	}
	setEnv(assetMsg.EnvEventTopicStatus)
	setEnv(compartmentmsg.EnvCommandTopic)
	setEnv(consumable.EnvEventTopic)
	setEnv(foodmsg.EnvEventTopic)
	setEnv(monsterMsg.EnvCommandTopic)
	setEnv(monsterMsg.EnvEventTopicCatch)
	setEnv(monsterbook.EnvCommandTopic)
	setEnv(sagamsg.EnvStatusEventTopic)
	emitted = producertest.InstallCapturing()
	os.Exit(m.Run())
}
