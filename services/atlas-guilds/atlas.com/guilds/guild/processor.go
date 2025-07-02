package guild

import (
	"atlas-guilds/character"
	"atlas-guilds/coordinator"
	"atlas-guilds/database"
	character2 "atlas-guilds/guild/character"
	"atlas-guilds/guild/member"
	"atlas-guilds/guild/title"
	"atlas-guilds/invite"
	guild2 "atlas-guilds/kafka/message/guild"
	"atlas-guilds/kafka/producer"
	"atlas-guilds/party"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
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

type Processor interface {
	WithTransaction(tx *gorm.DB) Processor
	AllProvider() model.Provider[[]Model]
	ByIdProvider(guildId uint32) model.Provider[Model]
	ByNameProvider(worldId byte, name string) model.Provider[Model]
	GetSlice(filters ...model.Filter[Model]) ([]Model, error)
	GetById(guildId uint32) (Model, error)
	GetByName(worldId byte, name string) (Model, error)
	GetByMemberId(memberId uint32) (Model, error)
	RequestCreate(characterId uint32, worldId byte, channelId byte, mapId uint32, name string, transactionId uuid.UUID) error
	Create(worldId byte, leaderId uint32, name string) (Model, error)
	CreationAgreementResponse(characterId uint32, agreed bool, transactionId uuid.UUID) error
	ChangeEmblem(guildId uint32, characterId uint32, logo uint16, logoColor byte, logoBackground uint16, logoBackgroundColor byte, transactionId uuid.UUID) error
	UpdateMemberOnline(characterId uint32, online bool, transactionId uuid.UUID) error
	ChangeNotice(guildId uint32, characterId uint32, notice string, transactionId uuid.UUID) error
	Leave(guildId uint32, characterId uint32, force bool, transactionId uuid.UUID) error
	RequestInvite(guildId uint32, characterId uint32, targetId uint32) error
	Join(guildId uint32, characterId uint32, transactionId uuid.UUID) error
	ChangeTitles(guildId uint32, characterId uint32, titles []string, transactionId uuid.UUID) error
	ChangeMemberTitle(guildId uint32, characterId uint32, targetId uint32, title byte, transactionId uuid.UUID) error
	RequestDisband(characterId uint32, transactionId uuid.UUID) error
	RequestCapacityIncrease(characterId uint32, transactionId uuid.UUID) error
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

func (p *ProcessorImpl) WithTransaction(tx *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   p.l,
		ctx: p.ctx,
		db:  tx,
		t:   p.t,
	}
}

func (p *ProcessorImpl) AllProvider() model.Provider[[]Model] {
	return model.SliceMap(Make)(getAll(p.t.Id())(p.db))()
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

func (p *ProcessorImpl) GetSlice(filters ...model.Filter[Model]) ([]Model, error) {
	return model.FilteredProvider(p.AllProvider(), model.Filters[Model](filters...))()
}

func (p *ProcessorImpl) ByIdProvider(guildId uint32) model.Provider[Model] {
	return model.Map(Make)(getById(p.t.Id(), guildId)(p.db))
}

func (p *ProcessorImpl) GetById(guildId uint32) (Model, error) {
	return p.ByIdProvider(guildId)()
}

func (p *ProcessorImpl) ByNameProvider(worldId byte, name string) model.Provider[Model] {
	ep := model.SliceMap[Entity, Model](Make)(getForName(p.t.Id(), worldId, name)(p.db))(model.ParallelMap())
	return model.FirstProvider(ep, model.Filters[Model]())
}

func (p *ProcessorImpl) GetByName(worldId byte, name string) (Model, error) {
	return p.ByNameProvider(worldId, name)()
}

func (p *ProcessorImpl) GetByMemberId(memberId uint32) (Model, error) {
	c, err := character2.NewProcessor(p.l, p.ctx, p.db).GetById(memberId)
	if err != nil {
		return Model{}, err
	}
	g, err := p.GetById(c.GuildId())
	if err != nil {
		return Model{}, err
	}

	return g, nil
}

func (p *ProcessorImpl) RequestCreate(characterId uint32, worldId byte, channelId byte, mapId uint32, name string, transactionId uuid.UUID) error {
	if p.nameInUse(worldId, name) {
		_ = producer.ProviderImpl(p.l)(p.ctx)(guild2.EnvStatusEventTopic)(statusEventErrorProvider(worldId, characterId, ErrorNameInUse, transactionId))
		return errors.New("name in use")
	}

	if isValidName(name) {
		_ = producer.ProviderImpl(p.l)(p.ctx)(guild2.EnvStatusEventTopic)(statusEventErrorProvider(worldId, characterId, CreateError, transactionId))
		return errors.New("invalid name")
	}

	c, err := character.NewProcessor(p.l, p.ctx).GetById(characterId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to retrieve character [%d] attempting to create guild.", characterId)
		_ = producer.ProviderImpl(p.l)(p.ctx)(guild2.EnvStatusEventTopic)(statusEventErrorProvider(worldId, characterId, CreateError, transactionId))
		return err
	}

	if c.Gm() {
		p.l.WithError(err).Errorf("Game administrators cannot create guild.")
		_ = producer.ProviderImpl(p.l)(p.ctx)(guild2.EnvStatusEventTopic)(statusEventErrorProvider(worldId, characterId, ErrorCannotAsAdmin, transactionId))
		return err
	}

	pa, err := party.NewProcessor(p.l, p.ctx).GetByMemberId(characterId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to retrieve party for character [%d] attempting to create guild.", characterId)
		_ = producer.ProviderImpl(p.l)(p.ctx)(guild2.EnvStatusEventTopic)(statusEventErrorProvider(worldId, characterId, CreateError, transactionId))
		return err
	}
	if pa.LeaderId() != characterId {
		p.l.WithError(err).Errorf("Character [%d] must be party leader to create guild.", characterId)
		_ = producer.ProviderImpl(p.l)(p.ctx)(guild2.EnvStatusEventTopic)(statusEventErrorProvider(worldId, characterId, CreateError, transactionId))
		return errors.New("must be party leader")
	}

	if len(pa.Members()) < MemberThreshold {
		p.l.WithError(err).Errorf("Unable to create guild with less than [%d] party members.", MemberThreshold)
		_ = producer.ProviderImpl(p.l)(p.ctx)(guild2.EnvStatusEventTopic)(statusEventErrorProvider(worldId, characterId, CreateError, transactionId))
		return errors.New("not enough members")
	}

	var members = make([]uint32, 0)
	var alreadyInGuild = false
	for _, m := range pa.Members() {
		// TODO this should be better
		g, _ := p.GetByMemberId(m.Id())
		if g.Id() != 0 {
			alreadyInGuild = true
		}
		members = append(members, m.Id())
	}
	if alreadyInGuild {
		p.l.WithError(err).Errorf("All party members must not be in a guild.")
		_ = producer.ProviderImpl(p.l)(p.ctx)(guild2.EnvStatusEventTopic)(statusEventErrorProvider(worldId, characterId, CreateError, transactionId))
		return errors.New("party member in guild")
	}

	err = coordinator.GetRegistry().Initiate(p.t, worldId, channelId, name, characterId, members)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to initialize a guild creation coordinator.")
		_ = producer.ProviderImpl(p.l)(p.ctx)(guild2.EnvStatusEventTopic)(statusEventErrorProvider(worldId, characterId, CreateError, transactionId))
		return errors.New("creation coordinator initialization failed")
	}

	_ = producer.ProviderImpl(p.l)(p.ctx)(guild2.EnvStatusEventTopic)(statusEventRequestAgreementProvider(worldId, characterId, name, transactionId))
	return nil
}

func (p *ProcessorImpl) nameInUse(worldId byte, name string) bool {
	g, _ := p.GetByName(worldId, name)
	return g.Id() != 0
}

func isValidName(name string) bool {
	// TODO validate name
	return name == "Stupid"
}

func (p *ProcessorImpl) Create(worldId byte, leaderId uint32, name string) (Model, error) {
	var err error
	var lc character.Model
	lc, err = character.NewProcessor(p.l, p.ctx).GetById(leaderId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to locate character [%d] creating guild.", leaderId)
		return Model{}, err
	}

	var g Model
	txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
		g, err = p.WithTransaction(tx).GetByName(worldId, name)
		if g.Id() != 0 {
			p.l.WithError(err).Errorf("Attempting to create a guild [%s] by name which already exists.", name)
			return errors.New("already exists")
		}

		lge, err := character2.NewProcessor(p.l, p.ctx, tx).GetById(leaderId)
		if lge.GuildId() != 0 {
			p.l.WithError(err).Errorf("Character [%d] already in guild. Cannot create one.", leaderId)
		}

		g, err = create(tx, p.t, worldId, leaderId, name)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to create guild [%s].", name)
			return err
		}

		_, err = member.NewProcessor(p.l, p.ctx, tx).AddMember(g.Id(), leaderId, lc.Name(), lc.JobId(), lc.Level(), 1)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to add leader [%d] as guild [%d] member.", leaderId, g.Id())
			return err
		}

		_, err = title.NewProcessor(p.l, p.ctx, tx).CreateDefaults(g.Id())
		if err != nil {
			p.l.WithError(err).Errorf("Unable to create default titles for guild [%d].", g.Id())
			return err
		}

		return nil
	})
	return g, txErr
}

func (p *ProcessorImpl) CreationAgreementResponse(characterId uint32, agreed bool, transactionId uuid.UUID) error {
	gc, err := coordinator.GetRegistry().Respond(p.t, characterId, agreed)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to record character [%d] guild creation agreement [%t].", characterId, agreed)
		return err
	}
	p.l.Debugf("Character [%d] responded to [%d] request to create guild [%s] with a [%t].", characterId, gc.LeaderId(), gc.Name(), agreed)

	if !agreed {
		p.l.Debugf("Creation of guild [%s] failed due to [%d] rejecting the invite.", gc.Name(), characterId)
		// TODO respond with failure
		return nil
	}

	if len(gc.Responses()) != len(gc.Requests()) {
		p.l.Debugf("[%d/%d] responses needed to create guild [%s]. Continuing to wait other responses.", len(gc.Responses()), len(gc.Requests()), gc.Name())
		return nil
	}

	g, err := p.Create(gc.WorldId(), gc.LeaderId(), gc.Name())
	if err != nil {
		p.l.WithError(err).Errorf("Failed to create guild [%s].", gc.Name())
		return err
	}

	for _, gmid := range gc.Requests() {
		if gmid != gc.LeaderId() {
			c, err := character.NewProcessor(p.l, p.ctx).GetById(gmid)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to request character information on [%d]. Unable to add them to guild.", gmid)
				continue
			}
			_, err = member.NewProcessor(p.l, p.ctx, p.db).AddMember(g.Id(), gmid, c.Name(), c.JobId(), c.Level(), 2)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to add character [%d] to guild [%d].", gmid, g.Id())
				continue
			}
		}
	}

	p.l.Debugf("Guild [%d] created.", g.Id())
	_ = producer.ProviderImpl(p.l)(p.ctx)(guild2.EnvStatusEventTopic)(statusEventCreatedProvider(g.WorldId(), g.Id(), transactionId))
	return nil
}

func (p *ProcessorImpl) ChangeEmblem(guildId uint32, characterId uint32, logo uint16, logoColor byte, logoBackground uint16, logoBackgroundColor byte, transactionId uuid.UUID) error {
	p.l.Debugf("Character [%d] attempting to update guild [%d] emblem to Logo [%d], Logo Color [%d], Logo Background [%d], Logo Background Color [%d].", characterId, guildId, logo, logoColor, logoBackground, logoBackgroundColor)
	g, err := updateEmblem(p.db, p.t.Id(), guildId, logo, logoColor, logoBackground, logoBackgroundColor)
	if err != nil {
		return err
	}
	_ = producer.ProviderImpl(p.l)(p.ctx)(guild2.EnvStatusEventTopic)(statusEventEmblemUpdatedProvider(g.WorldId(), g.Id(), logo, logoColor, logoBackground, logoBackgroundColor, transactionId))
	return nil
}

func (p *ProcessorImpl) UpdateMemberOnline(characterId uint32, online bool, transactionId uuid.UUID) error {
	return database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
		g, err := p.WithTransaction(tx).GetByMemberId(characterId)
		if err != nil {
			return nil
		}
		p.l.Debugf("Updating guild [%d] member [%d] status to online [%t]", g.Id(), characterId, online)
		err = updateMemberStatus(tx, p.t.Id(), g.Id(), characterId, online)
		if err != nil {
			return err
		}
		_ = producer.ProviderImpl(p.l)(p.ctx)(guild2.EnvStatusEventTopic)(statusEventMemberStatusUpdatedProvider(g.WorldId(), g.Id(), characterId, online, transactionId))
		return nil
	})
}

func (p *ProcessorImpl) ChangeNotice(guildId uint32, characterId uint32, notice string, transactionId uuid.UUID) error {
	p.l.Debugf("Character [%d] is setting guild [%d] notice [%s].", characterId, guildId, notice)
	g, err := updateNotice(p.db, p.t.Id(), guildId, notice)
	if err != nil {
		return err
	}
	_ = producer.ProviderImpl(p.l)(p.ctx)(guild2.EnvStatusEventTopic)(statusEventNoticeUpdatedProvider(g.WorldId(), g.Id(), notice, transactionId))
	return nil
}

func (p *ProcessorImpl) Leave(guildId uint32, characterId uint32, force bool, transactionId uuid.UUID) error {
	p.l.Debugf("Character [%d] is leaving guild [%d]. Forced? [%t].", characterId, guildId, force)
	g, err := p.GetById(guildId)
	if err != nil {
		return err
	}

	err = member.NewProcessor(p.l, p.ctx, p.db).RemoveMember(guildId, characterId)
	if err != nil {
		return err
	}

	_ = producer.ProviderImpl(p.l)(p.ctx)(guild2.EnvStatusEventTopic)(statusEventMemberLeftProvider(g.WorldId(), g.Id(), characterId, force, transactionId))
	return nil
}

func (p *ProcessorImpl) RequestInvite(guildId uint32, characterId uint32, targetId uint32) error {
	p.l.Debugf("Character [%d] requesting that [%d] be invited to guild [%d].", characterId, targetId, guildId)
	g, err := p.GetById(guildId)
	if err != nil {
		// TODO issue error
		return err
	}
	if uint32(len(g.Members())) >= g.Capacity() {
		// TODO issue error
		return errors.New("guild full")
	}

	return invite.NewProcessor(p.l, p.ctx).Create(characterId, g.WorldId(), g.Id(), targetId)
}

func (p *ProcessorImpl) Join(guildId uint32, characterId uint32, transactionId uuid.UUID) error {
	c, err := character.NewProcessor(p.l, p.ctx).GetById(characterId)
	if err != nil {
		return err
	}

	g, err := p.GetById(guildId)
	if err != nil {
		return err
	}

	_, err = member.NewProcessor(p.l, p.ctx, p.db).AddMember(guildId, characterId, c.Name(), c.JobId(), c.Level(), 5)
	if err != nil {
		return err
	}

	_ = producer.ProviderImpl(p.l)(p.ctx)(guild2.EnvStatusEventTopic)(statusEventMemberJoinedProvider(g.WorldId(), g.Id(), characterId, c.Name(), c.JobId(), c.Level(), 5, 5, transactionId))
	return nil
}

func (p *ProcessorImpl) ChangeTitles(guildId uint32, characterId uint32, titles []string, transactionId uuid.UUID) error {
	p.l.Debugf("Character [%d] changing guild [%d] titles to [%s].", characterId, guildId, strings.Join(titles, ":"))
	g, err := p.GetById(guildId)
	if err != nil {
		return err
	}

	err = title.NewProcessor(p.l, p.ctx, p.db).Replace(guildId, titles)
	if err != nil {
		return err
	}

	_ = producer.ProviderImpl(p.l)(p.ctx)(guild2.EnvStatusEventTopic)(statusEventTitlesUpdatedProvider(g.WorldId(), g.Id(), titles, transactionId))
	return nil
}

func (p *ProcessorImpl) ChangeMemberTitle(guildId uint32, characterId uint32, targetId uint32, title byte, transactionId uuid.UUID) error {
	p.l.Debugf("Character [%d] attempting to change [%d] title to [%d].", characterId, targetId, title)
	return database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
		g, err := p.WithTransaction(tx).GetByMemberId(characterId)
		if err != nil {
			return nil
		}
		p.l.Debugf("Updating guild [%d] member [%d] title to [%d]", g.Id(), targetId, title)
		err = updateMemberTitle(tx, p.t.Id(), g.Id(), targetId, title)
		if err != nil {
			return err
		}
		_ = producer.ProviderImpl(p.l)(p.ctx)(guild2.EnvStatusEventTopic)(statusEventMemberTitleUpdatedProvider(g.WorldId(), g.Id(), targetId, title, transactionId))
		return nil
	})
}

func (p *ProcessorImpl) RequestDisband(characterId uint32, transactionId uuid.UUID) error {
	p.l.Debugf("Character [%d] attempting to disband guild.", characterId)
	return database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
		g, err := p.WithTransaction(tx).GetByMemberId(characterId)
		if err != nil {
			return err
		}
		if g.LeaderId() != characterId {
			return errors.New("must be leader")
		}

		members := make([]uint32, 0)
		for _, gm := range g.Members() {
			members = append(members, gm.CharacterId())
			_ = member.NewProcessor(p.l, p.ctx, tx).RemoveMember(g.Id(), gm.CharacterId())
		}
		_ = title.NewProcessor(p.l, p.ctx, tx).Clear(g.Id())
		_ = deleteGuild(tx, p.t.Id(), g.Id())

		_ = producer.ProviderImpl(p.l)(p.ctx)(guild2.EnvStatusEventTopic)(statusEventDisbandedProvider(g.WorldId(), g.Id(), members, transactionId))
		return nil
	})
}

func (p *ProcessorImpl) RequestCapacityIncrease(characterId uint32, transactionId uuid.UUID) error {
	g, err := p.GetByMemberId(characterId)
	if err != nil {
		return err
	}
	if g.LeaderId() != characterId {
		return errors.New("must be leader")
	}

	p.l.Debugf("Character [%d] is attempting to increase guild [%d] capacity.", characterId, g.Id())
	g, err = updateCapacity(p.db, p.t.Id(), g.Id())
	if err != nil {
		return err
	}
	_ = producer.ProviderImpl(p.l)(p.ctx)(guild2.EnvStatusEventTopic)(statusEventCapacityUpdatedProvider(g.WorldId(), g.Id(), g.Capacity(), transactionId))
	return nil
}
