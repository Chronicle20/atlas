package consumable

import (
	"atlas-consumables/kafka/message/consumable"
	"atlas-consumables/kafka/message/monsterbook"
	"os"
	"testing"

	assetMsg "atlas-consumables/kafka/message/asset"
	compartmentmsg "atlas-consumables/kafka/message/compartment"

	foodmsg "atlas-consumables/kafka/message/food"
	monsterMsg "atlas-consumables/kafka/message/monster"

	sagamsg "atlas-consumables/kafka/message/saga"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
)

// emitted captures everything this package's tests produce to Kafka. Installed
// once for the package; individual tests call emitted.Reset() rather than
// reinstalling the manager.
var emitted *producertest.Capture

func TestMain(m *testing.M) {
	os.Setenv(string(assetMsg.EnvEventTopicStatus), string(assetMsg.EnvEventTopicStatus))
	os.Setenv(string(compartmentmsg.EnvCommandTopic), string(compartmentmsg.EnvCommandTopic))
	os.Setenv(string(consumable.EnvEventTopic), string(consumable.EnvEventTopic))
	os.Setenv(string(foodmsg.EnvEventTopic), string(foodmsg.EnvEventTopic))
	os.Setenv(string(monsterMsg.EnvCommandTopic), string(monsterMsg.EnvCommandTopic))
	os.Setenv(string(monsterMsg.EnvEventTopicCatch), string(monsterMsg.EnvEventTopicCatch))
	os.Setenv(string(monsterbook.EnvCommandTopic), string(monsterbook.EnvCommandTopic))
	os.Setenv(string(sagamsg.EnvStatusEventTopic), string(sagamsg.EnvStatusEventTopic))
	emitted = producertest.InstallCapturing()
	os.Exit(m.Run())
}
