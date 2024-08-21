package slottable

import (
	"atlas-character/asset"
	"github.com/Chronicle20/atlas-model/model"
	"sort"
)

func MinFreeSlot(items []asset.Slottable) int16 {
	slot := int16(1)
	i := 0

	for {
		if i >= len(items) {
			return slot
		} else if slot < items[i].Slot() {
			return slot
		} else if slot == items[i].Slot() {
			slot += 1
			i += 1
		} else if items[i].Slot() <= 0 {
			i += 1
		}
	}
}

func GetNextFreeSlot(provider model.Provider[[]asset.Slottable]) (int16, error) {
	es, err := provider()
	if err != nil {
		return 1, err
	}
	if len(es) == 0 {
		return 1, nil
	}

	sort.Slice(es, func(i, j int) bool {
		return es[i].Slot() < es[j].Slot()
	})
	return MinFreeSlot(es), nil
}
