package model

import (
	"errors"
	"math/rand"
	"sync"
)

type Operator[M any] func(M) error

type Provider[M any] func() (M, error)

type Decorator[M any] func(M) M

//goland:noinspection GoUnusedExportedFunction
func Flip[A any, B any, C any](f func(A) func(B) C) func(B) func(A) C {
	return func(b B) func(A) C {
		return func(a A) C {
			return f(a)(b)
		}
	}
}

func ExecuteForEach[M any](f Operator[M]) Operator[[]M] {
	return func(models []M) error {
		for _, m := range models {
			err := f(m)
			if err != nil {
				return err
			}
		}
		return nil
	}
}

type Filter[M any] func(M) bool

//goland:noinspection GoUnusedExportedFunction
func FilteredProvider[M any](provider Provider[[]M], filters ...Filter[M]) Provider[[]M] {
	models, err := provider()
	if err != nil {
		return ErrorProvider[[]M](err)
	}

	var results []M
	for _, m := range models {
		good := true
		for _, f := range filters {
			if !f(m) {
				good = false
				break
			}
		}
		if good {
			results = append(results, m)
		}
	}
	return FixedProvider(results)
}

//goland:noinspection GoUnusedExportedFunction
func FixedProvider[M any](model M) Provider[M] {
	return func() (M, error) {
		return model, nil
	}
}

//goland:noinspection GoUnusedExportedFunction
func AsSliceProvider[M any](model M) Provider[[]M] {
	return FixedProvider([]M{model})
}

//goland:noinspection GoUnusedExportedFunction
func ToSliceProvider[M any](provider Provider[M]) Provider[[]M] {
	m, err := provider()
	if err != nil {
		return ErrorProvider[[]M](err)
	}
	return AsSliceProvider(m)
}

// Deprecated: use AsSliceProvider
//
//goland:noinspection GoUnusedExportedFunction
func FixedSliceProvider[M any](model M) Provider[[]M] {
	return FixedProvider([]M{model})
}

//goland:noinspection GoUnusedExportedFunction
func ErrorProvider[M any](err error) Provider[M] {
	return func() (M, error) {
		var m M
		return m, err
	}
}

//goland:noinspection GoUnusedExportedFunction
func RandomPreciselyOneFilter[M any](ms []M) (M, error) {
	var def M
	if len(ms) == 0 {
		return def, errors.New("empty slice")
	}
	return ms[rand.Intn(len(ms))], nil
}

// Deprecated: use For
//
//goland:noinspection GoUnusedExportedFunction
func IfPresent[M any](provider Provider[M], operator Operator[M]) {
	model, err := provider()
	if err != nil {
		return
	}
	_ = operator(model)
}

//goland:noinspection GoUnusedExportedFunction
func For[M any](provider Provider[M], operator Operator[M]) error {
	models, err := provider()
	if err != nil {
		return err
	}
	return operator(models)
}

//goland:noinspection GoUnusedExportedFunction
func ForEach[M any](provider Provider[[]M], operator Operator[M]) error {
	return For(provider, ExecuteForEach(operator))
}

//goland:noinspection GoUnusedExportedFunction
type Transformer[M any, N any] func(M) (N, error)

// Deprecated: just use Map
//
//goland:noinspection GoUnusedExportedFunction
func Transform[M any, N any](model M, transformer Transformer[M, N]) (N, error) {
	return Map(FixedProvider(model), transformer)()
}

//goland:noinspection GoUnusedExportedFunction
func Map[M any, N any](provider Provider[M], transformer Transformer[M, N]) Provider[N] {
	m, err := provider()
	if err != nil {
		return ErrorProvider[N](err)
	}
	n, err := transformer(m)
	if err != nil {
		return ErrorProvider[N](err)
	}
	return FixedProvider(n)
}

type MapFuncConfigurator Decorator[MapConfig]

type MapConfig struct {
	parallel bool
}

func (c MapConfig) SetParallel(val bool) MapConfig {
	return MapConfig{parallel: val}
}

//goland:noinspection GoUnusedExportedFunction
func ParallelMap() MapFuncConfigurator {
	return func(config MapConfig) MapConfig {
		return config.SetParallel(true)
	}
}

type mapResult[E any] struct {
	index int
	value E
	err   error
}

//goland:noinspection GoUnusedExportedFunction
func SliceMap[M any, N any](provider Provider[[]M], transformer Transformer[M, N], configurators ...MapFuncConfigurator) Provider[[]N] {
	c := MapConfig{parallel: false}
	for _, configurator := range configurators {
		c = configurator(c)
	}

	models, err := provider()
	if err != nil {
		return ErrorProvider[[]N](err)
	}
	var results = make([]N, len(models))

	if c.parallel {
		var wg sync.WaitGroup

		resCh := make(chan mapResult[N], len(models))

		for i, m := range models {
			wg.Add(1)
			go parallelTransform(&wg, transformer, i, m, resCh)
		}
		wg.Wait()

		close(resCh)
		for res := range resCh {
			if res.err != nil {
				return ErrorProvider[[]N](res.err)
			}
			results[res.index] = res.value
		}
	} else {
		for i, m := range models {
			var n N
			n, err = transformer(m)
			if err != nil {
				return ErrorProvider[[]N](err)
			}
			results[i] = n
		}
	}
	return FixedProvider(results)
}

func parallelTransform[M any, N any](wg *sync.WaitGroup, transformer Transformer[M, N], index int, model M, resCh chan<- mapResult[N]) {
	defer wg.Done()
	r, err := transformer(model)
	if err != nil {
		resCh <- mapResult[N]{index: index, err: err}
		return
	}
	resCh <- mapResult[N]{index: index, value: r}
}

//goland:noinspection GoUnusedExportedFunction
type Folder[M any, N any] func(N, M) (N, error)

//goland:noinspection GoUnusedExportedFunction
func Fold[M any, N any](provider Provider[[]M], supplier Provider[N], folder Folder[M, N]) Provider[N] {
	ms, err := provider()
	if err != nil {
		return ErrorProvider[N](err)
	}

	n, err := supplier()
	if err != nil {
		return ErrorProvider[N](err)
	}

	for _, wip := range ms {
		n, err = folder(n, wip)
		if err != nil {
			return ErrorProvider[N](err)
		}
	}
	return FixedProvider(n)
}

//goland:noinspection GoUnusedExportedFunction
func Decorate[M any](decorators ...Decorator[M]) func(m M) (M, error) {
	return func(m M) (M, error) {
		var n = m
		for _, d := range decorators {
			n = d(n)
		}
		return n, nil
	}
}

//goland:noinspection GoUnusedExportedFunction
func FirstProvider[M any](provider Provider[[]M], filters ...Filter[M]) Provider[M] {
	ms, err := provider()
	if err != nil {
		return ErrorProvider[M](err)
	}

	if len(ms) == 0 {
		return ErrorProvider[M](errors.New("empty slice"))
	}

	if len(filters) == 0 {
		return FixedProvider[M](ms[0])
	}

	for _, m := range ms {
		ok := true
		for _, filter := range filters {
			if !filter(m) {
				ok = false
			}
		}
		if ok {
			return FixedProvider[M](m)
		}
	}
	return ErrorProvider[M](errors.New("no result found"))
}

//goland:noinspection GoUnusedExportedFunction
func First[M any](provider Provider[[]M], filters ...Filter[M]) (M, error) {
	return FirstProvider(provider, filters...)()
}
