package batch

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Builder struct {
	id             uuid.UUID
	description    string
	requestedCount uint32
	generatedCount uint32
	redeemedCount  uint32
	createdAt      time.Time
}

// NewBuilder takes the requested count up front — a batch always exists to
// generate a specific number of codes, so a Builder can never be missing it.
func NewBuilder(requestedCount uint32) *Builder {
	return &Builder{requestedCount: requestedCount}
}

func (b *Builder) SetId(id uuid.UUID) *Builder         { b.id = id; return b }
func (b *Builder) SetDescription(d string) *Builder    { b.description = d; return b }
func (b *Builder) SetGeneratedCount(n uint32) *Builder { b.generatedCount = n; return b }
func (b *Builder) SetRedeemedCount(n uint32) *Builder  { b.redeemedCount = n; return b }
func (b *Builder) SetCreatedAt(t time.Time) *Builder   { b.createdAt = t; return b }

func (b *Builder) Build() (Model, error) {
	if b.requestedCount == 0 {
		return Model{}, fmt.Errorf("%w: requestedCount must be positive", ErrInvalidBatch)
	}
	if b.generatedCount > b.requestedCount {
		return Model{}, fmt.Errorf("%w: generatedCount (%d) exceeds requestedCount (%d)", ErrInvalidBatch, b.generatedCount, b.requestedCount)
	}
	return Model{
		id:             b.id,
		description:    b.description,
		requestedCount: b.requestedCount,
		generatedCount: b.generatedCount,
		redeemedCount:  b.redeemedCount,
		createdAt:      b.createdAt,
	}, nil
}
