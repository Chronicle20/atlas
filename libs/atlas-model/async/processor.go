package async

import (
	"context"
	"errors"
	"time"

	"github.com/Chronicle20/atlas-model/model"
)

var ErrAwaitTimeout = errors.New("timeout")

type Config struct {
	ctx     context.Context
	timeout time.Duration
}

type Configurator func(*Config)

//goland:noinspection GoUnusedExportedFunction
func SetContext(ctx context.Context) Configurator {
	return func(config *Config) {
		config.ctx = ctx
	}
}

//goland:noinspection GoUnusedExportedFunction
func SetTimeout(timeout time.Duration) Configurator {
	return func(config *Config) {
		config.timeout = timeout
	}
}

type Provider[M any] func(ctx context.Context, rchan chan M, echan chan error)

//goland:noinspection GoUnusedExportedFunction
func SingleProvider[M any](p Provider[M]) model.Provider[Provider[M]] {
	return model.FixedProvider[Provider[M]](p)
}

//goland:noinspection GoUnusedExportedFunction
func FixedProvider[M any](ps []Provider[M]) model.Provider[[]Provider[M]] {
	return model.FixedProvider[[]Provider[M]](ps)
}

//goland:noinspection GoUnusedExportedFunction
func Await[M any](provider model.Provider[Provider[M]], configurators ...Configurator) model.Provider[M] {
	return model.FirstProvider(AwaitSlice(model.ToSliceProvider(provider), configurators...), model.Filters[M]())
}

//goland:noinspection GoUnusedExportedFunction
func AwaitSlice[M any](provider model.Provider[[]Provider[M]], configurators ...Configurator) model.Provider[[]M] {
	return func() ([]M, error) {
		c := &Config{ctx: context.Background(), timeout: 500 * time.Millisecond}
		for _, configurator := range configurators {
			configurator(c)
		}

		ctx, cancel := context.WithTimeout(c.ctx, c.timeout)
		defer cancel()

		providers, err := provider()
		if err != nil {
			return nil, err
		}

		resultChannels := make(chan M, len(providers))
		errChannels := make(chan error, len(providers))

		for _, provider := range providers {
			p := provider
			go func() {
				p(ctx, resultChannels, errChannels)
			}()
		}

		var results = make([]M, 0)
		for i := 0; i < len(providers); i++ {
			select {
			case result := <-resultChannels:
				results = append(results, result)
			case <-ctx.Done():
				return nil, ErrAwaitTimeout
			case err := <-errChannels:
				return nil, err
			}
		}
		return results, nil
	}
}
