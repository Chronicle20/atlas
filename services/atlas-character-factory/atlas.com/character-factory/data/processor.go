package data

import (
	"context"
	"errors"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type SkillInfo struct {
	Id       uint32
	Name     string
	MaxLevel uint8
}

type ItemInfo struct {
	Id        uint32
	Equipable bool
}

var ErrNotFound = errors.New("not found")

type Client interface {
	GetSkillsByIds(ctx context.Context, ids []uint32) ([]SkillInfo, error)
	GetItemById(ctx context.Context, id uint32) (ItemInfo, error)
}

type ClientImpl struct {
	l logrus.FieldLogger
}

func NewClient(l logrus.FieldLogger) *ClientImpl { return &ClientImpl{l: l} }

func (c *ClientImpl) GetSkillsByIds(ctx context.Context, ids []uint32) ([]SkillInfo, error) {
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

func (c *ClientImpl) GetItemById(ctx context.Context, id uint32) (ItemInfo, error) {
	invType, ok := inventory.TypeFromItemId(item.Id(id))
	if !ok {
		return ItemInfo{}, ErrNotFound
	}
	if invType != inventory.TypeValueEquip {
		// Non-equip items don't have a "/data/equipment/{id}" entry; existence is presumed.
		return ItemInfo{Id: id, Equipable: false}, nil
	}
	if _, err := requestEquipmentById(id)(c.l, ctx); err != nil {
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
