package monster

import (
	"atlas-dis/monster/drop"
)

type Model struct {
	id    uint32
	drops []drop.Model
}
