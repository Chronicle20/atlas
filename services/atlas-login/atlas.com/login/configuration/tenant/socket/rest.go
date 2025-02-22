package socket

import (
	"atlas-login/configuration/tenant/socket/handler"
	"atlas-login/configuration/tenant/socket/writer"
)

type RestModel struct {
	Handlers []handler.RestModel `json:"handlers"`
	Writers  []writer.RestModel  `json:"writers"`
}
