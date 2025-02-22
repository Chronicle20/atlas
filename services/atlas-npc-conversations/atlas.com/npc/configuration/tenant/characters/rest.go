package characters

import "atlas-npc-conversations/configuration/tenant/characters/template"

type RestModel struct {
	Templates []template.RestModel `json:"templates"`
}
