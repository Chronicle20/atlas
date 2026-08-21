package playernpc

import (
	msg "atlas-player-npcs/kafka/message/playernpc"
	"context"
	"fmt"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// NewEmitter returns the Kafka-backed EventEmitter (see processor.go's
// EventEmitter doc): it publishes the design §7/§8.3 domain events to
// EVENT_TOPIC_PLAYER_NPC_STATUS. It is always invoked after the
// transaction that produced Event has committed (processor.go), so a
// publish failure here is logged at warn and otherwise swallowed -- the
// deploy/redeploy/remove already succeeded and must not roll back or
// block on a broker hiccup.
func NewEmitter(l logrus.FieldLogger, ctx context.Context) EventEmitter {
	return func(e Event) {
		provider, err := statusEventProvider(e)
		if err != nil {
			l.WithError(err).Warnf("Unable to build Player NPC status event [%s].", e.Type)
			return
		}
		if err := producer.ProviderImpl(l)(ctx)(msg.EnvEventTopicStatus)(provider); err != nil {
			l.WithError(err).Warnf("Unable to emit Player NPC status event [%s].", e.Type)
		}
	}
}

// statusEventProvider maps a domain Event to the wire StatusEvent its Type
// carries, per kafka/message/playernpc/kafka.go's doc: DEPLOYED/UPDATED
// carry StatusModel (single occupant), REMOVED carries StatusRemovedBody
// (single occupant -- RemoveById/Remove's loop, processor.go, emits one
// REMOVED per row), REPOSITIONED carries StatusRepositionedBody (every
// occupant a reorganize moved, in one message).
func statusEventProvider(e Event) (model.Provider[[]kafka.Message], error) {
	switch e.Type {
	case EventTypeDeployed:
		m, err := singleModel(e)
		if err != nil {
			return nil, err
		}
		return singleMessageProvider(m.CharacterId(), &msg.StatusEvent[msg.StatusModel]{Type: msg.EventTypeDeployed, Body: toStatusModel(m)}), nil
	case EventTypeUpdated:
		m, err := singleModel(e)
		if err != nil {
			return nil, err
		}
		return singleMessageProvider(m.CharacterId(), &msg.StatusEvent[msg.StatusModel]{Type: msg.EventTypeUpdated, Body: toStatusModel(m)}), nil
	case EventTypeRemoved:
		m, err := singleModel(e)
		if err != nil {
			return nil, err
		}
		body := msg.StatusRemovedBody{Id: m.Id(), ObjectId: m.ObjectId(), MapId: m.MapId(), WorldId: m.WorldId()}
		return singleMessageProvider(m.CharacterId(), &msg.StatusEvent[msg.StatusRemovedBody]{Type: msg.EventTypeRemoved, Body: body}), nil
	case EventTypeRepositioned:
		npcs := make([]msg.StatusRepositionedNpc, 0, len(e.Models))
		for _, m := range e.Models {
			npcs = append(npcs, msg.StatusRepositionedNpc{Id: m.Id(), ObjectId: m.ObjectId(), X: m.X(), Cy: m.Cy(), Fh: m.Fh(), Rx0: m.RX0(), Rx1: m.RX1()})
		}
		body := msg.StatusRepositionedBody{WorldId: e.WorldId, MapId: e.MapId, Npcs: npcs}
		key := producer.CreateKey(int(e.MapId))
		return producer.SingleMessageProvider(key, &msg.StatusEvent[msg.StatusRepositionedBody]{Type: msg.EventTypeRepositioned, Body: body}), nil
	default:
		return nil, fmt.Errorf("unknown Player NPC event type %q", e.Type)
	}
}

// singleModel returns e.Models' one occupant, per DEPLOYED/UPDATED/
// REMOVED's single-occupant contract (processor.go's Event doc).
func singleModel(e Event) (Model, error) {
	if len(e.Models) != 1 {
		return Model{}, fmt.Errorf("player NPC event %q expected exactly one model, got %d", e.Type, len(e.Models))
	}
	return e.Models[0], nil
}

func singleMessageProvider(characterId uint32, value interface{}) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	return producer.SingleMessageProvider(key, value)
}

// toStatusModel maps a domain Model to StatusModel, mirroring
// playernpc.Transform's REST mapping (resource.go) field for field.
func toStatusModel(m Model) msg.StatusModel {
	equipment := make([]msg.StatusEquipment, 0, len(m.Equipment()))
	for _, e := range m.Equipment() {
		equipment = append(equipment, msg.StatusEquipment{Slot: e.Slot(), ItemId: e.ItemId()})
	}
	return msg.StatusModel{
		Id:             m.Id(),
		CharacterId:    m.CharacterId(),
		Name:           m.Name(),
		WorldId:        m.WorldId(),
		MapId:          m.MapId(),
		ScriptId:       m.ScriptId(),
		ObjectId:       m.ObjectId(),
		Gender:         m.Gender(),
		Skin:           m.Skin(),
		Face:           m.Face(),
		Hair:           m.Hair(),
		JobId:          m.JobId(),
		X:              m.X(),
		Cy:             m.Cy(),
		Fh:             m.Fh(),
		Rx0:            m.RX0(),
		Rx1:            m.RX1(),
		Dir:            m.Dir(),
		WorldRank:      m.WorldRank(),
		OverallRank:    m.OverallRank(),
		WorldJobRank:   m.WorldJobRank(),
		OverallJobRank: m.OverallJobRank(),
		Equipment:      equipment,
		DeployedAt:     m.CreatedAt(),
	}
}
