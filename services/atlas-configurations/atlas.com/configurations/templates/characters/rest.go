package characters

import (
	"atlas-configurations/templates/characters/preset"
	"atlas-configurations/templates/characters/template"
)

type RestModel struct {
	Templates []template.RestModel `json:"templates"`
	Presets   []preset.RestModel   `json:"presets"`
}
