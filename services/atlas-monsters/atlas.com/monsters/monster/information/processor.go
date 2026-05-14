package information

import (
	"context"
	"errors"

	redislib "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

// GetById returns the parsed monster information for monsterId, served
// from a tenant-scoped Redis-backed read-through cache when enabled.
// Signature is preserved for all existing call sites.
func GetById(l logrus.FieldLogger) func(ctx context.Context) func(monsterId uint32) (Model, error) {
	return func(ctx context.Context) func(monsterId uint32) (Model, error) {
		return func(monsterId uint32) (Model, error) {
			if dataCachePtr == nil || !dataCachePtr.cfg.enabled {
				return upstreamAndExtract(l, ctx, monsterId)
			}
			c := dataCachePtr

			t := tenant.MustFromContext(ctx)
			tenantStr := t.Id().String()

			// Positive lookup.
			if rm, err := c.posReg.Get(ctx, t, monsterId); err == nil {
				hitsTotal.WithLabelValues(tenantStr, "positive").Inc()
				return Extract(rm)
			} else if !errors.Is(err, redislib.ErrNotFound) {
				redisErrorsTotal.WithLabelValues(tenantStr, "get_positive").Inc()
				l.WithError(err).Debug("data cache positive lookup failed; falling through")
			}

			// Negative lookup (only if NegativeTTL > 0).
			if c.cfg.negativeTTL > 0 {
				if _, nerr := c.negReg.Get(ctx, t, monsterId); nerr == nil {
					hitsTotal.WithLabelValues(tenantStr, "negative").Inc()
					return Model{}, notFoundError(monsterId)
				} else if !errors.Is(nerr, redislib.ErrNotFound) {
					redisErrorsTotal.WithLabelValues(tenantStr, "get_negative").Inc()
					l.WithError(nerr).Debug("data cache negative lookup failed; falling through")
				}
			}

			// True miss → upstream.
			missesTotal.WithLabelValues(tenantStr).Inc()
			rm, ferr := upstreamFn(l, ctx, monsterId)
			if ferr == nil {
				if perr := c.posReg.PutWithTTL(ctx, t, monsterId, rm, c.cfg.ttl); perr != nil {
					redisErrorsTotal.WithLabelValues(tenantStr, "put_positive").Inc()
					l.WithError(perr).Debug("data cache positive put failed; serving fetched value uncached")
				}
				return Extract(rm)
			}
			switch classifyError(ferr) {
			case errKindNotFound:
				errorsTotal.WithLabelValues(tenantStr, "not_found").Inc()
				if c.cfg.negativeTTL > 0 {
					if perr := c.negReg.PutWithTTL(ctx, t, monsterId, struct{}{}, c.cfg.negativeTTL); perr != nil {
						redisErrorsTotal.WithLabelValues(tenantStr, "put_negative").Inc()
						l.WithError(perr).Debug("data cache negative put failed; not caching")
					}
				}
			default:
				errorsTotal.WithLabelValues(tenantStr, "transient").Inc()
			}
			return Model{}, ferr
		}
	}
}

// upstreamAndExtract is the kill-switch / unwired path: it skips the cache
// entirely and reproduces the pre-task behavior — fetch RestModel and
// Extract.
func upstreamAndExtract(l logrus.FieldLogger, ctx context.Context, monsterId uint32) (Model, error) {
	rm, err := upstreamFn(l, ctx, monsterId)
	if err != nil {
		return Model{}, err
	}
	return Extract(rm)
}
