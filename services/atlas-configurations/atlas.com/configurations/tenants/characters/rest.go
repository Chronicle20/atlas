package characters

import (
	"atlas-configurations/templates/characters/template"
	"atlas-configurations/tenants/characters/preset"
)

type RestModel struct {
	Templates []template.RestModel `json:"templates"`
	Presets   []preset.RestModel   `json:"presets"`
}
