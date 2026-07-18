package teleport_rock

import (
	"atlas-character/kafka/message"
	teleportrock2 "atlas-character/kafka/message/teleportrock"
	"context"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type Processor interface {
	GetByCharacterId(characterId uint32) (Model, error)
	AddMap(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error
	AddMapAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error
	RemoveMap(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error
	RemoveMapAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
		t:   tenant.MustFromContext(ctx),
	}
}

func (p *ProcessorImpl) GetByCharacterId(characterId uint32) (Model, error) {
	es, err := getByCharacterId(p.db.WithContext(p.ctx), p.t.Id(), characterId)
	if err != nil {
		return Model{}, err
	}
	return modelFromEntities(characterId, es), nil
}

func (p *ProcessorImpl) AddMapAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error {
	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		return p.AddMap(buf)(transactionId, worldId, characterId, mapId, vip)
	})
}

// AddMap registers the character's current map (server-derived, design §1 Q1)
// on the selected list. Validation failures buffer an ERROR status event and
// mutate nothing (FR-7: the client updates its UI only from the result packet).
func (p *ProcessorImpl) AddMap(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error {
	return func(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error {
		txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			es, err := getByCharacterId(tx, p.t.Id(), characterId)
			if err != nil {
				return err
			}
			m := modelFromEntities(characterId, es)
			list := m.List(vip)

			if !EligibleForRegistration(mapId) {
				p.l.Warnf("Character [%d] attempted to register ineligible map [%d].", characterId, mapId)
				return mb.Put(teleportrock2.EnvEventTopicStatus, errorEventProvider(transactionId, worldId, characterId, vip, teleportrock2.ErrorReasonMapNotAllowed))
			}
			if len(list) >= Capacity(vip) {
				p.l.Warnf("Character [%d] attempted to register map [%d] on a full list (vip=%v).", characterId, mapId, vip)
				return mb.Put(teleportrock2.EnvEventTopicStatus, errorEventProvider(transactionId, worldId, characterId, vip, teleportrock2.ErrorReasonListFull))
			}
			if m.Contains(vip, mapId) {
				p.l.Warnf("Character [%d] attempted to register duplicate map [%d] (vip=%v).", characterId, mapId, vip)
				return mb.Put(teleportrock2.EnvEventTopicStatus, errorEventProvider(transactionId, worldId, characterId, vip, teleportrock2.ErrorReasonDuplicate))
			}

			newList := append(append([]_map.Id{}, list...), mapId)
			if err := replaceList(tx, p.t.Id(), characterId, ListType(vip), newList); err != nil {
				return err
			}
			p.l.Debugf("Registered map [%d] for character [%d] (vip=%v, %d entries).", mapId, characterId, vip, len(newList))
			return mb.Put(teleportrock2.EnvEventTopicStatus, listUpdatedEventProvider(transactionId, worldId, characterId, vip, true, newList))
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Unable to register map [%d] for character [%d].", mapId, characterId)
			return txErr
		}
		return nil
	}
}

func (p *ProcessorImpl) RemoveMapAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error {
	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		return p.RemoveMap(buf)(transactionId, worldId, characterId, mapId, vip)
	})
}

// RemoveMap deletes a map from the selected list and compacts the remaining
// slots to a contiguous prefix (design §3).
func (p *ProcessorImpl) RemoveMap(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error {
	return func(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error {
		txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			es, err := getByCharacterId(tx, p.t.Id(), characterId)
			if err != nil {
				return err
			}
			m := modelFromEntities(characterId, es)
			if !m.Contains(vip, mapId) {
				p.l.Warnf("Character [%d] attempted to remove absent map [%d] (vip=%v).", characterId, mapId, vip)
				return mb.Put(teleportrock2.EnvEventTopicStatus, errorEventProvider(transactionId, worldId, characterId, vip, teleportrock2.ErrorReasonNotFound))
			}

			newList := make([]_map.Id, 0, len(m.List(vip)))
			for _, v := range m.List(vip) {
				if v != mapId {
					newList = append(newList, v)
				}
			}
			if err := replaceList(tx, p.t.Id(), characterId, ListType(vip), newList); err != nil {
				return err
			}
			p.l.Debugf("Removed map [%d] for character [%d] (vip=%v, %d entries).", mapId, characterId, vip, len(newList))
			return mb.Put(teleportrock2.EnvEventTopicStatus, listUpdatedEventProvider(transactionId, worldId, characterId, vip, false, newList))
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Unable to remove map [%d] for character [%d].", mapId, characterId)
			return txErr
		}
		return nil
	}
}
