package templates

import (
	"atlas-configurations/templates/cashshop"
	"atlas-configurations/templates/characters"
	"atlas-configurations/templates/npcs"
	"atlas-configurations/templates/socket"
	"atlas-configurations/templates/worlds"
)

type RestModel struct {
	Id           string               `json:"-"`
	Region       string               `json:"region"`
	MajorVersion uint16               `json:"majorVersion"`
	MinorVersion uint16               `json:"minorVersion"`
	UsesPin      bool                 `json:"usesPin"`
	Socket       socket.RestModel     `json:"socket"`
	Characters   characters.RestModel `json:"characters"`
	NPCs         []npcs.RestModel     `json:"npcs"`
	Worlds       []worlds.RestModel   `json:"worlds"`
	CashShop     cashshop.RestModel   `json:"cashShop"`
}

func (r RestModel) GetName() string {
	return "templates"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// ViewRestModel is the READ-ONLY projection of a template: RestModel plus the
// three computed drift attributes. It is a separate type on purpose (design
// D3): Create persists json.Marshal(input) verbatim, so any field with a JSON
// tag on RestModel would be written INTO the stored document, read back by
// Make, and folded into the next revision - self-reference and permanent
// phantom drift. Keeping the write model untouched means that failure class
// does not exist rather than being defended against.
//
// encoding/json flattens anonymous embedded structs, and api2go builds the
// attributes object with a plain json.Marshal, so the wire shape is exactly
// RestModel's attributes plus three keys. GetName / GetID / SetID promote from
// the embedded RestModel.
//
// The PATCH path still binds RestModel, so the three attributes are ignored on
// write by omission rather than by code (PRD §5.1).
type ViewRestModel struct {
	RestModel
	ShippedRevision string `json:"shippedRevision"`
	StoredRevision  string `json:"storedRevision"`
	SeedDrift       bool   `json:"seedDrift"`
}
