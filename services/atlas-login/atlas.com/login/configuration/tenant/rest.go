package tenant

import (
	"atlas-login/configuration/tenant/characters"
	"atlas-login/configuration/tenant/npcs"
	"atlas-login/configuration/tenant/socket"
	"atlas-login/configuration/tenant/worlds"
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
