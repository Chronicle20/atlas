package mock

import (
	"atlas-pets/pet"
	"context"

	"github.com/Chronicle20/atlas-tenant"
)

type TemporalRegistry struct {
	UpdatePositionFn func(ctx context.Context, t tenant.Model, petId uint32, x int16, y int16, fh int16)
	UpdateFn         func(ctx context.Context, t tenant.Model, petId uint32, x int16, y int16, stance byte, fh int16)
	UpdateStanceFn   func(ctx context.Context, t tenant.Model, petId uint32, stance byte)
	GetByIdFn        func(ctx context.Context, t tenant.Model, petId uint32) *pet.TemporalData
	RemoveFn         func(ctx context.Context, t tenant.Model, petId uint32)
}

func (m *TemporalRegistry) UpdatePosition(ctx context.Context, t tenant.Model, petId uint32, x int16, y int16, fh int16) {
	if m.UpdatePositionFn != nil {
		m.UpdatePositionFn(ctx, t, petId, x, y, fh)
	}
}

func (m *TemporalRegistry) Update(ctx context.Context, t tenant.Model, petId uint32, x int16, y int16, stance byte, fh int16) {
	if m.UpdateFn != nil {
		m.UpdateFn(ctx, t, petId, x, y, stance, fh)
	}
}

func (m *TemporalRegistry) UpdateStance(ctx context.Context, t tenant.Model, petId uint32, stance byte) {
	if m.UpdateStanceFn != nil {
		m.UpdateStanceFn(ctx, t, petId, stance)
	}
}

func (m *TemporalRegistry) GetById(ctx context.Context, t tenant.Model, petId uint32) *pet.TemporalData {
	if m.GetByIdFn != nil {
		return m.GetByIdFn(ctx, t, petId)
	}
	return nil
}

func (m *TemporalRegistry) Remove(ctx context.Context, t tenant.Model, petId uint32) {
	if m.RemoveFn != nil {
		m.RemoveFn(ctx, t, petId)
	}
}
