package mist

import (
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

// ProtectionBuilder constructs a Protection via fluent setters.
type ProtectionBuilder struct {
	p Protection
}

// NewProtectionBuilder starts a Protection anchored to a mist id and field.
func NewProtectionBuilder(id uuid.UUID, f field.Model) *ProtectionBuilder {
	return &ProtectionBuilder{p: Protection{id: id, f: f}}
}

func (b *ProtectionBuilder) SetOwnerId(v uint32) *ProtectionBuilder {
	b.p.ownerId = v
	return b
}

// SetRect sets the ABSOLUTE world-coordinate bounding box (origin already
// added to the lt/rb offsets).
func (b *ProtectionBuilder) SetRect(minX, minY, maxX, maxY int16) *ProtectionBuilder {
	b.p.minX, b.p.minY, b.p.maxX, b.p.maxY = minX, minY, maxX, maxY
	return b
}

func (b *ProtectionBuilder) SetExpiresAt(v time.Time) *ProtectionBuilder {
	b.p.expiresAt = v
	return b
}

func (b *ProtectionBuilder) Build() Protection { return b.p }
