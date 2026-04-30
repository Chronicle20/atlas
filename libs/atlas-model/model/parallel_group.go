package model

import (
	"context"

	"golang.org/x/sync/errgroup"
)

// Group runs heterogeneously-typed providers concurrently. It is a thin
// wrapper around errgroup.Group that pairs each registered provider with a
// typed Future handle so call sites can reclaim results without runtime
// type assertions.
type Group struct {
	g *errgroup.Group
}

// Future holds the result of a provider submitted to a Group. After Wait
// returns nil, Get returns the provider's successful value. Get's behaviour
// is undefined when Wait returned an error.
type Future[T any] struct {
	value T
}

// Get returns the value produced by the provider this Future represents.
// Only valid after the parent Group's Wait has returned nil.
func (f *Future[T]) Get() T { return f.value }

// NewGroup returns a Group bound to a child of ctx. The child context is
// cancelled when any submitted provider returns a non-nil error or when
// Wait completes.
func NewGroup(ctx context.Context) (*Group, context.Context) {
	g, gctx := errgroup.WithContext(ctx)
	return &Group{g: g}, gctx
}

// Submit registers a provider with the group, returning a typed Future.
// Submit is a free function rather than a method because Go does not allow
// type parameters on methods.
func Submit[T any](g *Group, p Provider[T]) *Future[T] {
	f := &Future[T]{}
	g.g.Go(func() error {
		v, err := p()
		if err != nil {
			return err
		}
		f.value = v
		return nil
	})
	return f
}

// Wait blocks until all submitted providers complete and returns the first
// non-nil error, if any.
func (g *Group) Wait() error { return g.g.Wait() }
