package guild

import (
	"atlas-channel/guild/member"
	guild2 "atlas-channel/kafka/message/guild"
	"atlas-channel/kafka/producer"
	"context"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
	"strings"
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

func (p *Processor) GetById(guildId uint32) (Model, error) {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(guildId), Extract)()
}

func (p *Processor) GetByMemberId(memberId uint32) (Model, error) {
	return model.First[Model](p.ByMemberIdProvider(memberId), model.Filters[Model]())
}

func (p *Processor) ByMemberIdProvider(memberId uint32) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByMemberId(memberId), Extract, model.Filters[Model]())
}

func MemberOnline(m member.Model) bool {
	return m.Online()
}

func NotMember(characterId uint32) model.Filter[member.Model] {
	return func(m member.Model) bool {
		return m.CharacterId() != characterId
	}
}

func (p *Processor) GetMemberIds(guildId uint32, filters []model.Filter[member.Model]) model.Provider[[]uint32] {
	g, err := p.GetById(guildId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to retrieve guild [%d].", guildId)
		return model.ErrorProvider[[]uint32](err)
	}
	ids := make([]uint32, 0)
	for _, m := range g.Members() {
		ok := true
		for _, f := range filters {
			if !f(m) {
				ok = false
			}
		}
		if ok {
			ids = append(ids, m.CharacterId())
		}
	}
	return model.FixedProvider(ids)
}

func (p *Processor) RequestCreate(m _map.Model, characterId uint32, name string) error {
	p.l.Debugf("Character [%d] attempting to create guild [%s] in world [%d] channel [%d] map [%d].", characterId, name, m.WorldId(), m.ChannelId(), m.MapId())
	return producer.ProviderImpl(p.l)(p.ctx)(guild2.EnvCommandTopic)(RequestCreateProvider(m, characterId, name))
}

func (p *Processor) CreationAgreement(characterId uint32, agreed bool) error {
	p.l.Debugf("Character [%d] responded to guild creation agreement with [%t].", characterId, agreed)
	return producer.ProviderImpl(p.l)(p.ctx)(guild2.EnvCommandTopic)(CreationAgreementProvider(characterId, agreed))
}

func (p *Processor) RequestEmblemUpdate(guildId uint32, characterId uint32, logoBackground uint16, logoBackgroundColor byte, logo uint16, logoColor byte) error {
	p.l.Debugf("Character [%d] is attempting to change their guild emblem. Logo [%d], Logo Color [%d], Logo Background [%d], Logo Background Color [%d]", characterId, logo, logoColor, logoBackground, logoBackgroundColor)
	return producer.ProviderImpl(p.l)(p.ctx)(guild2.EnvCommandTopic)(ChangeEmblemProvider(guildId, characterId, logo, logoColor, logoBackground, logoBackgroundColor))
}

func (p *Processor) RequestNoticeUpdate(guildId uint32, characterId uint32, notice string) error {
	p.l.Debugf("Character [%d] is attempting to set guild [%d] notice [%s].", characterId, guildId, notice)
	return producer.ProviderImpl(p.l)(p.ctx)(guild2.EnvCommandTopic)(ChangeNoticeProvider(guildId, characterId, notice))
}

func (p *Processor) Leave(guildId uint32, characterId uint32) error {
	p.l.Debugf("Character [%d] is leaving guild [%d].", characterId, guildId)
	return producer.ProviderImpl(p.l)(p.ctx)(guild2.EnvCommandTopic)(LeaveGuildProvider(guildId, characterId))
}

func (p *Processor) Expel(guildId uint32, characterId uint32, targetId uint32, targetName string) error {
	p.l.Debugf("Character [%d] expelling [%d] - [%s] from guild [%d].", characterId, targetId, targetName, guildId)
	return producer.ProviderImpl(p.l)(p.ctx)(guild2.EnvCommandTopic)(ExpelGuildProvider(guildId, targetId))
}

func (p *Processor) RequestInvite(guildId uint32, characterId uint32, targetId uint32) error {
	p.l.Debugf("Character [%d] is inviting [%d] to guild [%d].", characterId, targetId, guildId)
	return producer.ProviderImpl(p.l)(p.ctx)(guild2.EnvCommandTopic)(RequestInviteProvider(guildId, characterId, targetId))
}

func (p *Processor) RequestTitleChanges(guildId uint32, characterId uint32, titles []string) error {
	p.l.Debugf("Character [%d] attempting to change guild [%d] titles to [%s].", characterId, guildId, strings.Join(titles, ":"))
	return producer.ProviderImpl(p.l)(p.ctx)(guild2.EnvCommandTopic)(ChangeTitlesProvider(guildId, characterId, titles))
}

func (p *Processor) RequestMemberTitleUpdate(guildId uint32, characterId uint32, targetId uint32, newTitle byte) error {
	p.l.Debugf("Character [%d] attempting to change [%d] title to [%d].", characterId, targetId, newTitle)
	return producer.ProviderImpl(p.l)(p.ctx)(guild2.EnvCommandTopic)(ChangeMemberTitleProvider(guildId, characterId, targetId, newTitle))
}
