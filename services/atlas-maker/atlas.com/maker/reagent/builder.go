package reagent

import (
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

// ValidStats is the closed set of stat names a reagent may carry: the fifteen
// child fields CItemMakerInfo::Load_GemEffect reads under
// Item.wz/Etc/0425.img/<gem>/info, in the order the client reads them. The
// spelling is the archive's, verbatim and case-sensitive.
//
// randOption and randStat are members because the client stores them in the
// same block and the seed must round-trip them; they are the equip
// random-option / random-stat variance keys, NOT additive equip stats. Applying
// them is the caller's concern, not this package's.
var ValidStats = []string{
	"incPAD",
	"incMAD",
	"incACC",
	"incEVA",
	"incSpeed",
	"incJump",
	"incMaxHP",
	"incMaxMP",
	"incSTR",
	"incINT",
	"incLUK",
	"incDEX",
	"incReqLevel",
	"randOption",
	"randStat",
}

// IsValidStat reports whether name is one of ValidStats.
func IsValidStat(name string) bool {
	for _, s := range ValidStats {
		if s == name {
			return true
		}
	}
	return false
}

type Builder struct {
	tenantId      uuid.UUID
	reagentItemId item.Id
	stat          string
	value         int16
}

func NewBuilder(tenantId uuid.UUID, reagentItemId item.Id) *Builder {
	return &Builder{tenantId: tenantId, reagentItemId: reagentItemId}
}

func (b *Builder) SetStat(stat string) *Builder {
	b.stat = stat
	return b
}

func (b *Builder) SetValue(value int16) *Builder {
	b.value = value
	return b
}

func (b *Builder) Build() (Model, error) {
	if b.tenantId == uuid.Nil {
		return Model{}, errors.New("reagent: tenantId cannot be nil")
	}
	if b.reagentItemId == 0 {
		return Model{}, errors.New("reagent: reagentItemId cannot be zero")
	}
	if !IsValidStat(b.stat) {
		return Model{}, fmt.Errorf("reagent: invalid stat %q", b.stat)
	}
	return Model{
		tenantId:      b.tenantId,
		reagentItemId: b.reagentItemId,
		stat:          b.stat,
		value:         b.value,
	}, nil
}
