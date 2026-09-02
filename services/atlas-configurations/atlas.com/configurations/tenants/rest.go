package tenants

import (
	"atlas-configurations/tenants/cashshop"
	"atlas-configurations/tenants/characters"
	"atlas-configurations/tenants/diagnostics"
	"atlas-configurations/tenants/maplelife"
	"atlas-configurations/tenants/npcs"
	"atlas-configurations/tenants/socket"
	"atlas-configurations/tenants/worlds"
)

type RestModel struct {
	Id           string                `json:"-"`
	Region       string                `json:"region"`
	MajorVersion uint16                `json:"majorVersion"`
	MinorVersion uint16                `json:"minorVersion"`
	UsesPin      bool                  `json:"usesPin"`
	Socket       socket.RestModel      `json:"socket"`
	Characters   characters.RestModel  `json:"characters"`
	NPCs         []npcs.RestModel      `json:"npcs"`
	Worlds       []worlds.RestModel    `json:"worlds"`
	CashShop     cashshop.RestModel    `json:"cashShop"`
	MapleLife    maplelife.RestModel   `json:"mapleLife"`
	Diagnostics  diagnostics.RestModel `json:"diagnostics"`
	// Environment is server-owned and read-only (task-232 FR-7.3): it always
	// reflects Entity.Environment, set once by the write path's existing
	// scoping (task-232 D5). Make() overwrites whatever this field held
	// after unmarshaling Entity.Data, so a client-supplied value in a
	// create/update request body is never round-tripped into the column —
	// Create and UpdateById never read RestModel.Environment.
	Environment string `json:"environment"`
}

func (r RestModel) GetName() string {
	return "tenants"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// ResetRestModel is the reset endpoint's request body (FR-4.2): a `tenants`
// resource carrying only the sections to reset. An absent `sections` key
// and `sections: []` both mean "every comparable section" -- resource.go
// additionally normalizes an absent or `{}` body into the equivalent
// envelope before this decodes, so the same "reset everything" default
// applies uniformly.
type ResetRestModel struct {
	Id       string   `json:"-"`
	Sections []string `json:"sections"`
}

func (r ResetRestModel) GetName() string {
	return "tenants"
}

func (r ResetRestModel) GetID() string {
	return r.Id
}

func (r *ResetRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// ViewRestModel is the READ-ONLY projection of a tenant configuration:
// RestModel plus the five computed drift attributes. It is a separate
// type for the same reason templates.ViewRestModel is (see its comment):
// Create persists json.Marshal(input) verbatim, so any field with a JSON
// tag on RestModel would be written INTO the stored document, read back
// by Make, and folded into the next revision -- self-reference and
// permanent phantom drift. Keeping the write model untouched means that
// failure class does not exist rather than being defended against.
//
// encoding/json flattens anonymous embedded structs, so the wire shape is
// exactly RestModel's attributes plus five keys, and sparse fieldsets
// (?fields[tenants]=...) keep working. GetName / GetID / SetID promote
// from the embedded RestModel.
//
// SectionDrift is a map, not a struct: a struct would have to be edited
// every time a section is added, which is the FR-2.7 trap one level up.
// The map is ALWAYS fully populated -- all six keys present, all false
// when no baseline resolved -- so a client never distinguishes "absent"
// from "false".
//
// The PATCH path still binds RestModel, so these five are ignored on
// write by omission rather than by code (FR-3.3).
type ViewRestModel struct {
	RestModel
	BaselineTemplateId string          `json:"baselineTemplateId"`
	BaselineRevision   string          `json:"baselineRevision"`
	StoredRevision     string          `json:"storedRevision"`
	TemplateDrift      bool            `json:"templateDrift"`
	SectionDrift       map[string]bool `json:"sectionDrift"`
}
