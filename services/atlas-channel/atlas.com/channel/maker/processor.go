package maker

import (
	"context"
	"errors"

	"github.com/sirupsen/logrus"
)

// Stable JSON:API error codes atlas-maker's POST /crafts can return
// (services/atlas-maker/atlas.com/maker/craft/errors.go's Code). Kept as a
// channel-side copy of the wire contract -- the channel must not reach
// across the service boundary into atlas-maker's internal craft package.
const (
	CodeRecipeNotFound           = "recipe_not_found"
	CodeLevelTooLow              = "level_too_low"
	CodeSkillLevelTooLow         = "skill_level_too_low"
	CodeInsufficientMaterials    = "insufficient_materials"
	CodeMissingPrerequisiteItem  = "missing_prerequisite_item"
	CodeMissingPrerequisiteQuest = "missing_prerequisite_quest"
	CodeInsufficientMesos        = "insufficient_mesos"
	CodeInventoryFull            = "inventory_full"
	CodeEquipNotFound            = "equip_not_found"
	CodeNoCrystalMapping         = "no_crystal_mapping"
	CodeCraftInProgress          = "craft_in_progress"
	CodeInvalidMode              = "invalid_mode"
	// CodeUnknown is the channel-side fallback for a rejection whose response
	// carried no recognizable `code`, and for a transport failure (atlas-maker
	// unreachable) -- FR-5.2 requires a MAKER_RESULT failure write on both,
	// not just on a well-formed rejection.
	CodeUnknown = "unknown"
)

// Processor is the channel-side atlas-maker craft client.
type Processor interface {
	// Create POSTs req verbatim to atlas-maker for characterId. A non-nil
	// error is either a craftError (CodeOf resolves its PRD §5 code) or a
	// transport failure (CodeOf resolves CodeUnknown).
	Create(characterId uint32, req CraftRequest) (CraftResponse, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *ProcessorImpl {
	return &ProcessorImpl{l: l, ctx: ctx}
}

func (p *ProcessorImpl) Create(characterId uint32, req CraftRequest) (CraftResponse, error) {
	return requestCreateCraft(p.ctx, characterId, req)
}

// CodeOf resolves err's PRD §5 code, defaulting to CodeUnknown for anything
// that did not carry one (a transport error, a malformed error document, or
// a code this list has not been taught yet).
func CodeOf(err error) string {
	var ce craftError
	if errors.As(err, &ce) {
		return ce.code
	}
	return CodeUnknown
}
