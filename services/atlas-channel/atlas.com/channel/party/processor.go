package party

import (
	party2 "atlas-channel/kafka/message/party"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

type Processor interface {
	Create(characterId uint32) error
	Leave(partyId uint32, characterId uint32) error
	Expel(partyId uint32, characterId uint32, targetCharacterId uint32) error
	ChangeLeader(partyId uint32, characterId uint32, targetCharacterId uint32) error
	RequestInvite(characterId uint32, targetCharacterId uint32) error
	GetById(partyId uint32) (Model, error)
	ByIdProvider(partyId uint32) model.Provider[Model]
	GetByMemberId(memberId uint32) (Model, error)
	ByMemberIdProvider(memberId uint32) model.Provider[Model]
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
	return p
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) Create(characterId uint32) error {
	p.l.Debugf("Character [%d] attempting to create a party.", characterId)
	return producer.ProviderImpl(p.l)(p.ctx)(party2.EnvCommandTopic)(CreateCommandProvider(characterId))
}

func (p *ProcessorImpl) Leave(partyId uint32, characterId uint32) error {
	p.l.Debugf("Character [%d] attempting to leave party [%d].", characterId, partyId)
	return producer.ProviderImpl(p.l)(p.ctx)(party2.EnvCommandTopic)(LeaveCommandProvider(characterId, partyId, characterId, false))
}

func (p *ProcessorImpl) Expel(partyId uint32, characterId uint32, targetCharacterId uint32) error {
	p.l.Debugf("Character [%d] attempting to expel [%d] from party [%d].", characterId, targetCharacterId, partyId)
	return producer.ProviderImpl(p.l)(p.ctx)(party2.EnvCommandTopic)(LeaveCommandProvider(characterId, partyId, targetCharacterId, true))
}

func (p *ProcessorImpl) ChangeLeader(partyId uint32, characterId uint32, targetCharacterId uint32) error {
	p.l.Debugf("Character [%d] attempting to pass leadership to [%d] in party [%d].", characterId, targetCharacterId, partyId)
	return producer.ProviderImpl(p.l)(p.ctx)(party2.EnvCommandTopic)(ChangeLeaderCommandProvider(characterId, partyId, targetCharacterId))
}

func (p *ProcessorImpl) RequestInvite(characterId uint32, targetCharacterId uint32) error {
	p.l.Debugf("Character [%d] attempting to invite [%d] to a party.", characterId, targetCharacterId)
	return producer.ProviderImpl(p.l)(p.ctx)(party2.EnvCommandTopic)(RequestInviteCommandProvider(characterId, targetCharacterId))
}

func (p *ProcessorImpl) GetById(partyId uint32) (Model, error) {
	return p.ByIdProvider(partyId)()
}

func (p *ProcessorImpl) ByIdProvider(partyId uint32) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(partyId), Extract)
}

func (p *ProcessorImpl) GetByMemberId(memberId uint32) (Model, error) {
	return p.ByMemberIdProvider(memberId)()
}

func (p *ProcessorImpl) ByMemberIdProvider(memberId uint32) model.Provider[Model] {
	rp := requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByMemberId(memberId), Extract, model.Filters[Model]())
	return model.FirstProvider(rp, model.Filters[Model]())
}

func FilteredMemberProvider(filters ...model.Filter[MemberModel]) func(p model.Provider[Model]) model.Provider[[]MemberModel] {
	return func(p model.Provider[Model]) model.Provider[[]MemberModel] {
		return model.FilteredProvider(model.Map(model.Always(Model.Members))(p), filters)
	}
}

func MemberToMemberIdMapper(mp model.Provider[[]MemberModel]) model.Provider[[]uint32] {
	return model.SliceMap(model.Always(MemberModel.Id))(mp)(model.ParallelMap())
}

func MemberInMap(field field.Model) model.Filter[MemberModel] {
	return func(m MemberModel) bool {
		return m.online && m.WorldId() == field.WorldId() && m.ChannelId() == field.ChannelId() && m.MapId() == field.MapId() && m.Instance() == field.Instance()
	}
}

func OtherMemberInMap(field field.Model, characterId uint32) model.Filter[MemberModel] {
	return func(m MemberModel) bool {
		return m.online && m.WorldId() == field.WorldId() && m.ChannelId() == field.ChannelId() && m.MapId() == field.MapId() && m.Instance() == field.Instance() && m.id != characterId
	}
}
