package inventory

func makeInventory(e entity) (Model, error) {
	var i Model
	//if Type(e.InventoryType) == TypeValueEquip {
	//	i = EquipInventory{
	//		id:       e.ID,
	//		capacity: e.Capacity,
	//	}
	//} else {
	//	i = ItemInventory{
	//		id:            e.ID,
	//		inventoryType: Type(e.InventoryType),
	//		capacity:      e.Capacity,
	//	}
	//}
	return i, nil
}
