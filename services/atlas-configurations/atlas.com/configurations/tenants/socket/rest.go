package socket

import (
	"atlas-configurations/tenants/socket/handler"
	"atlas-configurations/tenants/socket/writer"
)

type RestModel struct {
	Handlers []handler.RestModel `json:"handlers"`
	Writers  []writer.RestModel  `json:"writers"`
}
