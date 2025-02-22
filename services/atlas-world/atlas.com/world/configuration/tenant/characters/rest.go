package characters

import "atlas-world/configuration/tenant/characters/template"

type RestModel struct {
	Templates []template.RestModel `json:"templates"`
}
