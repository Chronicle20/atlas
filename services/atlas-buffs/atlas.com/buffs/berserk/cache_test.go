package berserk

import (
	"context"
	"errors"
	"testing"

	"atlas-buffs/external/dataskill"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func fixedSkill(xs ...int16) dataskill.RestModel {
	effects := make([]dataskill.EffectModel, 0, len(xs))
	for _, x := range xs {
		effects = append(effects, dataskill.EffectModel{X: x})
	}
	return dataskill.RestModel{Effects: effects}
}

func cacheCtx(t *testing.T) context.Context {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	assert.NoError(t, err)
	return tenant.WithContext(context.Background(), ten)
}

func TestEffectXCacheResolvesPerLevel(t *testing.T) {
	calls := 0
	c := NewEffectXCache(func(_ logrus.FieldLogger, _ context.Context) (dataskill.RestModel, error) {
		calls++
		return fixedSkill(21, 22, 23), nil
	})
	l := logrus.New()
	ctx := cacheCtx(t)

	x, err := c.X(l, ctx, 1)
	assert.NoError(t, err)
	assert.Equal(t, int16(21), x)

	x, err = c.X(l, ctx, 3)
	assert.NoError(t, err)
	assert.Equal(t, int16(23), x)

	assert.Equal(t, 1, calls, "effect data is immutable per tenant: fetched once")
}

func TestEffectXCacheTenantScoped(t *testing.T) {
	calls := 0
	c := NewEffectXCache(func(_ logrus.FieldLogger, _ context.Context) (dataskill.RestModel, error) {
		calls++
		return fixedSkill(21), nil
	})
	l := logrus.New()

	_, err := c.X(l, cacheCtx(t), 1)
	assert.NoError(t, err)
	_, err = c.X(l, cacheCtx(t), 1)
	assert.NoError(t, err)
	assert.Equal(t, 2, calls, "one fetch per tenant")
}

func TestEffectXCacheBounds(t *testing.T) {
	c := NewEffectXCache(func(_ logrus.FieldLogger, _ context.Context) (dataskill.RestModel, error) {
		return fixedSkill(21, 22), nil
	})
	l := logrus.New()
	ctx := cacheCtx(t)

	_, err := c.X(l, ctx, 0)
	assert.Error(t, err, "level 0 has no effect entry")
	_, err = c.X(l, ctx, 3)
	assert.Error(t, err, "level beyond data is an error, not a panic")
}

func TestEffectXCacheFetchFailureNotCached(t *testing.T) {
	fail := true
	c := NewEffectXCache(func(_ logrus.FieldLogger, _ context.Context) (dataskill.RestModel, error) {
		if fail {
			return dataskill.RestModel{}, errors.New("boom")
		}
		return fixedSkill(21), nil
	})
	l := logrus.New()
	ctx := cacheCtx(t)

	_, err := c.X(l, ctx, 1)
	assert.Error(t, err)

	fail = false
	x, err := c.X(l, ctx, 1)
	assert.NoError(t, err, "failed fetch must not poison the cache")
	assert.Equal(t, int16(21), x)
}
