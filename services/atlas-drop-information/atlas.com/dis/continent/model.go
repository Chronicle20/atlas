package continent

import (
	"atlas-drops-information/continent/drop"
)

type Model struct {
	id    int32
	drops []drop.Model
}
