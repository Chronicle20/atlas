package async

import (
	"context"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"time"
)

var ErrAwaitTimeout = errors.New("timeout")

type Config struct {
	timeout time.Duration
}

type Configurator func(*Config)

func SetTimeout(timeout time.Duration) Configurator {
	return func(config *Config) {
		config.timeout = timeout
	}
}

type Provider[M any] func(ctx context.Context, rchan chan M, echan chan error)

func SingleProvider[M any](p Provider[M]) model.Provider[Provider[M]] {
	return model.FixedProvider[Provider[M]](p)
}

func FixedProvider[M any](ps []Provider[M]) model.SliceProvider[Provider[M]] {
	return model.FixedSliceProvider[Provider[M]](ps)
}

func Await[M any](provider model.Provider[Provider[M]], configurators ...Configurator) model.Provider[M] {
	p, err := provider()
	if err != nil {
		return model.ErrorProvider[M](err)
	}

	m, err := model.First(AwaitSlice(model.FixedSingleSliceProvider(p), configurators...))
	if err != nil {
		return model.ErrorProvider[M](err)
	}
	return model.FixedProvider[M](m)
}

func AwaitSlice[M any](provider model.SliceProvider[Provider[M]], configurators ...Configurator) model.SliceProvider[M] {
	c := &Config{timeout: 5 * time.Second}
	for _, configurator := range configurators {
		configurator(c)
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	providers, err := provider()
	if err != nil {
		return model.ErrorSliceProvider[M](err)
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
			err = ErrAwaitTimeout
		case <-errChannels:
			err = <-errChannels
		}
	}
	return func() ([]M, error) {
		return results, err
	}
}
