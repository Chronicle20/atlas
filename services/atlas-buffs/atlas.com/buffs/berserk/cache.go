package berserk

import (
	"atlas-buffs/external/dataskill"
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// EffectXCache caches Berserk's per-level effect x values per tenant. Effect
// data is immutable for a tenant's lifetime, so one atlas-data fetch per
// tenant suffices (design D5). Failed fetches are not cached.
type EffectXCache struct {
	mu       sync.RWMutex
	byTenant map[uuid.UUID][]int16
	fetch    func(l logrus.FieldLogger, ctx context.Context) (dataskill.RestModel, error)
}

func NewEffectXCache(fetch func(l logrus.FieldLogger, ctx context.Context) (dataskill.RestModel, error)) *EffectXCache {
	return &EffectXCache{
		byTenant: make(map[uuid.UUID][]int16),
		fetch:    fetch,
	}
}

var (
	effectXCache     *EffectXCache
	effectXCacheOnce sync.Once
)

func GetEffectXCache() *EffectXCache {
	effectXCacheOnce.Do(func() {
		effectXCache = NewEffectXCache(func(l logrus.FieldLogger, ctx context.Context) (dataskill.RestModel, error) {
			return dataskill.RequestById(uint32(skill.DarkKnightBerserkId))(l, ctx)
		})
	})
	return effectXCache
}

func (c *EffectXCache) X(l logrus.FieldLogger, ctx context.Context, skillLevel byte) (int16, error) {
	t := tenant.MustFromContext(ctx)

	c.mu.RLock()
	xs, ok := c.byTenant[t.Id()]
	c.mu.RUnlock()

	if !ok {
		rm, err := c.fetch(l, ctx)
		if err != nil {
			return 0, err
		}
		xs = make([]int16, 0, len(rm.Effects))
		for _, e := range rm.Effects {
			xs = append(xs, e.X)
		}
		c.mu.Lock()
		c.byTenant[t.Id()] = xs
		c.mu.Unlock()
	}

	if skillLevel == 0 || int(skillLevel) > len(xs) {
		return 0, fmt.Errorf("no effect data for skill [%d] level [%d]", uint32(skill.DarkKnightBerserkId), skillLevel)
	}
	return xs[skillLevel-1], nil
}
