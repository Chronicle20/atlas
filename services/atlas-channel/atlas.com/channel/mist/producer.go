package mist

import (
	mistmsg "atlas-channel/kafka/message/mist"

	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// CreateCommandProvider builds the CREATE command that asks atlas-maps to
// spawn a mist. Keyed on the map id so every mist command for one map lands on
// the same partition and is processed in cast order.
//
// The envelope's Tenant field is left zero-valued: atlas-maps' mist command
// consumer reads tenant from the Kafka message headers (TenantHeaderParser),
// not from the body, and producer.ProviderImpl attaches that header from the
// caller's context automatically -- mirroring the atlas-monsters producer's
// key convention without duplicating the now-inert body field.
func CreateCommandProvider(body mistmsg.CreateCommandBody) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(body.MapId))
	value := &mistmsg.Command[mistmsg.CreateCommandBody]{
		Type: mistmsg.CommandTypeCreate,
		Body: body,
	}
	return producer.SingleMessageProvider(key, value)
}
