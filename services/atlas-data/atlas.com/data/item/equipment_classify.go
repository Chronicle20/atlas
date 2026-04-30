package item

import (
	"context"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"gorm.io/gorm"
)

// UpdateEquipmentClassification fills job_mask for an existing equipment row
// in the search index, and reassigns NX cosmetic equipment to the Cash
// compartment. The caller-supplied tx is the same transaction
// equipment.Register uses, so this UPDATE participates in any rollback.
//
// When isCash is true (NX cosmetic equipment) the row is reassigned to the
// Cash compartment so it doesn't surface under Equipment filters — its id
// prefix lies in the equipment range but it functionally belongs to Cash.
//
// Precondition: the StringStorage.Add pass already wrote the search-index row
// for this item; if it didn't, GORM's Updates is a silent no-op (RowsAffected=0)
// rather than an error.
func UpdateEquipmentClassification(tx *gorm.DB, ctx context.Context, itemId uint32, reqJob uint16, isCash bool) error {
	t := tenant.MustFromContext(ctx)
	// Bits 0..4 = Warrior/Magician/Bowman/Thief/Pirate per PRD §4.3.
	mask := uint8(reqJob & 0x1F)

	updates := map[string]interface{}{
		"job_mask": mask,
	}
	if isCash {
		updates["compartment"] = uint8(CompartmentCash)
	}

	return tx.WithContext(ctx).
		Table("item_string_search_index").
		Where("tenant_id = ? AND item_id = ?", t.Id(), itemId).
		Updates(updates).Error
}
