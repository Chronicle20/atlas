package socket

import "github.com/Chronicle20/atlas/libs/atlas-opcodes"

type RestModel struct {
	Handlers []opcodes.HandlerConfig `json:"handlers"`
	Writers  []opcodes.WriterConfig  `json:"writers"`
}
