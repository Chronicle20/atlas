package craft

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
)

// Code is a PRD §5 stable JSON:API error code.
type Code string

const (
	CodeRecipeNotFound           Code = "recipe_not_found"
	CodeLevelTooLow              Code = "level_too_low"
	CodeSkillLevelTooLow         Code = "skill_level_too_low"
	CodeInsufficientMaterials    Code = "insufficient_materials"
	CodeMissingPrerequisiteItem  Code = "missing_prerequisite_item"
	CodeMissingPrerequisiteQuest Code = "missing_prerequisite_quest"
	CodeInsufficientMesos        Code = "insufficient_mesos"
	CodeInventoryFull            Code = "inventory_full"
	CodeEquipNotFound            Code = "equip_not_found"
	CodeNoCrystalMapping         Code = "no_crystal_mapping"
	CodeCraftInProgress          Code = "craft_in_progress"
	CodeInvalidMode              Code = "invalid_mode"
)

// CraftError is Processor.Create's rejection type. Code maps 1:1 onto PRD
// §5's stable JSON:API codes (missing_prerequisite_quest from design C-5,
// craft_in_progress from design §7); Status is the HTTP status Task 24's
// resource layer returns for it.
type CraftError struct {
	Code   Code
	Status int
}

func (e CraftError) Error() string {
	return string(e.Code)
}

// ErrRecipeNotFound and ErrCraftInProgress are the two rejections not
// derived from an Eligibility.Reason.
var (
	ErrRecipeNotFound  = CraftError{Code: CodeRecipeNotFound, Status: http.StatusNotFound}
	ErrCraftInProgress = CraftError{Code: CodeCraftInProgress, Status: http.StatusConflict}
)

// reasonToCraftError maps an Eligibility.Reason onto its PRD §5 code. Every
// Reason constant eligibility.go declares has an entry; there is no other
// caller of Reason that could introduce an unmapped one.
func reasonToCraftError(reason Reason) CraftError {
	switch reason {
	case ReasonLevelTooLow:
		return CraftError{Code: CodeLevelTooLow, Status: http.StatusUnprocessableEntity}
	case ReasonSkillLevelTooLow:
		return CraftError{Code: CodeSkillLevelTooLow, Status: http.StatusUnprocessableEntity}
	case ReasonInsufficientMaterials:
		return CraftError{Code: CodeInsufficientMaterials, Status: http.StatusUnprocessableEntity}
	case ReasonMissingPrerequisiteItem:
		return CraftError{Code: CodeMissingPrerequisiteItem, Status: http.StatusUnprocessableEntity}
	case ReasonMissingPrerequisiteQuest:
		return CraftError{Code: CodeMissingPrerequisiteQuest, Status: http.StatusUnprocessableEntity}
	case ReasonInsufficientMesos:
		return CraftError{Code: CodeInsufficientMesos, Status: http.StatusUnprocessableEntity}
	case ReasonInventoryFull:
		return CraftError{Code: CodeInventoryFull, Status: http.StatusUnprocessableEntity}
	default:
		return CraftError{Code: CodeInvalidMode, Status: http.StatusUnprocessableEntity}
	}
}

// errorDocument is a minimal JSON:API error document carrying a stable
// `code` alongside the standard `status`/`title` (libs/atlas-rest/server's
// own WriteErrorResponse has no code field, so Task 24's resource layer
// writes its own here rather than extending a shared library for one
// package's need).
type errorDocument struct {
	Errors []jsonapi.Error `json:"errors"`
}

// writeCraftError writes ce as a single JSON:API error, using ce.Status and
// ce.Code as PRD §5 specifies. Task 24's resource handlers use it for every
// CraftError rejection; a non-CraftError from a Processor call still falls
// through to server.WriteErrorResponse (500, or 503 if transient).
func writeCraftError(l logrus.FieldLogger, w http.ResponseWriter, ce CraftError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(ce.Status)
	doc := errorDocument{Errors: []jsonapi.Error{{
		Status: strconv.Itoa(ce.Status),
		Code:   string(ce.Code),
		Title:  http.StatusText(ce.Status),
	}}}
	if err := json.NewEncoder(w).Encode(doc); err != nil {
		l.WithError(err).Errorf("Encoding craft error response body.")
	}
}
