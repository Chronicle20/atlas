package socket

import (
	"atlas-world/configuration/tenant/socket/handler"
	"atlas-world/configuration/tenant/socket/writer"
)

type RestModel struct {
	Handlers []handler.RestModel `json:"handlers"`
	Writers  []writer.RestModel  `json:"writers"`
}
