package minigame

import (
	minigame2 "atlas-channel/kafka/message/minigame"
	"context"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	return &Processor{l: l, ctx: ctx}
}

// InFieldModelProvider retrieves every mini-game room currently registered
// in field f from atlas-mini-games (task-16 endpoint), for map-entry balloon
// spawn (task-19).
func (p *Processor) InFieldModelProvider(f field.Model) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestInField(f), Extract, model.Filters[Model]())
}

// ForEachInField applies o to every mini-game room in field f.
func (p *Processor) ForEachInField(f field.Model, o model.Operator[Model]) error {
	return model.ForEachSlice(p.InFieldModelProvider(f), o, model.ParallelExecute())
}

// MemberModelProvider retrieves the (0-or-1) mini-game room characterId is
// currently seated in (owner or visitor) from atlas-mini-games.
func (p *Processor) MemberModelProvider(characterId uint32) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByMember(characterId), Extract, model.Filters[Model]())
}

// InGame reports whether characterId is currently seated in a mini-game room.
// Used to block cash-shop / MTS entry while in a room (a player must not migrate
// out of the channel while seated at a mini-game table).
func (p *Processor) InGame(characterId uint32) (bool, error) {
	rooms, err := p.MemberModelProvider(characterId)()
	if err != nil {
		return false, err
	}
	return len(rooms) > 0, nil
}

func (p *Processor) Create(f field.Model, characterId uint32, roomType byte, title string, private bool, password string, pieceType byte) error {
	return producer.ProviderImpl(p.l)(p.ctx)(minigame2.EnvCommandTopic)(CreateCommandProvider(uuid.New(), f, characterId, roomType, title, private, password, pieceType))
}

func (p *Processor) Visit(f field.Model, characterId uint32, roomId uint32, password string) error {
	return producer.ProviderImpl(p.l)(p.ctx)(minigame2.EnvCommandTopic)(VisitCommandProvider(uuid.New(), f, characterId, roomId, password))
}

func (p *Processor) Leave(f field.Model, characterId uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(minigame2.EnvCommandTopic)(LeaveCommandProvider(uuid.New(), f, characterId))
}

func (p *Processor) Chat(f field.Model, characterId uint32, message string) error {
	return producer.ProviderImpl(p.l)(p.ctx)(minigame2.EnvCommandTopic)(ChatCommandProvider(uuid.New(), f, characterId, message))
}

func (p *Processor) Ready(f field.Model, characterId uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(minigame2.EnvCommandTopic)(ReadyCommandProvider(uuid.New(), f, characterId))
}

func (p *Processor) Unready(f field.Model, characterId uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(minigame2.EnvCommandTopic)(UnreadyCommandProvider(uuid.New(), f, characterId))
}

func (p *Processor) Start(f field.Model, characterId uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(minigame2.EnvCommandTopic)(StartCommandProvider(uuid.New(), f, characterId))
}

func (p *Processor) GiveUp(f field.Model, characterId uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(minigame2.EnvCommandTopic)(GiveUpCommandProvider(uuid.New(), f, characterId))
}

func (p *Processor) RequestTie(f field.Model, characterId uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(minigame2.EnvCommandTopic)(RequestTieCommandProvider(uuid.New(), f, characterId))
}

func (p *Processor) RequestRetreat(f field.Model, characterId uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(minigame2.EnvCommandTopic)(RequestRetreatCommandProvider(uuid.New(), f, characterId))
}

func (p *Processor) Expel(f field.Model, characterId uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(minigame2.EnvCommandTopic)(ExpelCommandProvider(uuid.New(), f, characterId))
}

func (p *Processor) Skip(f field.Model, characterId uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(minigame2.EnvCommandTopic)(SkipCommandProvider(uuid.New(), f, characterId))
}

func (p *Processor) MoveStone(f field.Model, characterId uint32, x uint32, y uint32, stoneType byte) error {
	return producer.ProviderImpl(p.l)(p.ctx)(minigame2.EnvCommandTopic)(MoveStoneCommandProvider(uuid.New(), f, characterId, x, y, stoneType))
}

func (p *Processor) FlipCard(f field.Model, characterId uint32, first bool, cardIndex byte) error {
	return producer.ProviderImpl(p.l)(p.ctx)(minigame2.EnvCommandTopic)(FlipCardCommandProvider(uuid.New(), f, characterId, first, cardIndex))
}

func (p *Processor) AnswerTie(f field.Model, characterId uint32, accept bool) error {
	return producer.ProviderImpl(p.l)(p.ctx)(minigame2.EnvCommandTopic)(AnswerTieCommandProvider(uuid.New(), f, characterId, accept))
}

func (p *Processor) AnswerRetreat(f field.Model, characterId uint32, accept bool) error {
	return producer.ProviderImpl(p.l)(p.ctx)(minigame2.EnvCommandTopic)(AnswerRetreatCommandProvider(uuid.New(), f, characterId, accept))
}

func (p *Processor) ExitAfterGame(f field.Model, characterId uint32, cancel bool) error {
	return producer.ProviderImpl(p.l)(p.ctx)(minigame2.EnvCommandTopic)(ExitAfterGameCommandProvider(uuid.New(), f, characterId, cancel))
}
