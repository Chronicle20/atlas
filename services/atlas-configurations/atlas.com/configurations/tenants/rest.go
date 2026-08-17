package tenants

import (
	"atlas-configurations/tenants/cashshop"
	"atlas-configurations/tenants/characters"
	"atlas-configurations/tenants/npcs"
	"atlas-configurations/tenants/socket"
	"atlas-configurations/tenants/worlds"
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
