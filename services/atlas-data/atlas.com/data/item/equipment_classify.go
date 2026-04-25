package item

import (
	"context"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"gorm.io/gorm"
)

// UpdateEquipmentClassification fills job_mask and overrides subcategory (when
// the WZ slot disambiguates a classification collision) for an existing
// equipment row in the search index. The caller-supplied tx is the same
// transaction equipment.Register uses, so this UPDATE participates in any
// rollback.
func UpdateEquipmentClassification(tx *gorm.DB, ctx context.Context, itemId uint32, slotWZ string, reqJob uint16) error {
	t := tenant.MustFromContext(ctx)
	mask := uint8(reqJob & 0x1F)

	updates := map[string]interface{}{
		"job_mask": mask,
	}
	if sub, ok := disambiguateSlotSubcategory(itemId, slotWZ); ok {
		updates["subcategory"] = sub
	}

	return tx.WithContext(ctx).
		Table("item_string_search_index").
		Where("tenant_id = ? AND item_id = ?", t.Id(), itemId).
		Updates(updates).Error
}

// disambiguateSlotSubcategory returns the slot-derived subcategory when the
// item's classification cannot decide on its own. Today only classification
// 104 (earring vs top) needs slot input.
func disambiguateSlotSubcategory(itemId uint32, slotWZ string) (string, bool) {
	if classification(itemId) != 104 {
		return "", false
	}
	switch slotWZ {
	case "Cp", "Vs":
		return "top", true
	case "Ae":
		return "earring", true
	default:
		return "", false
	}
}
