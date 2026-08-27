package data

import (
	"context"
	"errors"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

type SkillInfo struct {
	Id       uint32
	Name     string
	MaxLevel uint8
	// EffectX is the per-level HP/MP gain bonus (level 1..N, index 0 ==
	// level 1), sourced from atlas-data's skill effect resource -- see
	// SkillEffectRestModel.
	EffectX []int16
}

// EffectXAt returns the skill's effect X value at the given level, mirroring
// atlas-character's data/skill.Processor.GetEffect: level 0 (unlearned) is
// always 0, and an out-of-range level yields ok=false rather than a panic.
func (s SkillInfo) EffectXAt(level byte) (int16, bool) {
	if level == 0 {
		return 0, true
	}
	idx := int(level) - 1
	if idx < 0 || idx >= len(s.EffectX) {
		return 0, false
	}
	return s.EffectX[idx], true
}

type ItemInfo struct {
	Id        uint32
	Equipable bool
}

var ErrNotFound = errors.New("not found")

type Processor interface {
	GetSkillsByIds(ctx context.Context, ids []uint32) ([]SkillInfo, error)
	GetItemById(ctx context.Context, id uint32) (ItemInfo, error)
}

type ProcessorImpl struct {
	l logrus.FieldLogger
}

func NewProcessor(l logrus.FieldLogger) Processor { return &ProcessorImpl{l: l} }

var _ Processor = (*ProcessorImpl)(nil)

func (c *ProcessorImpl) GetSkillsByIds(ctx context.Context, ids []uint32) ([]SkillInfo, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rms, err := requestSkillsByIds(ctx, ids)(c.l, ctx)
	if err != nil {
		return nil, err
	}
	out := make([]SkillInfo, 0, len(rms))
	for _, rm := range rms {
		effectX := make([]int16, 0, len(rm.Effects))
		for _, e := range rm.Effects {
			effectX = append(effectX, e.X)
		}
		out = append(out, SkillInfo{Id: rm.Id, Name: rm.Name, MaxLevel: rm.MaxLevel, EffectX: effectX})
	}
	return out, nil
}

func (c *ProcessorImpl) GetItemById(ctx context.Context, id uint32) (ItemInfo, error) {
	invType, ok := inventory.TypeFromItemId(item.Id(id))
	if !ok {
		return ItemInfo{}, ErrNotFound
	}
	if invType != inventory.TypeValueEquip {
		// Non-equip items don't have a "/data/equipment/{id}" entry; existence is presumed.
		return ItemInfo{Id: id, Equipable: false}, nil
	}
	if _, err := requestEquipmentById(ctx, id)(c.l, ctx); err != nil {
		// Distinguish a real 404 from atlas-data ("template not present") from
		// any other failure (HTTP transport, JSON:API decode, etc). Surfacing
		// every error as ErrNotFound makes deploy bugs (see task-037, where
		// missing UnmarshalToManyRelations stubs surfaced as "item not found")
		// indistinguishable from genuine missing-data. Log non-404 errors at
		// warn so the cause is one log line away.
		if errors.Is(err, requests.ErrNotFound) {
			return ItemInfo{}, ErrNotFound
		}
		c.l.WithError(err).Warnf("atlas-data lookup for equipment [%d] failed (non-404)", id)
		return ItemInfo{}, err
	}
	return ItemInfo{Id: id, Equipable: true}, nil
}
