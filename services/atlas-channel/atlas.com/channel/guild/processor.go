package guild

import (
	"atlas-channel/guild/member"
	"atlas-channel/kafka/producer"
	"context"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
	"strings"
)

func GetById(l logrus.FieldLogger) func(ctx context.Context) func(guildId uint32) (Model, error) {
	return func(ctx context.Context) func(guildId uint32) (Model, error) {
		return func(guildId uint32) (Model, error) {
			return requests.Provider[RestModel, Model](l, ctx)(requestById(guildId), Extract)()
		}
	}
}

func GetByMemberId(l logrus.FieldLogger) func(ctx context.Context) func(memberId uint32) (Model, error) {
	return func(ctx context.Context) func(memberId uint32) (Model, error) {
		return func(memberId uint32) (Model, error) {
			return model.First[Model](byMemberIdProvider(l)(ctx)(memberId), model.Filters[Model]())
		}
	}
}

func byMemberIdProvider(l logrus.FieldLogger) func(ctx context.Context) func(memberId uint32) model.Provider[[]Model] {
	return func(ctx context.Context) func(memberId uint32) model.Provider[[]Model] {
		return func(memberId uint32) model.Provider[[]Model] {
			return requests.SliceProvider[RestModel, Model](l, ctx)(requestByMemberId(memberId), Extract, model.Filters[Model]())
		}
	}
}

func MemberOnline(m member.Model) bool {
	return m.Online()
}

func NotMember(characterId uint32) model.Filter[member.Model] {
	return func(m member.Model) bool {
		return m.CharacterId() != characterId
	}
}

func GetMemberIds(l logrus.FieldLogger) func(ctx context.Context) func(guildId uint32, filters []model.Filter[member.Model]) model.Provider[[]uint32] {
	return func(ctx context.Context) func(guildId uint32, filters []model.Filter[member.Model]) model.Provider[[]uint32] {
		return func(guildId uint32, filters []model.Filter[member.Model]) model.Provider[[]uint32] {
			g, err := GetById(l)(ctx)(guildId)
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve guild [%d].", guildId)
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
	}
}

func RequestCreate(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, characterId uint32, name string) error {
	return func(ctx context.Context) func(m _map.Model, characterId uint32, name string) error {
		return func(m _map.Model, characterId uint32, name string) error {
			l.Debugf("Character [%d] attempting to create guild [%s] in world [%d] channel [%d] map [%d].", characterId, name, m.WorldId(), m.ChannelId(), m.MapId())
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(requestCreateProvider(m, characterId, name))
		}
	}
}

func CreationAgreement(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, agreed bool) error {
	return func(ctx context.Context) func(characterId uint32, agreed bool) error {
		return func(characterId uint32, agreed bool) error {
			l.Debugf("Character [%d] responded to guild creation agreement with [%t].", characterId, agreed)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(creationAgreementProvider(characterId, agreed))
		}
	}
}

func RequestEmblemUpdate(l logrus.FieldLogger) func(ctx context.Context) func(guildId uint32, characterId uint32, logoBackground uint16, logoBackgroundColor byte, logo uint16, logoColor byte) error {
	return func(ctx context.Context) func(guildId uint32, characterId uint32, logoBackground uint16, logoBackgroundColor byte, logo uint16, logoColor byte) error {
		return func(guildId uint32, characterId uint32, logoBackground uint16, logoBackgroundColor byte, logo uint16, logoColor byte) error {
			l.Debugf("Character [%d] is attempting to change their guild emblem. Logo [%d], Logo Color [%d], Logo Background [%d], Logo Background Color [%d]", characterId, logo, logoColor, logoBackground, logoBackgroundColor)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(changeEmblemProvider(guildId, characterId, logo, logoColor, logoBackground, logoBackgroundColor))
		}
	}
}

func RequestNoticeUpdate(l logrus.FieldLogger) func(ctx context.Context) func(guildId uint32, characterId uint32, notice string) error {
	return func(ctx context.Context) func(guildId uint32, characterId uint32, notice string) error {
		return func(guildId uint32, characterId uint32, notice string) error {
			l.Debugf("Character [%d] is attempting to set guild [%d] notice [%s].", characterId, guildId, notice)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(changeNoticeProvider(guildId, characterId, notice))
		}
	}
}

func Leave(l logrus.FieldLogger) func(ctx context.Context) func(guildId uint32, characterId uint32) error {
	return func(ctx context.Context) func(guildId uint32, characterId uint32) error {
		return func(guildId uint32, characterId uint32) error {
			l.Debugf("Character [%d] is leaving guild [%d].", characterId, guildId)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(leaveGuildProvider(guildId, characterId))
		}
	}
}

func Expel(l logrus.FieldLogger) func(ctx context.Context) func(guildId uint32, characterId uint32, targetId uint32, targetName string) error {
	return func(ctx context.Context) func(guildId uint32, characterId uint32, targetId uint32, targetName string) error {
		return func(guildId uint32, characterId uint32, targetId uint32, targetName string) error {
			l.Debugf("Character [%d] expelling [%d] - [%s] from guild [%d].", characterId, targetId, targetName, guildId)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(expelGuildProvider(guildId, targetId))
		}
	}
}

func RequestInvite(l logrus.FieldLogger) func(ctx context.Context) func(guildId uint32, characterId uint32, targetId uint32) error {
	return func(ctx context.Context) func(guildId uint32, characterId uint32, targetId uint32) error {
		return func(guildId uint32, characterId uint32, targetId uint32) error {
			l.Debugf("Character [%d] is inviting [%d] to guild [%d].", characterId, targetId, guildId)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(requestInviteProvider(guildId, characterId, targetId))
		}
	}
}

func RequestTitleChanges(l logrus.FieldLogger) func(ctx context.Context) func(guildId uint32, characterId uint32, titles []string) error {
	return func(ctx context.Context) func(guildId uint32, characterId uint32, titles []string) error {
		return func(guildId uint32, characterId uint32, titles []string) error {
			l.Debugf("Character [%d] attempting to change guild [%d] titles to [%s].", characterId, guildId, strings.Join(titles, ":"))
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(changeTitlesProvider(guildId, characterId, titles))
		}
	}
}

func RequestMemberTitleUpdate(l logrus.FieldLogger) func(ctx context.Context) func(guildId uint32, characterId uint32, targetId uint32, newTitle byte) error {
	return func(ctx context.Context) func(guildId uint32, characterId uint32, targetId uint32, newTitle byte) error {
		return func(guildId uint32, characterId uint32, targetId uint32, newTitle byte) error {
			l.Debugf("Character [%d] attempting to change [%d] title to [%d].", characterId, targetId, newTitle)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(changeMemberTitleProvider(guildId, characterId, targetId, newTitle))
		}
	}
}
