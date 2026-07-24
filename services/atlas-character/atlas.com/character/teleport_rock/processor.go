package teleport_rock

import (
	"atlas-character/kafka/message"
	teleportrock2 "atlas-character/kafka/message/teleportrock"
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Typed validation errors: the REST path maps these to HTTP status; the async
// game path (AddMapAndEmit/RemoveMapAndEmit) maps them to ERROR status events.
var (
	ErrMapNotAllowed = errors.New("map not allowed")
	ErrListFull      = errors.New("list full")
	ErrDuplicate     = errors.New("duplicate map")
	ErrNotFound      = errors.New("map not found")
)

// reasonForError maps a validation sentinel to its wire ERROR reason (game path).
func reasonForError(err error) (string, bool) {
	switch {
	case errors.Is(err, ErrMapNotAllowed):
		return teleportrock2.ErrorReasonMapNotAllowed, true
	case errors.Is(err, ErrListFull):
		return teleportrock2.ErrorReasonListFull, true
	case errors.Is(err, ErrDuplicate):
		return teleportrock2.ErrorReasonDuplicate, true
	case errors.Is(err, ErrNotFound):
		return teleportrock2.ErrorReasonNotFound, true
	default:
		return "", false
	}
}

type Processor interface {
	GetByCharacterId(characterId uint32) (Model, error)
	AddMap(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error
	AddMapAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error
	Add(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) (Model, error)
	RemoveMap(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error
	RemoveMapAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error
	Remove(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) (Model, error)
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
// It is a thin wrapper over addMap that preserves the async game path's
// existing behavior: a typed validation error becomes a buffered ERROR event
// (and a nil return), while an infrastructure error propagates.
func (p *ProcessorImpl) AddMap(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error {
	return func(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error {
		_, err := p.addMap(mb, transactionId, worldId, characterId, mapId, vip)
		if err == nil {
			return nil
		}
		if reason, ok := reasonForError(err); ok {
			p.l.Warnf("Character [%d] attempted to register map [%d] (vip=%v): %s.", characterId, mapId, vip, reason)
			return mb.Put(teleportrock2.EnvEventTopicStatus, errorEventProvider(transactionId, worldId, characterId, vip, reason))
		}
		p.l.WithError(err).Errorf("Unable to register map [%d] for character [%d].", mapId, characterId)
		return err
	}
}

// addMap validates then mutates. On validation failure it returns a typed
// error and buffers nothing. On success it buffers LIST_UPDATED and reports
// the post-mutation Model.
func (p *ProcessorImpl) addMap(mb *message.Buffer, transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) (Model, error) {
	var updated Model
	txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		es, err := getByCharacterId(tx, p.t.Id(), characterId)
		if err != nil {
			return err
		}
		m := modelFromEntities(characterId, es)
		list := m.List(vip)

		if !EligibleForRegistration(mapId) {
			return ErrMapNotAllowed
		}
		if len(list) >= Capacity(vip) {
			return ErrListFull
		}
		if m.Contains(vip, mapId) {
			return ErrDuplicate
		}

		newList := append(append([]_map.Id{}, list...), mapId)
		if err := replaceList(tx, p.t.Id(), characterId, ListType(vip), newList); err != nil {
			return err
		}
		b := NewBuilder().SetCharacterId(characterId)
		if vip {
			updated = b.SetRegular(m.Regular()).SetVip(newList).Build()
		} else {
			updated = b.SetRegular(newList).SetVip(m.Vip()).Build()
		}
		p.l.Debugf("Registered map [%d] for character [%d] (vip=%v, %d entries).", mapId, characterId, vip, len(newList))
		return mb.Put(teleportrock2.EnvEventTopicStatus, listUpdatedEventProvider(transactionId, worldId, characterId, vip, true, newList))
	})
	return updated, txErr
}

// Add is the synchronous REST-facing counterpart to AddMap: it propagates the
// typed validation error to the caller (for HTTP status mapping) instead of
// converting it to a buffered ERROR event, and emits LIST_UPDATED for real on
// success.
func (p *ProcessorImpl) Add(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) (Model, error) {
	var updated Model
	err := message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		m, e := p.addMap(buf, transactionId, worldId, characterId, mapId, vip)
		if e != nil {
			return e // Emit discards the buffer; no event on failure
		}
		updated = m
		return nil
	})
	return updated, err
}

func (p *ProcessorImpl) RemoveMapAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error {
	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		return p.RemoveMap(buf)(transactionId, worldId, characterId, mapId, vip)
	})
}

// RemoveMap deletes a map from the selected list and compacts the remaining
// slots to a contiguous prefix (design §3). It is a thin wrapper over
// removeMap that preserves the async game path's existing behavior: a typed
// validation error becomes a buffered ERROR event (and a nil return), while
// an infrastructure error propagates.
func (p *ProcessorImpl) RemoveMap(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error {
	return func(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error {
		_, err := p.removeMap(mb, transactionId, worldId, characterId, mapId, vip)
		if err == nil {
			return nil
		}
		if reason, ok := reasonForError(err); ok {
			p.l.Warnf("Character [%d] attempted to remove map [%d] (vip=%v): %s.", characterId, mapId, vip, reason)
			return mb.Put(teleportrock2.EnvEventTopicStatus, errorEventProvider(transactionId, worldId, characterId, vip, reason))
		}
		p.l.WithError(err).Errorf("Unable to remove map [%d] for character [%d].", mapId, characterId)
		return err
	}
}

// removeMap validates then mutates. On validation failure it returns a typed
// error and buffers nothing. On success it buffers LIST_UPDATED and reports
// the post-mutation Model.
func (p *ProcessorImpl) removeMap(mb *message.Buffer, transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) (Model, error) {
	var updated Model
	txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		es, err := getByCharacterId(tx, p.t.Id(), characterId)
		if err != nil {
			return err
		}
		m := modelFromEntities(characterId, es)
		if !m.Contains(vip, mapId) {
			return ErrNotFound
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
		b := NewBuilder().SetCharacterId(characterId)
		if vip {
			updated = b.SetRegular(m.Regular()).SetVip(newList).Build()
		} else {
			updated = b.SetRegular(newList).SetVip(m.Vip()).Build()
		}
		p.l.Debugf("Removed map [%d] for character [%d] (vip=%v, %d entries).", mapId, characterId, vip, len(newList))
		return mb.Put(teleportrock2.EnvEventTopicStatus, listUpdatedEventProvider(transactionId, worldId, characterId, vip, false, newList))
	})
	return updated, txErr
}

// Remove is the synchronous REST-facing counterpart to RemoveMap: it
// propagates the typed validation error to the caller (for HTTP status
// mapping) instead of converting it to a buffered ERROR event, and emits
// LIST_UPDATED for real on success.
func (p *ProcessorImpl) Remove(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) (Model, error) {
	var updated Model
	err := message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		m, e := p.removeMap(buf, transactionId, worldId, characterId, mapId, vip)
		if e != nil {
			return e // Emit discards the buffer; no event on failure
		}
		updated = m
		return nil
	})
	return updated, err
}
