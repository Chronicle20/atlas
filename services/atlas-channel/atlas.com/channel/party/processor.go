package party

import (
	party2 "atlas-channel/kafka/message/party"
	"atlas-channel/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	p := &Processor{
		l:   l,
		ctx: ctx,
	}
	return p
}

func (p *Processor) Create(characterId uint32) error {
	p.l.Debugf("Character [%d] attempting to create a party.", characterId)
	return producer.ProviderImpl(p.l)(p.ctx)(party2.EnvCommandTopic)(CreateCommandProvider(characterId))
}

func (p *Processor) Leave(partyId uint32, characterId uint32) error {
	p.l.Debugf("Character [%d] attempting to leave party [%d].", characterId, partyId)
	return producer.ProviderImpl(p.l)(p.ctx)(party2.EnvCommandTopic)(LeaveCommandProvider(characterId, partyId, false))
}

func (p *Processor) Expel(partyId uint32, characterId uint32, targetCharacterId uint32) error {
	p.l.Debugf("Character [%d] attempting to expel [%d] from party [%d].", characterId, targetCharacterId, partyId)
	return producer.ProviderImpl(p.l)(p.ctx)(party2.EnvCommandTopic)(LeaveCommandProvider(characterId, partyId, true))
}

func (p *Processor) ChangeLeader(partyId uint32, characterId uint32, targetCharacterId uint32) error {
	p.l.Debugf("Character [%d] attempting to pass leadership to [%d] in party [%d].", characterId, targetCharacterId, partyId)
	return producer.ProviderImpl(p.l)(p.ctx)(party2.EnvCommandTopic)(ChangeLeaderCommandProvider(characterId, partyId, targetCharacterId))
}

func (p *Processor) RequestInvite(characterId uint32, targetCharacterId uint32) error {
	p.l.Debugf("Character [%d] attempting to invite [%d] to a party.", characterId, targetCharacterId)
	return producer.ProviderImpl(p.l)(p.ctx)(party2.EnvCommandTopic)(RequestInviteCommandProvider(characterId, targetCharacterId))
}

func (p *Processor) GetById(partyId uint32) (Model, error) {
	return p.ByIdProvider(partyId)()
}

func (p *Processor) ByIdProvider(partyId uint32) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(partyId), Extract)
}

func (p *Processor) GetByMemberId(memberId uint32) (Model, error) {
	return p.ByMemberIdProvider(memberId)()
}

func (p *Processor) ByMemberIdProvider(memberId uint32) model.Provider[Model] {
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

func MemberInMap(worldId world.Id, channelId channel.Id, mapId _map.Id) model.Filter[MemberModel] {
	return func(m MemberModel) bool {
		return m.online && m.worldId == worldId && m.channelId == channelId && m.mapId == mapId
	}
}

func OtherMemberInMap(worldId world.Id, channelId channel.Id, mapId _map.Id, characterId uint32) model.Filter[MemberModel] {
	return func(m MemberModel) bool {
		return m.online && m.worldId == worldId && m.channelId == channelId && m.mapId == mapId && m.id != characterId
	}
}
