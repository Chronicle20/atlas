// Package bosshp is the single home for FR-1: whether a monster qualifies
// for the boss HP gauge (Boss() && TagColor() != 0), plus the operator that
// announces the gauge to a session. Call sites that need a boss HP gauge
// must resolve it here rather than re-deriving the rule.
package bosshp

import (
	monsterdata "atlas-channel/data/monster"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	fieldpkt "github.com/Chronicle20/atlas/libs/atlas-packet/field"
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
)

// Gauge is a resolved, qualifying boss HP gauge. Constructed only by
// Resolve, so a Gauge value is proof that FR-1 held.
type Gauge struct {
	monsterId          uint32
	currentHp          uint32
	maxHp              uint32
	tagColor           byte
	tagBackgroundColor byte
}

func (g Gauge) MonsterId() uint32 {
	return g.monsterId
}

func (g Gauge) CurrentHp() uint32 {
	return g.currentHp
}

func (g Gauge) MaxHp() uint32 {
	return g.maxHp
}

func (g Gauge) TagColor() byte {
	return g.tagColor
}

func (g Gauge) TagBackgroundColor() byte {
	return g.tagBackgroundColor
}

// Resolver resolves a monster template's qualification for the boss HP
// gauge against atlas-data.
type Resolver struct {
	p monsterdata.Processor
}

// NewResolver builds a Resolver backed by the production atlas-data
// processor.
func NewResolver(l logrus.FieldLogger, ctx context.Context) *Resolver {
	return NewResolverFrom(monsterdata.NewProcessor(l, ctx))
}

// NewResolverFrom builds a Resolver backed by the given processor. Exposed
// for tests to inject a mock.
func NewResolverFrom(p monsterdata.Processor) *Resolver {
	return &Resolver{p: p}
}

// Resolve reports whether monsterTemplateId qualifies for the gauge. A
// false ok with a nil error means "does not qualify"; a non-nil error means
// the atlas-data lookup failed and the caller must log and skip (FR-17).
func (r *Resolver) Resolve(monsterTemplateId uint32, currentHp uint32, maxHp uint32) (Gauge, bool, error) {
	m, err := r.p.GetById(monsterTemplateId)
	if err != nil {
		return Gauge{}, false, err
	}

	if !m.Boss() || m.TagColor() == 0 {
		return Gauge{}, false, nil
	}

	return Gauge{
		monsterId:          monsterTemplateId,
		currentHp:          currentHp,
		maxHp:              maxHp,
		tagColor:           m.TagColor(),
		tagBackgroundColor: m.TagBackgroundColor(),
	}, true, nil
}

// AnnounceOperator sends one BOSS_HP field effect to a session.
func AnnounceOperator(l logrus.FieldLogger) func(context.Context) func(writer.Producer) func(Gauge) model.Operator[session.Model] {
	return func(ctx context.Context) func(writer.Producer) func(Gauge) model.Operator[session.Model] {
		return func(wp writer.Producer) func(Gauge) model.Operator[session.Model] {
			return func(g Gauge) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(fieldcb.FieldEffectWriter)(fieldpkt.FieldEffectBossHpBody(g.monsterId, g.currentHp, g.maxHp, g.tagColor, g.tagBackgroundColor))
			}
		}
	}
}
