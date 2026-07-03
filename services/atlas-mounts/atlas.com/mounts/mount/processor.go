package mount

import (
	"atlas-mounts/kafka/message"
	mountmsg "atlas-mounts/kafka/message/mount"
	"context"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Processor exposes the persistence-backed mount progression operations. The
// emitting methods take a *message.Buffer so the caller controls the
// transaction/emit boundary (see ApplyTickAndEmit-style wrappers and the
// consumers in later tasks). worldId is always supplied by the caller — it is
// never stored on the model.
type Processor interface {
	With(opts ...ProcessorOption) *ProcessorImpl
	GetByCharacterId(characterId uint32) (Model, error)
	ApplyTick(mb *message.Buffer) func(worldId world.Id, characterId uint32) error
	ApplyFeedAndEmit(mb *message.Buffer) func(worldId world.Id, characterId uint32, healMax int) error
	EmitSet(mb *message.Buffer) func(worldId world.Id, characterId uint32) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) *ProcessorImpl {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
		t:   tenant.MustFromContext(ctx),
	}
}

type ProcessorOption func(*ProcessorImpl)

func WithTransaction(db *gorm.DB) ProcessorOption {
	return func(p *ProcessorImpl) {
		p.db = db
	}
}

func (p *ProcessorImpl) With(opts ...ProcessorOption) *ProcessorImpl {
	clone := *p
	cp := &clone
	for _, opt := range opts {
		opt(cp)
	}
	return cp
}

// GetByCharacterId loads the mount for the given character, scoped to the
// tenant-in-context. If no row exists yet it is created with the default
// progression (level 1 / exp 0 / tiredness 0) and returned (FR-5.4,
// default-on-first-read).
func (p *ProcessorImpl) GetByCharacterId(characterId uint32) (Model, error) {
	e, err := getByCharacterId(p.db.WithContext(p.ctx), characterId)
	if err == nil {
		return Make(e)
	}
	if err != gorm.ErrRecordNotFound {
		return Model{}, err
	}

	p.l.Debugf("No mount row for character [%d]; creating default.", characterId)
	var om Model
	txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		// Re-check inside the transaction to tolerate a concurrent creation.
		ie, ierr := getByCharacterId(tx, characterId)
		if ierr == nil {
			om, ierr = Make(ie)
			return ierr
		}
		if ierr != gorm.ErrRecordNotFound {
			return ierr
		}
		m, berr := NewModelBuilder(p.t.Id(), characterId, uuid.New()).Build()
		if berr != nil {
			return berr
		}
		om, ierr = create(tx)(p.t, m)
		return ierr
	})
	if txErr != nil {
		p.l.WithError(txErr).Errorf("Unable to create default mount for character [%d].", characterId)
		return Model{}, txErr
	}
	return om, nil
}

// ApplyTick advances the mount's tiredness by one tick (default-on-read),
// persists the new tiredness and LastTirednessTickAt, and buffers a TICK event.
// The DB write and the event Put share one transaction so a crash neither loses
// the tiredness increment nor emits without persisting.
func (p *ProcessorImpl) ApplyTick(mb *message.Buffer) func(worldId world.Id, characterId uint32) error {
	return func(worldId world.Id, characterId uint32) error {
		p.l.Debugf("Applying tiredness tick for character [%d] mount.", characterId)
		txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			m, err := p.With(WithTransaction(tx)).GetByCharacterId(characterId)
			if err != nil {
				return err
			}
			newTiredness, tooTired := TickTiredness(m.Tiredness())
			now := time.Now()
			nm, err := Clone(m).SetTiredness(newTiredness).SetLastTirednessTickAt(&now).Build()
			if err != nil {
				return err
			}
			if err = update(tx)(nm); err != nil {
				return err
			}
			body := mountmsg.StatusEventBody{
				Level:     nm.Level(),
				Exp:       nm.Exp(),
				Tiredness: nm.Tiredness(),
				LevelUp:   false,
				TooTired:  tooTired,
			}
			return mb.Put(mountmsg.EnvStatusEventTopic, tickEventProvider(worldId, characterId, body))
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Unable to apply tiredness tick for character [%d].", characterId)
			return txErr
		}
		return nil
	}
}

// ApplyFeedAndEmit applies the feed math (heal → exp → level-up) for the
// character's mount (default-on-read), persists the new level/exp/tiredness, and
// buffers a FEED event. healMax is supplied by the caller (sourced from the feed
// event / consumables config) and never hardcoded.
func (p *ProcessorImpl) ApplyFeedAndEmit(mb *message.Buffer) func(worldId world.Id, characterId uint32, healMax int) error {
	return func(worldId world.Id, characterId uint32, healMax int) error {
		p.l.Debugf("Applying feed for character [%d] mount with healMax [%d].", characterId, healMax)
		txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			m, err := p.With(WithTransaction(tx)).GetByCharacterId(characterId)
			if err != nil {
				return err
			}
			res := ApplyFeed(FeedInput{
				Level:     m.Level(),
				Exp:       m.Exp(),
				Tiredness: m.Tiredness(),
				HealMax:   healMax,
			})
			nm, err := Clone(m).
				SetLevel(res.Level).
				SetExp(res.Exp).
				SetTiredness(res.Tiredness).
				Build()
			if err != nil {
				return err
			}
			if err = update(tx)(nm); err != nil {
				return err
			}
			body := mountmsg.StatusEventBody{
				Level:     nm.Level(),
				Exp:       nm.Exp(),
				Tiredness: nm.Tiredness(),
				LevelUp:   res.LevelUp,
				TooTired:  false,
			}
			return mb.Put(mountmsg.EnvStatusEventTopic, feedEventProvider(worldId, characterId, body))
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Unable to apply feed for character [%d].", characterId)
			return txErr
		}
		return nil
	}
}

// EmitSet loads (or default-creates) the character's mount and buffers a SET
// event carrying its current progression. Used on mount activation; it changes
// no progression state.
func (p *ProcessorImpl) EmitSet(mb *message.Buffer) func(worldId world.Id, characterId uint32) error {
	return func(worldId world.Id, characterId uint32) error {
		p.l.Debugf("Emitting set event for character [%d] mount.", characterId)
		m, err := p.GetByCharacterId(characterId)
		if err != nil {
			return err
		}
		body := mountmsg.StatusEventBody{
			Level:     m.Level(),
			Exp:       m.Exp(),
			Tiredness: m.Tiredness(),
			LevelUp:   false,
			TooTired:  false,
		}
		return mb.Put(mountmsg.EnvStatusEventTopic, setEventProvider(worldId, characterId, body))
	}
}
