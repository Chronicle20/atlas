package ranking

import (
	"atlas-rankings/character"
	"context"
	"errors"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// CharacterSupplier abstracts the atlas-character scan so tests can inject
// fixtures without an HTTP server.
type CharacterSupplier func() ([]character.Model, error)

type Processor interface {
	ByCharacterIdProvider(characterId uint32) model.Provider[Model]
	GetByCharacterId(characterId uint32) (Model, error)
	ByCharacterIdsProvider(characterIds []uint32) model.Provider[[]Model]
	GetByCharacterIds(characterIds []uint32) ([]Model, error)
	// IsDue reports whether the tenant's recompute interval has elapsed
	// since the last cycle start (true when no cycle has ever run).
	IsDue(interval time.Duration, now time.Time) (bool, error)
	// Recompute scans characters, ranks them, upserts rows stamped with
	// now, prunes rows older than now, and records the cycle. Ranks are
	// idempotent and convergent — a crashed run's ranks are fully repaired
	// by the next one. The move fields (OverallRankMove/JobRankMove) are
	// not: a crash between upsertBatch and completeCycle, or any
	// back-to-back double-run, makes the next cycle build prevById from
	// its own freshly-written ranks, so Move(new, new) reads 0 for every
	// unchanged character for that one cycle. This is structural to the
	// single-row schema (there is no previous-rank column) and self-heals
	// on the following cycle.
	Recompute(now time.Time) error
	WithCharacterSupplier(s CharacterSupplier) Processor
}

type ProcessorImpl struct {
	l          logrus.FieldLogger
	ctx        context.Context
	db         *gorm.DB
	t          tenant.Model
	characters CharacterSupplier
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	cp := character.NewProcessor(l, ctx)
	return &ProcessorImpl{
		l:          l,
		ctx:        ctx,
		db:         db,
		t:          tenant.MustFromContext(ctx),
		characters: cp.GetAll,
	}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) WithCharacterSupplier(s CharacterSupplier) Processor {
	return &ProcessorImpl{l: p.l, ctx: p.ctx, db: p.db, t: p.t, characters: s}
}

func (p *ProcessorImpl) ByCharacterIdProvider(characterId uint32) model.Provider[Model] {
	return model.Map(Make)(byCharacterIdEntityProvider(characterId)(p.db.WithContext(p.ctx)))
}

func (p *ProcessorImpl) GetByCharacterId(characterId uint32) (Model, error) {
	return p.ByCharacterIdProvider(characterId)()
}

func (p *ProcessorImpl) ByCharacterIdsProvider(characterIds []uint32) model.Provider[[]Model] {
	return model.SliceMap(Make)(byCharacterIdsEntityProvider(characterIds)(p.db.WithContext(p.ctx)))(model.ParallelMap())
}

func (p *ProcessorImpl) GetByCharacterIds(characterIds []uint32) ([]Model, error) {
	return p.ByCharacterIdsProvider(characterIds)()
}

func (p *ProcessorImpl) IsDue(interval time.Duration, now time.Time) (bool, error) {
	c, err := cycleEntityProvider()(p.db.WithContext(p.ctx))()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return now.Sub(c.LastStartedAt) >= interval, nil
}

func (p *ProcessorImpl) Recompute(now time.Time) error {
	tdb := p.db.WithContext(p.ctx)
	wallStart := time.Now()

	if err := startCycle(tdb, p.t.Id(), now); err != nil {
		return err
	}

	cs, err := p.characters()
	if err != nil {
		return err
	}

	inputs := make([]Input, 0, len(cs))
	for _, c := range cs {
		if c.Gm() > 0 {
			continue
		}
		inputs = append(inputs, Input{
			CharacterId: c.Id(),
			WorldId:     c.WorldId(),
			JobId:       c.JobId(),
			Level:       c.Level(),
			Experience:  c.Experience(),
		})
	}

	ranked := Rank(inputs)

	prev, err := allEntityProvider()(tdb)()
	if err != nil {
		return err
	}
	prevById := make(map[uint32]Entity, len(prev))
	for _, e := range prev {
		prevById[e.CharacterId] = e
	}

	entities := make([]Entity, 0, len(ranked))
	worldCounts := make(map[byte]int)
	for _, r := range ranked {
		var prevOverall, prevJob uint32
		if pe, ok := prevById[r.CharacterId]; ok {
			prevOverall = pe.OverallRank
			prevJob = pe.JobRank
		}
		entities = append(entities, Entity{
			CharacterId:     r.CharacterId,
			WorldId:         r.WorldId,
			JobCategory:     r.JobCategory,
			OverallRank:     r.OverallRank,
			OverallRankMove: Move(prevOverall, r.OverallRank),
			JobRank:         r.JobRank,
			JobRankMove:     Move(prevJob, r.JobRank),
			ComputedAt:      now,
		})
		worldCounts[byte(r.WorldId)]++
	}

	if err := upsertBatch(tdb, p.t.Id(), entities); err != nil {
		return err
	}

	// Guard: an entirely empty character scan against a non-empty rankings
	// table is indistinguishable, from here, between a genuinely-emptied
	// tenant and a transient empty-without-error scan (e.g.
	// character.Processor.GetAll draining an HTTP 200 with an empty data
	// array — a known failure mode in this codebase). Pruning
	// unconditionally in that case would zero every live player's rank
	// for up to a full recompute interval. Skipping the prune leaves dead
	// rows for a genuinely-emptied tenant, which is the safer failure:
	// nothing reads rankings for characters that no longer exist, whereas
	// every live player's rank is player-visible. The cycle itself must
	// still be recorded either way so IsDue advances.
	//
	// This is gated on the raw scan (cs), not the post-filter entities
	// slice: a scan that returns real characters who are all GM or all
	// departed since last cycle is a legitimate zero-entities cycle and
	// must still prune (e.g. the last non-GM character on a world just
	// got GM'd — their stale row must go).
	if len(cs) == 0 && len(prev) > 0 {
		p.l.WithFields(logrus.Fields{
			"tenant":        p.t.Id().String(),
			"existing_rows": len(prev),
		}).Warnf("Rankings recompute character scan returned zero characters while %d existing rows remain for this tenant; skipping prune to avoid wiping live rankings on a possibly-transient empty scan.", len(prev))
	} else if err := pruneBefore(tdb, now); err != nil {
		return err
	}

	duration := time.Since(wallStart)
	if err := completeCycle(tdb, p.t.Id(), time.Now(), uint32(len(entities)), uint32(duration.Milliseconds())); err != nil {
		return err
	}

	p.l.WithFields(logrus.Fields{
		"tenant":      p.t.Id().String(),
		"ranked":      len(entities),
		"worlds":      len(worldCounts),
		"world_sizes": worldCounts,
		"duration":    duration.String(),
	}).Infof("Rankings recompute cycle completed.")
	return nil
}
