package area_info

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
)

// Processor provides read access to a character's stored area-info state.
type Processor interface {
	GetByArea(characterId uint32, area uint16) (Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewProcessor creates a new area-info processor.
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

var _ Processor = (*ProcessorImpl)(nil)

// GetByArea retrieves the stored area-info string for a character/area.
func (p *ProcessorImpl) GetByArea(characterId uint32, area uint16) (Model, error) {
	resp, err := requestAreaInfo(p.ctx, characterId, area)(p.l, p.ctx)
	if err != nil {
		return Model{}, fmt.Errorf("failed to get area info for character %d area %d: %w", characterId, area, err)
	}
	return Extract(resp)
}
