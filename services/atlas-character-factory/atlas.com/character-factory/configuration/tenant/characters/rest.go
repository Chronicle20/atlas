package characters

import (
	"atlas-character-factory/configuration/tenant/characters/preset"
	"atlas-character-factory/configuration/tenant/characters/template"
)

type RestModel struct {
	Templates []template.RestModel `json:"templates"`
	Presets   []preset.RestModel   `json:"presets"`
}
