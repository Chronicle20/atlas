package guild

import (
	"atlas-guilds/character"
	"atlas-guilds/coordinator"
	character2 "atlas-guilds/guild/character"
	"atlas-guilds/guild/member"
	"atlas-guilds/guild/title"
	"atlas-guilds/invite"
	"atlas-guilds/kafka/producer"
	"atlas-guilds/party"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"strings"
)

const (
	ErrorNameInUse     = "THE_NAME_IS_ALREADY_IN_USE_PLEASE_TRY_OTHER_ONES"
	CreateError        = "THE_PROBLEM_HAS_HAPPENED_DURING_THE_PROCESS_OF_FORMING_THE_GUILD_PLEASE_TRY_AGAIN"
	ErrorCannotAsAdmin = "ADMIN_CANNOT_MAKE_A_GUILD"

	MemberThreshold = 2
)

func allProvider(ctx context.Context) func(db *gorm.DB) model.Provider[[]Model] {
	t := tenant.MustFromContext(ctx)
	return func(db *gorm.DB) model.Provider[[]Model] {
		return model.SliceMap(Make)(getAll(t.Id())(db))()
	}
}

func MemberFilter(memberId uint32) model.Filter[Model] {
	return func(m Model) bool {
		for _, mm := range m.members {
			if mm.CharacterId() == memberId {
				return true
			}
		}
		return false
	}
}

func GetSlice(_ logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(filters ...model.Filter[Model]) ([]Model, error) {
	return func(ctx context.Context) func(db *gorm.DB) func(filters ...model.Filter[Model]) ([]Model, error) {
		return func(db *gorm.DB) func(filters ...model.Filter[Model]) ([]Model, error) {
			return func(filters ...model.Filter[Model]) ([]Model, error) {
				return model.FilteredProvider(allProvider(ctx)(db), model.Filters[Model](filters...))()
			}
		}
	}
}

func byIdProvider(_ logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(guildId uint32) model.Provider[Model] {
	return func(ctx context.Context) func(db *gorm.DB) func(guildId uint32) model.Provider[Model] {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(guildId uint32) model.Provider[Model] {
			return func(guildId uint32) model.Provider[Model] {
				return model.Map(Make)(getById(t.Id(), guildId)(db))
			}
		}
	}
}

func GetById(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(guildId uint32) (Model, error) {
	return func(ctx context.Context) func(db *gorm.DB) func(guildId uint32) (Model, error) {
		return func(db *gorm.DB) func(guildId uint32) (Model, error) {
			return func(guildId uint32) (Model, error) {
				return byIdProvider(l)(ctx)(db)(guildId)()
			}
		}
	}
}

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

func GetByMemberId(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(memberId uint32) (Model, error) {
	return func(ctx context.Context) func(db *gorm.DB) func(memberId uint32) (Model, error) {
		return func(db *gorm.DB) func(memberId uint32) (Model, error) {
			return func(memberId uint32) (Model, error) {
				c, err := character2.GetById(l)(ctx)(db)(memberId)
				if err != nil {
					return Model{}, err
				}
				g, err := GetById(l)(ctx)(db)(c.GuildId())
				if err != nil {
					return Model{}, err
				}

				return g, nil
			}
		}
	}
}

func RequestCreate(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, worldId byte, channelId byte, mapId uint32, name string) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, worldId byte, channelId byte, mapId uint32, name string) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32, worldId byte, channelId byte, mapId uint32, name string) error {
			return func(characterId uint32, worldId byte, channelId byte, mapId uint32, name string) error {
				if nameInUse(l)(ctx)(db)(worldId, name) {
					_ = producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)(statusEventErrorProvider(worldId, characterId, ErrorNameInUse))
					return errors.New("name in use")
				}

				if isValidName(name) {
					_ = producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)(statusEventErrorProvider(worldId, characterId, CreateError))
					return errors.New("invalid name")
				}

				c, err := character.GetById(l)(ctx)(characterId)
				if err != nil {
					l.WithError(err).Errorf("Unable to retrieve character [%d] attempting to create guild.", characterId)
					_ = producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)(statusEventErrorProvider(worldId, characterId, CreateError))
					return err
				}

				if c.Gm() {
					l.WithError(err).Errorf("Game administrators cannot create guild.")
					_ = producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)(statusEventErrorProvider(worldId, characterId, ErrorCannotAsAdmin))
					return err
				}

				p, err := party.GetByMemberId(l)(ctx)(characterId)
				if err != nil {
					l.WithError(err).Errorf("Unable to retrieve party for character [%d] attempting to create guild.", characterId)
					_ = producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)(statusEventErrorProvider(worldId, characterId, CreateError))
					return err
				}
				if p.LeaderId() != characterId {
					l.WithError(err).Errorf("Character [%d] must be party leader to create guild.", characterId)
					_ = producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)(statusEventErrorProvider(worldId, characterId, CreateError))
					return errors.New("must be party leader")
				}

				if len(p.Members()) < MemberThreshold {
					l.WithError(err).Errorf("Unable to create guild with less than [%d] party members.", MemberThreshold)
					_ = producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)(statusEventErrorProvider(worldId, characterId, CreateError))
					return errors.New("not enough members")
				}

				var members = make([]uint32, 0)
				var alreadyInGuild = false
				for _, m := range p.Members() {
					// TODO this should be better
					g, _ := GetByMemberId(l)(ctx)(db)(m.Id())
					if g.Id() != 0 {
						alreadyInGuild = true
					}
					members = append(members, m.Id())
				}
				if alreadyInGuild {
					l.WithError(err).Errorf("All party members must not be in a guild.")
					_ = producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)(statusEventErrorProvider(worldId, characterId, CreateError))
					return errors.New("party member in guild")
				}

				err = coordinator.GetRegistry().Initiate(t, worldId, channelId, name, characterId, members)
				if err != nil {
					l.WithError(err).Errorf("Unable to initialize a guild creation coordinator.")
					_ = producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)(statusEventErrorProvider(worldId, characterId, CreateError))
					return errors.New("creation coordinator initialization failed")
				}

				_ = producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)(statusEventRequestAgreementProvider(worldId, characterId, name))
				return nil
			}
		}
	}

}

func nameInUse(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(worldId byte, name string) bool {
	return func(ctx context.Context) func(db *gorm.DB) func(worldId byte, name string) bool {
		return func(db *gorm.DB) func(worldId byte, name string) bool {
			return func(worldId byte, name string) bool {
				g, _ := GetByName(l)(ctx)(db)(worldId, name)
				return g.Id() != 0
			}
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
					g, err = GetByName(l)(ctx)(tx)(worldId, name)
					if g.Id() != 0 {
						l.WithError(err).Errorf("Attempting to create a guild [%s] by name which already exists.", name)
						return errors.New("already exists")
					}

					lge, err := character2.GetById(l)(ctx)(tx)(leaderId)
					if lge.GuildId() != 0 {
						l.WithError(err).Errorf("Character [%d] already in guild. Cannot create one.", leaderId)
					}

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
						l.WithError(err).Errorf("Unable to create default titles for guild [%d].", g.Id())
						return err
					}

					return nil
				})
				return g, txErr
			}
		}
	}
}

func CreationAgreementResponse(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, agreed bool) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, agreed bool) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32, agreed bool) error {
			return func(characterId uint32, agreed bool) error {
				gc, err := coordinator.GetRegistry().Respond(t, characterId, agreed)
				if err != nil {
					l.WithError(err).Errorf("Unable to record character [%d] guild creation agreement [%t].", characterId, agreed)
					return err
				}
				l.Debugf("Character [%d] responded to [%d] request to create guild [%s] with a [%t].", characterId, gc.LeaderId(), gc.Name(), agreed)

				if !agreed {
					l.Debugf("Creation of guild [%s] failed due to [%d] rejecting the invite.", gc.Name(), characterId)
					// TODO respond with failure
					return nil
				}

				if len(gc.Responses()) != len(gc.Requests()) {
					l.Debugf("[%d/%d] responses needed to create guild [%s]. Continuing to wait other responses.", len(gc.Responses()), len(gc.Requests()), gc.Name())
					return nil
				}

				g, err := Create(l)(ctx)(db)(gc.WorldId(), gc.LeaderId(), gc.Name())
				if err != nil {
					l.WithError(err).Errorf("Failed to create guild [%s].", gc.Name())
					return err
				}

				for _, gmid := range gc.Requests() {
					if gmid != gc.LeaderId() {
						c, err := character.GetById(l)(ctx)(gmid)
						if err != nil {
							l.WithError(err).Errorf("Unable to request character information on [%d]. Unable to add them to guild.", gmid)
							continue
						}
						_, err = member.AddMember(l)(ctx)(db)(g.Id(), gmid, c.Name(), c.JobId(), c.Level(), 2)
						if err != nil {
							l.WithError(err).Errorf("Unable to add character [%d] to guild [%d].", gmid, g.Id())
							continue
						}
					}
				}

				l.Debugf("Guild [%d] created.", g.Id())
				_ = producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)(statusEventCreatedProvider(g.WorldId(), g.Id()))
				return nil
			}
		}
	}
}

func ChangeEmblem(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(guildId uint32, characterId uint32, logo uint16, logoColor byte, logoBackground uint16, logoBackgroundColor byte) error {
	return func(ctx context.Context) func(db *gorm.DB) func(guildId uint32, characterId uint32, logo uint16, logoColor byte, logoBackground uint16, logoBackgroundColor byte) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(guildId uint32, characterId uint32, logo uint16, logoColor byte, logoBackground uint16, logoBackgroundColor byte) error {
			return func(guildId uint32, characterId uint32, logo uint16, logoColor byte, logoBackground uint16, logoBackgroundColor byte) error {
				l.Debugf("Character [%d] attempting to update guild [%d] emblem to Logo [%d], Logo Color [%d], Logo Background [%d], Logo Background Color [%d].", characterId, guildId, logo, logoColor, logoBackground, logoBackgroundColor)
				g, err := updateEmblem(db, t.Id(), guildId, logo, logoColor, logoBackground, logoBackgroundColor)
				if err != nil {
					return err
				}
				_ = producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)(statusEventEmblemUpdatedProvider(g.WorldId(), g.Id(), logo, logoColor, logoBackground, logoBackgroundColor))
				return nil
			}
		}
	}
}

func UpdateMemberOnline(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, online bool) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, online bool) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32, online bool) error {
			return func(characterId uint32, online bool) error {
				return db.Transaction(func(tx *gorm.DB) error {
					g, err := GetByMemberId(l)(ctx)(tx)(characterId)
					if err != nil {
						return nil
					}
					l.Debugf("Updating guild [%d] member [%d] status to online [%t]", g.Id(), characterId, online)
					err = updateMemberStatus(db, t.Id(), g.Id(), characterId, online)
					if err != nil {
						return err
					}
					_ = producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)(statusEventMemberStatusUpdatedProvider(g.WorldId(), g.Id(), characterId, online))
					return nil
				})
			}
		}
	}
}

func ChangeNotice(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(guildId uint32, characterId uint32, notice string) error {
	return func(ctx context.Context) func(db *gorm.DB) func(guildId uint32, characterId uint32, notice string) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(guildId uint32, characterId uint32, notice string) error {
			return func(guildId uint32, characterId uint32, notice string) error {
				l.Debugf("Character [%d] is setting guild [%d] notice [%s].", characterId, guildId, notice)
				g, err := updateNotice(db, t.Id(), guildId, notice)
				if err != nil {
					return err
				}
				_ = producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)(statusEventNoticeUpdatedProvider(g.WorldId(), g.Id(), notice))
				return nil
			}
		}
	}
}

func Leave(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(guildId uint32, characterId uint32, force bool) error {
	return func(ctx context.Context) func(db *gorm.DB) func(guildId uint32, characterId uint32, force bool) error {
		return func(db *gorm.DB) func(guildId uint32, characterId uint32, force bool) error {
			return func(guildId uint32, characterId uint32, force bool) error {
				l.Debugf("Character [%d] is leaving guild [%d]. Forced? [%t].", characterId, guildId, force)
				g, err := GetById(l)(ctx)(db)(guildId)
				if err != nil {
					return err
				}

				err = member.RemoveMember(l)(ctx)(db)(guildId, characterId)
				if err != nil {
					return err
				}

				_ = producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)(statusEventMemberLeftProvider(g.WorldId(), g.Id(), characterId, force))
				return nil
			}
		}
	}
}

func RequestInvite(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(guildId uint32, characterId uint32, targetId uint32) error {
	return func(ctx context.Context) func(db *gorm.DB) func(guildId uint32, characterId uint32, targetId uint32) error {
		return func(db *gorm.DB) func(guildId uint32, characterId uint32, targetId uint32) error {
			return func(guildId uint32, characterId uint32, targetId uint32) error {
				l.Debugf("Character [%d] requesting that [%d] be invited to guild [%d].", characterId, targetId, guildId)
				g, err := GetById(l)(ctx)(db)(guildId)
				if err != nil {
					// TODO issue error
					return err
				}
				if uint32(len(g.Members())) >= g.Capacity() {
					// TODO issue error
					return errors.New("guild full")
				}

				return invite.Create(l)(ctx)(characterId, g.WorldId(), g.Id(), targetId)
			}
		}
	}
}

func Join(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(guildId uint32, characterId uint32) error {
	return func(ctx context.Context) func(db *gorm.DB) func(guildId uint32, characterId uint32) error {
		return func(db *gorm.DB) func(guildId uint32, characterId uint32) error {
			return func(guildId uint32, characterId uint32) error {
				c, err := character.GetById(l)(ctx)(characterId)
				if err != nil {
					return err
				}

				g, err := GetById(l)(ctx)(db)(guildId)
				if err != nil {
					return err
				}

				_, err = member.AddMember(l)(ctx)(db)(guildId, characterId, c.Name(), c.JobId(), c.Level(), 5)
				if err != nil {
					return err
				}

				_ = producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)(statusEventMemberJoinedProvider(g.WorldId(), g.Id(), characterId, c.Name(), c.JobId(), c.Level(), 5, 5))
				return nil
			}
		}
	}
}

func ChangeTitles(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(guildId uint32, characterId uint32, titles []string) error {
	return func(ctx context.Context) func(db *gorm.DB) func(guildId uint32, characterId uint32, titles []string) error {
		return func(db *gorm.DB) func(guildId uint32, characterId uint32, titles []string) error {
			return func(guildId uint32, characterId uint32, titles []string) error {
				l.Debugf("Character [%d] changing guild [%d] titles to [%s].", characterId, guildId, strings.Join(titles, ":"))
				g, err := GetById(l)(ctx)(db)(guildId)
				if err != nil {
					return err
				}

				err = title.Replace(l)(ctx)(db)(guildId, titles)
				if err != nil {
					return err
				}

				_ = producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)(statusEventTitlesUpdatedProvider(g.WorldId(), g.Id(), titles))
				return nil
			}
		}
	}
}

func ChangeMemberTitle(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(guildId uint32, characterId uint32, targetId uint32, title byte) error {
	return func(ctx context.Context) func(db *gorm.DB) func(guildId uint32, characterId uint32, targetId uint32, title byte) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(guildId uint32, characterId uint32, targetId uint32, title byte) error {
			return func(guildId uint32, characterId uint32, targetId uint32, title byte) error {
				l.Debugf("Character [%d] attempting to change [%d] title to [%d].", characterId, targetId, title)
				return db.Transaction(func(tx *gorm.DB) error {
					g, err := GetByMemberId(l)(ctx)(tx)(characterId)
					if err != nil {
						return nil
					}
					l.Debugf("Updating guild [%d] member [%d] title to [%d]", g.Id(), targetId, title)
					err = updateMemberTitle(db, t.Id(), g.Id(), targetId, title)
					if err != nil {
						return err
					}
					_ = producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)(statusEventMemberTitleUpdatedProvider(g.WorldId(), g.Id(), targetId, title))
					return nil
				})
			}
		}
	}
}

func RequestDisband(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32) error {
			return func(characterId uint32) error {
				l.Debugf("Character [%d] attempting to disband guild.", characterId)
				return db.Transaction(func(tx *gorm.DB) error {
					g, err := GetByMemberId(l)(ctx)(tx)(characterId)
					if err != nil {
						return err
					}
					if g.LeaderId() != characterId {
						return errors.New("must be leader")
					}

					members := make([]uint32, 0)
					for _, gm := range g.Members() {
						members = append(members, gm.CharacterId())
						_ = member.RemoveMember(l)(ctx)(tx)(g.Id(), gm.CharacterId())
					}
					_ = title.Clear(l)(ctx)(tx)(g.Id())
					_ = deleteGuild(tx, t.Id(), g.Id())

					_ = producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)(statusEventDisbandedProvider(g.WorldId(), g.Id(), members))
					return nil
				})
			}
		}
	}
}
