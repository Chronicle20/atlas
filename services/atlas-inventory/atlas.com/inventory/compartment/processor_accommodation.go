package compartment

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

// AccommodationRequest is one item a caller wants to know it could grant.
type AccommodationRequest struct {
	TemplateId uint32
	Quantity   uint32
}

// AccommodationResult reports, per requested item, whether a CreateAsset of it
// would currently succeed.
type AccommodationResult struct {
	TemplateId   uint32
	Quantity     uint32
	Accommodated bool
}

// freeSlots counts unoccupied positive slots in a compartment. Equipped items
// live in negative slots and never consume inventory space, so they are ignored.
func freeSlots(c Model) uint32 {
	var occupied uint32
	for _, a := range c.Assets() {
		if a.Slot() >= 1 {
			occupied++
		}
	}
	if occupied >= c.Capacity() {
		return 0
	}
	return c.Capacity() - occupied
}

// CanAccommodate reports, for each requested item independently, whether a
// CreateAsset of that item would currently succeed for the character. Each item
// is evaluated against the same live inventory snapshot (not cumulatively), which
// is exactly what a single random reward grant needs: the caller does not know
// which one item will be granted, so it asks whether every possible one fits.
//
// The verdict mirrors CreateAsset's own allocation: a create succeeds if the
// target compartment has a free slot (merge-or-new-slot always works), OR — when
// the compartment is full — the item is a mergeable stackable whose quantity
// fully fits into an existing stack of the same template. Equipment and
// rechargeables (bullets/throwing stars) never merge, matching CreateAsset's
// merge guard.
func (p *ProcessorImpl) CanAccommodate(characterId uint32, reqs []AccommodationRequest) ([]AccommodationResult, error) {
	compartments, err := p.GetByCharacterId(characterId)
	if err != nil {
		return nil, err
	}
	byType := make(map[inventory.Type]Model, len(compartments))
	for _, c := range compartments {
		byType[c.Type()] = c
	}

	results := make([]AccommodationResult, 0, len(reqs))
	for _, req := range reqs {
		ok, aErr := p.accommodatesOne(byType, req.TemplateId, req.Quantity)
		if aErr != nil {
			return nil, aErr
		}
		results = append(results, AccommodationResult{TemplateId: req.TemplateId, Quantity: req.Quantity, Accommodated: ok})
	}
	return results, nil
}

// accommodatesOne is the per-item verdict. GetSlotMax is only consulted when the
// compartment is full and the item could still merge, so the common (has-a-free-slot)
// case costs no item-data lookup.
func (p *ProcessorImpl) accommodatesOne(byType map[inventory.Type]Model, templateId uint32, quantity uint32) (bool, error) {
	it, ok := inventory.TypeFromItemId(item.Id(templateId))
	if !ok {
		return false, nil
	}
	c, present := byType[it]
	if !present {
		return false, nil
	}
	if freeSlots(c) >= 1 {
		return true, nil
	}
	// Compartment full: only a full merge into an existing stack can succeed, and
	// only for mergeable stackables.
	if it == inventory.TypeValueEquip || item.IsBullet(item.Id(templateId)) || item.IsThrowingStar(item.Id(templateId)) {
		return false, nil
	}
	slotMax, err := p.assetProcessor.GetSlotMax(templateId)
	if err != nil {
		return false, err
	}
	for _, a := range c.Assets() {
		if a.TemplateId() == templateId && a.HasQuantity() && a.Quantity() < slotMax && a.Quantity()+quantity <= slotMax {
			return true, nil
		}
	}
	return false, nil
}
