package characters

import "atlas-channel/configuration/tenant/characters/template"

type RestModel struct {
	Templates []template.RestModel `json:"templates"`
}
