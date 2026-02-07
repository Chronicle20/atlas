package reward

import (
	"atlas-gachapons/gachapon"
	"atlas-gachapons/global"
	"atlas-gachapons/item"
	"context"
	"crypto/rand"
	"errors"
	"math/big"

	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type poolItem struct {
	ItemId   uint32
	Quantity uint32
}

type Processor interface {
	SelectReward(gachaponId string) (Model, error)
	GetPrizePool(gachaponId string, tier string) ([]Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	t := tenant.MustFromContext(ctx)
	return &ProcessorImpl{l: l, ctx: ctx, db: db, t: t}
}

func (p *ProcessorImpl) SelectReward(gachaponId string) (Model, error) {
	g, err := gachapon.NewProcessor(p.l, p.ctx, p.db).GetById(gachaponId)
	if err != nil {
		return Model{}, err
	}

	tier, err := selectTier(g.CommonWeight(), g.UncommonWeight(), g.RareWeight())
	if err != nil {
		return Model{}, err
	}

	pool, err := p.getMergedPool(gachaponId, tier)
	if err != nil {
		return Model{}, err
	}

	if len(pool) == 0 {
		return Model{}, errors.New("no items available in pool for tier: " + tier)
	}

	selected, err := selectItem(pool)
	if err != nil {
		return Model{}, err
	}

	result := NewBuilder(gachaponId).
		SetItemId(selected.ItemId).
		SetQuantity(selected.Quantity).
		SetTier(tier).
		Build()

	p.l.WithFields(logrus.Fields{
		"gachapon_id": gachaponId,
		"tier":        tier,
		"item_id":     selected.ItemId,
		"quantity":    selected.Quantity,
	}).Infof("Gachapon reward selected.")

	return result, nil
}

func (p *ProcessorImpl) GetPrizePool(gachaponId string, tier string) ([]Model, error) {
	tiers := []string{"common", "uncommon", "rare"}
	if tier != "" {
		tiers = []string{tier}
	}

	var results []Model
	for _, t := range tiers {
		pool, err := p.getMergedPool(gachaponId, t)
		if err != nil {
			return nil, err
		}
		for _, pi := range pool {
			results = append(results, NewBuilder(gachaponId).
				SetItemId(pi.ItemId).
				SetQuantity(pi.Quantity).
				SetTier(t).
				Build())
		}
	}
	return results, nil
}

func (p *ProcessorImpl) getMergedPool(gachaponId string, tier string) ([]poolItem, error) {
	machineItems, err := item.NewProcessor(p.l, p.ctx, p.db).GetByGachaponIdAndTier(gachaponId, tier)()
	if err != nil {
		return nil, err
	}

	globalItems, err := global.NewProcessor(p.l, p.ctx, p.db).GetByTier(tier)()
	if err != nil {
		return nil, err
	}

	var pool []poolItem
	for _, mi := range machineItems {
		pool = append(pool, poolItem{ItemId: mi.ItemId(), Quantity: mi.Quantity()})
	}
	for _, gi := range globalItems {
		pool = append(pool, poolItem{ItemId: gi.ItemId(), Quantity: gi.Quantity()})
	}
	return pool, nil
}

func selectTier(commonWeight uint32, uncommonWeight uint32, rareWeight uint32) (string, error) {
	totalWeight := commonWeight + uncommonWeight + rareWeight
	if totalWeight == 0 {
		return "", errors.New("total weight cannot be zero")
	}

	n, err := rand.Int(rand.Reader, big.NewInt(int64(totalWeight)))
	if err != nil {
		return "", err
	}
	roll := uint32(n.Int64())

	if roll < commonWeight {
		return "common", nil
	}
	if roll < commonWeight+uncommonWeight {
		return "uncommon", nil
	}
	return "rare", nil
}

func selectItem(pool []poolItem) (poolItem, error) {
	if len(pool) == 0 {
		return poolItem{}, errors.New("empty pool")
	}

	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(pool))))
	if err != nil {
		return poolItem{}, err
	}
	return pool[n.Int64()], nil
}
