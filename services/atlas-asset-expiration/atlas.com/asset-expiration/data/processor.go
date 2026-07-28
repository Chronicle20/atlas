package data

import (
	"context"

	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetReplaceInfo(templateId uint32) ReplaceInfo
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

// GetReplaceInfo retrieves replacement information for a template ID
// Returns ReplaceInfo with the replacement item ID and message if applicable
func (p *ProcessorImpl) GetReplaceInfo(templateId uint32) ReplaceInfo {
	// Determine item type from template ID
	// Equipment: 1000000-1999999
	// Consumables: 2000000-2999999
	// Setup: 3000000-3999999
	// Etc: 4000000-4999999
	// Cash: 5000000+

	if templateId >= 1000000 && templateId < 2000000 {
		eq, err := requestEquipment(templateId)(p.l, p.ctx)
		if err != nil {
			p.l.WithError(err).Warnf("Failed to get equipment data for template [%d].", templateId)
			return ReplaceInfo{}
		}
		return ReplaceInfo{
			ReplaceItemId:  eq.ReplaceItemId,
			ReplaceMessage: eq.ReplaceMessage,
		}
	}

	if templateId >= 2000000 && templateId < 3000000 {
		con, err := requestConsumable(templateId)(p.l, p.ctx)
		if err != nil {
			p.l.WithError(err).Warnf("Failed to get consumable data for template [%d].", templateId)
			return ReplaceInfo{}
		}
		return ReplaceInfo{
			ReplaceItemId:  con.ReplaceItemId,
			ReplaceMessage: con.ReplaceMessage,
		}
	}

	if templateId >= 3000000 && templateId < 4000000 {
		setup, err := requestSetup(templateId)(p.l, p.ctx)
		if err != nil {
			p.l.WithError(err).Warnf("Failed to get setup data for template [%d].", templateId)
			return ReplaceInfo{}
		}
		return ReplaceInfo{
			ReplaceItemId:  setup.ReplaceItemId,
			ReplaceMessage: setup.ReplaceMessage,
		}
	}

	if templateId >= 4000000 && templateId < 5000000 {
		etc, err := requestEtc(templateId)(p.l, p.ctx)
		if err != nil {
			p.l.WithError(err).Warnf("Failed to get etc data for template [%d].", templateId)
			return ReplaceInfo{}
		}
		return ReplaceInfo{
			ReplaceItemId:  etc.ReplaceItemId,
			ReplaceMessage: etc.ReplaceMessage,
		}
	}

	// Cash items and unknown ranges - no replacement info available
	return ReplaceInfo{}
}
