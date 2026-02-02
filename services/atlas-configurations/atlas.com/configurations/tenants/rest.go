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
