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
