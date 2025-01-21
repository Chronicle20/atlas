package guild

import (
	"atlas-guilds/character"
	"atlas-guilds/guild/member"
	"atlas-guilds/guild/title"
	"atlas-guilds/kafka/producer"
	"atlas-guilds/party"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const (
	GuildOperationCreateErrorNameInUse     = "THE_NAME_IS_ALREADY_IN_USE_PLEASE_TRY_OTHER_ONES"
	GuildOperationCreateError              = "THE_PROBLEM_HAS_HAPPENED_DURING_THE_PROCESS_OF_FORMING_THE_GUILD_PLEASE_TRY_AGAIN"
	GuildOperationCreateErrorCannotAsAdmin = "ADMIN_CANNOT_MAKE_A_GUILD"

	GuildPartyMemberThreshold = 1
)

func byNameProvider(_ logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(worldId byte, name string) model.Provider[Model] {
	return func(ctx context.Context) func(db *gorm.DB) func(worldId byte, name string) model.Provider[Model] {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(worldId byte, name string) model.Provider[Model] {
			return func(worldId byte, name string) model.Provider[Model] {
				ep := model.SliceMap[Entity, Model](Make)(getForName(t.Id(), worldId, name)(db))(model.ParallelMap())
				return model.FirstProvider(ep, model.Filters[Model]())
			}
		}
	}
}

func GetByName(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(worldId byte, name string) (Model, error) {
	return func(ctx context.Context) func(db *gorm.DB) func(worldId byte, name string) (Model, error) {
		return func(db *gorm.DB) func(worldId byte, name string) (Model, error) {
			return func(worldId byte, name string) (Model, error) {
				return byNameProvider(l)(ctx)(db)(worldId, name)()
			}
		}
	}
}

func MemberFilter(memberId uint32) model.Filter[Model] {
	return func(m Model) bool {
		for _, m := range m.members {
			if m.CharacterId() == memberId {
				return true
			}
		}
		return false
	}
}

func allProvider(ctx context.Context) model.Provider[[]Model] {
	return func() ([]Model, error) {
		return make([]Model, 0), nil
	}
}

func GetSlice(ctx context.Context) func(filters ...model.Filter[Model]) ([]Model, error) {
	return func(filters ...model.Filter[Model]) ([]Model, error) {
		return model.FilteredProvider(allProvider(ctx), model.Filters[Model](filters...))()
	}
}

func GetByMemberId(l logrus.FieldLogger) func(ctx context.Context) func(memberId uint32) (Model, error) {
	return func(ctx context.Context) func(memberId uint32) (Model, error) {
		return func(memberId uint32) (Model, error) {
			gs, err := GetSlice(ctx)(model.Filters[Model](MemberFilter(memberId))...)
			if err != nil {
				return Model{}, err
			}
			return model.First(model.FixedProvider(gs), model.Filters[Model]())
		}
	}
}

func RequestCreate(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, worldId byte, channelId byte, mapId uint32, name string) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, worldId byte, channelId byte, mapId uint32, name string) error {
		return func(db *gorm.DB) func(characterId uint32, worldId byte, channelId byte, mapId uint32, name string) error {
			return func(characterId uint32, worldId byte, channelId byte, mapId uint32, name string) error {
				if nameInUse(l)(ctx)(worldId, name) {
					_ = producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)(statusEventErrorProvider(worldId, characterId, GuildOperationCreateErrorNameInUse))
					return errors.New("name in use")
				}

				if isValidName(name) {
					_ = producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)(statusEventErrorProvider(worldId, characterId, GuildOperationCreateError))
					return errors.New("invalid name")
				}

				c, err := character.GetById(l)(ctx)(characterId)
				if err != nil {
					l.WithError(err).Errorf("Unable to retrieve character [%d] attempting to create guild.", characterId)
					_ = producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)(statusEventErrorProvider(worldId, characterId, GuildOperationCreateError))
					return err
				}

				if c.Gm() {
					l.WithError(err).Errorf("Game administrators cannot create guild.")
					_ = producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)(statusEventErrorProvider(worldId, characterId, GuildOperationCreateErrorCannotAsAdmin))
					return err
				}

				p, err := party.GetByMemberId(l)(ctx)(characterId)
				if err != nil {
					l.WithError(err).Errorf("Unable to retrieve party for character [%d] attempting to create guild.", characterId)
					_ = producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)(statusEventErrorProvider(worldId, characterId, GuildOperationCreateError))
					return err
				}
				if p.LeaderId() != characterId {
					l.WithError(err).Errorf("Character [%d] must be party leader to create guild.", characterId)
					_ = producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)(statusEventErrorProvider(worldId, characterId, GuildOperationCreateError))
					return errors.New("must be party leader")
				}

				if len(p.Members()) < GuildPartyMemberThreshold {
					l.WithError(err).Errorf("Unable to create guild with less than [%d] party members.", GuildPartyMemberThreshold)
					_ = producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)(statusEventErrorProvider(worldId, characterId, GuildOperationCreateError))
					return errors.New("not enough members")
				}

				var alreadyInGuild = false
				for _, m := range p.Members() {
					// TODO this should be better
					g, _ := GetByMemberId(l)(ctx)(m.Id())
					if g.Id() != 0 {
						alreadyInGuild = true
						break
					}
				}
				if alreadyInGuild {
					l.WithError(err).Errorf("All party members must not be in a guild.")
					_ = producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)(statusEventErrorProvider(worldId, characterId, GuildOperationCreateError))
					return errors.New("party member in guild")
				}

				_ = producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)(statusEventRequestAgreementProvider(worldId, characterId, name))
				return nil
			}
		}
	}

}

func nameInUse(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, name string) bool {
	return func(ctx context.Context) func(worldId byte, name string) bool {
		return func(worldId byte, name string) bool {
			// TODO identify if name in use
			return name == "Already"
		}
	}
}

func isValidName(name string) bool {
	// TODO validate name
	return name == "Stupid"
}

func Create(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(worldId byte, leaderId uint32, name string) (Model, error) {
	return func(ctx context.Context) func(db *gorm.DB) func(worldId byte, leaderId uint32, name string) (Model, error) {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(worldId byte, leaderId uint32, name string) (Model, error) {
			return func(worldId byte, leaderId uint32, name string) (Model, error) {
				var err error
				var lc character.Model
				lc, err = character.GetById(l)(ctx)(leaderId)
				if err != nil {
					l.WithError(err).Errorf("Unable to locate character [%d] creating guild.", leaderId)
					return Model{}, err
				}

				var g Model
				txErr := db.Transaction(func(tx *gorm.DB) error {
					_, err = GetByName(l)(ctx)(tx)(worldId, name)
					if !errors.Is(err, gorm.ErrRecordNotFound) {
						l.WithError(err).Errorf("Attempting to create a guild [%s] by name which already exists.", name)
						return errors.New("already exists")
					}

					// TODO ensure leader is not already in a guild.

					g, err = create(tx, t, worldId, leaderId, name)
					if err != nil {
						l.WithError(err).Errorf("Unable to create guild [%s].", name)
						return err
					}

					_, err = member.AddMember(l)(ctx)(tx)(g.Id(), leaderId, lc.Name(), lc.JobId(), lc.Level(), 1)
					if err != nil {
						l.WithError(err).Errorf("Unable to add leader [%d] as guild [%d] member.", leaderId, g.Id())
						return err
					}

					_, err = title.CreateDefaults(l)(ctx)(tx)(g.Id())
					if err != nil {
						l.WithError(err).Errorf("Unable to create default ranks for guild [%d].", g.Id())
						return err
					}

					return nil
				})
				return g, txErr
			}
		}
	}
}
