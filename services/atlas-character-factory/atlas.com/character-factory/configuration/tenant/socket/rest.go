package socket

import (
	"atlas-character-factory/configuration/tenant/socket/handler"
	"atlas-character-factory/configuration/tenant/socket/writer"
)

type RestModel struct {
	Handlers []handler.RestModel `json:"handlers"`
	Writers  []writer.RestModel  `json:"writers"`
}
