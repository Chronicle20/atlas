package _map

import (
	scriptData "atlas-maps/data/map/script"
	"atlas-maps/kafka/message"
	mapKafka "atlas-maps/kafka/message/map"
	"atlas-maps/kafka/message/mapactions"
	"atlas-maps/kafka/producer"
	"atlas-maps/map/character"
	monster2 "atlas-maps/map/monster"
	"atlas-maps/reactor"
	"atlas-maps/visit"
	"context"
	"errors"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	Enter(mb *message.Buffer) func(transactionId uuid.UUID, f field.Model, characterId uint32) error
	EnterAndEmit(transactionId uuid.UUID, f field.Model, characterId uint32) error
	Exit(mb *message.Buffer) func(transactionId uuid.UUID, f field.Model, characterId uint32) error
	ExitAndEmit(transactionId uuid.UUID, f field.Model, characterId uint32) error
	TransitionMap(mb *message.Buffer) func(transactionId uuid.UUID, newField field.Model, characterId uint32, oldField field.Model)
	TransitionMapAndEmit(transactionId uuid.UUID, newField field.Model, characterId uint32, oldField field.Model) error
	TransitionChannel(mb *message.Buffer) func(transactionId uuid.UUID, newField field.Model, oldChannelId channel.Id, characterId uint32)
	TransitionChannelAndEmit(transactionId uuid.UUID, newField field.Model, oldChannelId channel.Id, characterId uint32) error
	GetCharactersInMap(transactionId uuid.UUID, f field.Model) ([]uint32, error)
	GetCharactersInMapAllInstances(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id) ([]uint32, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	p   producer.Provider
	cp  character.Processor
	db  *gorm.DB
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, p producer.Provider, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		p:   p,
		cp:  character.NewProcessor(l, ctx),
		db:  db,
	}
}

func (p *ProcessorImpl) Enter(mb *message.Buffer) func(transactionId uuid.UUID, f field.Model, characterId uint32) error {
	return func(transactionId uuid.UUID, f field.Model, characterId uint32) error {
		p.cp.Enter(transactionId, f, characterId)

		isFirstVisit := false
		if p.db != nil {
			vp := visit.NewProcessor(p.l, p.ctx, p.db)
			_, err := vp.ByCharacterIdAndMapIdProvider(characterId, f.MapId())()
			if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
				isFirstVisit = true
				if err := vp.RecordVisit(characterId, f.MapId()); err != nil {
					p.l.WithError(err).Errorf("Failed to record visit for character [%d] map [%d].", characterId, f.MapId())
				}
			}
		}

		scripts, err := scriptData.NewProcessor(p.l, p.ctx).GetScripts(f.MapId())
		if err != nil {
			p.l.WithError(err).Warnf("Failed to fetch script names for map [%d]. Skipping map actions.", f.MapId())
		} else {
			if isFirstVisit && scripts.OnFirstUserEnter() != "" {
				_ = mb.Put(mapactions.EnvCommandTopic, enterMapActionsProvider(transactionId, f, characterId, scripts.OnFirstUserEnter(), "onFirstUserEnter"))
			}
			if scripts.OnUserEnter() != "" {
				_ = mb.Put(mapactions.EnvCommandTopic, enterMapActionsProvider(transactionId, f, characterId, scripts.OnUserEnter(), "onUserEnter"))
			}
		}

		go func() {
			_ = monster2.NewProcessor(p.l, p.ctx).SpawnMonsters(transactionId, f)
		}()
		go func() {
			_ = reactor.NewProcessor(p.l, p.ctx, p.p).SpawnAndEmit(transactionId, f)
		}()
		return mb.Put(mapKafka.EnvEventTopicMapStatus, enterMapProvider(transactionId, f, characterId))
	}
}

func (p *ProcessorImpl) EnterAndEmit(transactionId uuid.UUID, f field.Model, characterId uint32) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.Enter(buf)(transactionId, f, characterId)
	})
}

func (p *ProcessorImpl) Exit(mb *message.Buffer) func(transactionId uuid.UUID, f field.Model, characterId uint32) error {
	return func(transactionId uuid.UUID, f field.Model, characterId uint32) error {
		p.cp.Exit(transactionId, f, characterId)
		return mb.Put(mapKafka.EnvEventTopicMapStatus, exitMapProvider(transactionId, f, characterId))
	}
}

func (p *ProcessorImpl) ExitAndEmit(transactionId uuid.UUID, f field.Model, characterId uint32) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.Exit(buf)(transactionId, f, characterId)
	})
}

func (p *ProcessorImpl) TransitionMap(mb *message.Buffer) func(transactionId uuid.UUID, newField field.Model, characterId uint32, oldField field.Model) {
	return func(transactionId uuid.UUID, newField field.Model, characterId uint32, oldField field.Model) {
		_ = p.Exit(mb)(transactionId, oldField, characterId)
		_ = p.Enter(mb)(transactionId, newField, characterId)
	}
}

func (p *ProcessorImpl) TransitionMapAndEmit(transactionId uuid.UUID, newField field.Model, characterId uint32, oldField field.Model) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		p.TransitionMap(buf)(transactionId, newField, characterId, oldField)
		return nil
	})
}

func (p *ProcessorImpl) TransitionChannel(mb *message.Buffer) func(transactionId uuid.UUID, newField field.Model, oldChannelId channel.Id, characterId uint32) {
	return func(transactionId uuid.UUID, newField field.Model, oldChannelId channel.Id, characterId uint32) {
		oldField := field.NewBuilder(newField.WorldId(), oldChannelId, newField.MapId()).SetInstance(newField.Instance()).Build()
		_ = p.Exit(mb)(transactionId, oldField, characterId)
		_ = p.Enter(mb)(transactionId, newField, characterId)
	}
}

func (p *ProcessorImpl) TransitionChannelAndEmit(transactionId uuid.UUID, newField field.Model, oldChannelId channel.Id, characterId uint32) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		p.TransitionChannel(buf)(transactionId, newField, oldChannelId, characterId)
		return nil
	})
}

func (p *ProcessorImpl) GetCharactersInMap(transactionId uuid.UUID, f field.Model) ([]uint32, error) {
	return p.cp.GetCharactersInMap(transactionId, f)
}

func (p *ProcessorImpl) GetCharactersInMapAllInstances(transactionId uuid.UUID, worldId world.Id, channelId channel.Id, mapId _map.Id) ([]uint32, error) {
	return p.cp.GetCharactersInMapAllInstances(transactionId, worldId, channelId, mapId)
}
