package socket

import (
	"atlas-configurations/templates/socket/handler"
	"atlas-configurations/templates/socket/writer"
)

type RestModel struct {
	Handlers    []handler.RestModel  `json:"handlers"`
	Writers     []writer.RestModel   `json:"writers"`
	Unsupported UnsupportedRestModel `json:"unsupported"`
}

// UnsupportedRestModel names the definitions that have been audited and
// confirmed absent for this Region/Version. It is what makes "audited, this
// version does not have this packet" distinguishable from "nobody has looked
// yet" (PRD FR-1.x). Names here are implementation names, never opcodes.
type UnsupportedRestModel struct {
	Handlers []string `json:"handlers"`
	Writers  []string `json:"writers"`
}

// Normalize replaces nil slices with empty ones so the marshalled document
// always carries real arrays rather than nulls. Entries themselves are left
// untouched. Callers must funnel every read path (Make) and every write path
// (Create/UpdateById) through Normalize; that is what guarantees the
// invariant. As of this change that wiring has not happened yet - Make,
// Create and UpdateById still do plain json.Unmarshal/json.Marshal with no
// call to Normalize, so a nil Handlers/Writers/Unsupported.* still marshals
// as null today. Wiring the processors is a separate, later change.
func Normalize(rm RestModel) RestModel {
	if rm.Handlers == nil {
		rm.Handlers = []handler.RestModel{}
	}
	if rm.Writers == nil {
		rm.Writers = []writer.RestModel{}
	}
	if rm.Unsupported.Handlers == nil {
		rm.Unsupported.Handlers = []string{}
	}
	if rm.Unsupported.Writers == nil {
		rm.Unsupported.Writers = []string{}
	}
	return rm
}
