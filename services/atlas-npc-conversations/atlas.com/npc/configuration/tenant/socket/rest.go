package socket

import (
	"atlas-npc-conversations/configuration/tenant/socket/handler"
	"atlas-npc-conversations/configuration/tenant/socket/writer"
)

type RestModel struct {
	Handlers []handler.RestModel `json:"handlers"`
	Writers  []writer.RestModel  `json:"writers"`
}
