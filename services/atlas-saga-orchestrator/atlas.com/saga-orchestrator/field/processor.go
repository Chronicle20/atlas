package field

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// Processor is the interface for field-scoped operations against
// atlas-maps.
type Processor interface {
	// ResetField clears a field's objects and restores its spawn points via
	// atlas-maps' POST .../reset -- Cosmic's MapleMap.resetPQ(difficulty)
	// (task-290 G5).
	ResetField(f field.Model, difficulty int) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) ResetField(f field.Model, difficulty int) error {
	_, err := requestResetField(p.ctx, f, difficulty)(p.l, p.ctx)
	if err != nil {
		return fmt.Errorf("failed to reset field: %w", err)
	}
	return nil
}

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "MAPS")
}

// ResetFieldInputRestModel is the body of POST .../reset.
type ResetFieldInputRestModel struct {
	Id         string `json:"-"`
	Difficulty int    `json:"difficulty"`
}

func (r ResetFieldInputRestModel) GetName() string {
	return "maps"
}

func (r ResetFieldInputRestModel) GetID() string {
	return r.Id
}

func requestResetField(ctx context.Context, f field.Model, difficulty int) requests.Request[struct{}] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[struct{}](err)
	}
	url := fmt.Sprintf(root+"worlds/%d/channels/%d/maps/%d/instances/%s/reset",
		f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String())
	return requests.PostRequest[struct{}](url, ResetFieldInputRestModel{Difficulty: difficulty})
}
