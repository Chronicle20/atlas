package characters

import "atlas-login/configuration/tenant/characters/template"

type RestModel struct {
	Templates []template.RestModel `json:"templates"`
}
