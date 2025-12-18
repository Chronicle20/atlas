package socket

import (
	"atlas-channel/configuration/tenant/socket/handler"
	"atlas-channel/configuration/tenant/socket/writer"
)

type RestModel struct {
	Handlers []handler.RestModel `json:"handlers"`
	Writers  []writer.RestModel  `json:"writers"`
}
