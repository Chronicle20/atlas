package data

import (
	"context"
	"errors"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

// SkillInfo holds the subset of skill data needed for preset validation.
type SkillInfo struct {
	Id       uint32
	Name     string
	MaxLevel uint8
}

// ItemInfo holds the subset of item data needed for preset validation.
// Equipable is derived locally from inventory.TypeFromItemId — no extra round-trip needed.
type ItemInfo struct {
	Id        uint32
	Equipable bool
}

// ErrNotFound is returned when the requested resource does not exist in atlas-data.
var ErrNotFound = errors.New("not found")

// Processor is the interface that atlas-data callers must satisfy.
// A mock implementation lives in the mock sub-package.
type Processor interface {
	GetSkillsByIds(ctx context.Context, ids []uint32) ([]SkillInfo, error)
	GetItemById(ctx context.Context, id uint32) (ItemInfo, error)
}

// ProcessorImpl is the real HTTP-backed implementation.
type ProcessorImpl struct {
	l logrus.FieldLogger
}

// NewProcessor constructs a real Processor backed by atlas-data REST calls.
func NewProcessor(l logrus.FieldLogger) Processor {
	return &ProcessorImpl{l: l}
}

var _ Processor = (*ProcessorImpl)(nil)

// GetSkillsByIds fetches skill metadata for the given IDs in a single batched
// request to GET /data/skills?ids=<csv>.
func (c *ProcessorImpl) GetSkillsByIds(ctx context.Context, ids []uint32) ([]SkillInfo, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rms, err := requestSkillsByIds(ids)(c.l, ctx)
	if err != nil {
		return nil, err
	}
	out := make([]SkillInfo, 0, len(rms))
	for _, rm := range rms {
		out = append(out, SkillInfo{Id: rm.Id, Name: rm.Name, MaxLevel: rm.MaxLevel})
	}
	return out, nil
}

// GetItemById checks whether an item template exists in atlas-data and whether it
// is equippable. Existence is verified via GET /data/equipment/{id}; equippability
// is computed locally using inventory.TypeFromItemId so no second request is needed.
//
// If atlas-data returns 404, ErrNotFound is returned.
func (c *ProcessorImpl) GetItemById(ctx context.Context, id uint32) (ItemInfo, error) {
	invType, ok := inventory.TypeFromItemId(item.Id(id))

	// For non-equip items, inventory.TypeFromItemId tells us they're not equippable
	// without needing an atlas-data round-trip. We still attempt a lightweight lookup
	// to confirm the template ID is known, but only for equip-range IDs where
	// atlas-data has dedicated equipment records.
	if !ok || invType != inventory.TypeValueEquip {
		return ItemInfo{Id: id, Equipable: false}, nil
	}

	_, err := requestEquipmentById(id)(c.l, ctx)
	if err != nil {
		// Distinguish a real 404 from atlas-data ("template not present") from
		// any other failure (HTTP transport, JSON:API decode, etc). The
		// validator surfaces ErrNotFound as "item not found in atlas-data";
		// using that error for non-404s makes deploy bugs (see task-037, where
		// missing UnmarshalToManyRelations stubs surfaced as "item not found")
		// indistinguishable from genuine missing-data. Log the underlying
		// error at warn so the next time it happens, the cause is one log
		// line away.
		if errors.Is(err, requests.ErrNotFound) {
			return ItemInfo{}, ErrNotFound
		}
		c.l.WithError(err).Warnf("atlas-data lookup for equipment [%d] failed (non-404)", id)
		return ItemInfo{}, err
	}
	return ItemInfo{Id: id, Equipable: true}, nil
}
