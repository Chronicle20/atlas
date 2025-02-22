package socket

import (
	"atlas-configurations/templates/socket/handler"
	"atlas-configurations/templates/socket/writer"
)

type RestModel struct {
	Handlers []handler.RestModel `json:"handlers"`
	Writers  []writer.RestModel  `json:"writers"`
}
