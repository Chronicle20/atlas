package characters

import "atlas-configurations/templates/characters/template"

type RestModel struct {
	Templates []template.RestModel `json:"templates"`
}
