// Package item is the atlas-data item-information REST client. It answers one
// question — does this item template set WZ `tradeBlock` (FR-4.2)?
//
// atlas-data exposes tradeBlock on FIVE separate resources, one per inventory
// compartment, at five different paths and with five different JSON:API type
// names (equipment -> "statistics", use -> "consumables", setup -> "setups",
// etc -> "etcs", cash -> "cash_items"). There is no unified item resource, so
// this reader carries one RestModel per resource and dispatches on the
// compartment the asset came from.
package item

import (
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

// EquipmentRestModel mirrors atlas-data's GET /data/equipment/{id} response
// (services/atlas-data/atlas.com/data/equipment/rest.go:44,52).
type EquipmentRestModel struct {
	Id         item.Id `json:"-"`
	TradeBlock bool    `json:"tradeBlock"`
}

func (r EquipmentRestModel) GetName() string { return "statistics" }

func (r EquipmentRestModel) GetID() string { return strconv.FormatUint(uint64(r.Id), 10) }

func (r *EquipmentRestModel) SetID(strId string) error { return setItemId(&r.Id, strId) }

func (r *EquipmentRestModel) SetToOneReferenceID(_ string, _ string) error { return nil }

func (r *EquipmentRestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }

func (r EquipmentRestModel) tradeBlock() bool { return r.TradeBlock }

// ConsumableRestModel mirrors atlas-data's GET /data/consumables/{id} response
// (services/atlas-data/atlas.com/data/consumable/rest.go:46,108).
type ConsumableRestModel struct {
	Id         item.Id `json:"-"`
	TradeBlock bool    `json:"tradeBlock"`
}

func (r ConsumableRestModel) GetName() string { return "consumables" }

func (r ConsumableRestModel) GetID() string { return strconv.FormatUint(uint64(r.Id), 10) }

func (r *ConsumableRestModel) SetID(strId string) error { return setItemId(&r.Id, strId) }

func (r *ConsumableRestModel) SetToOneReferenceID(_ string, _ string) error { return nil }

func (r *ConsumableRestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }

func (r ConsumableRestModel) tradeBlock() bool { return r.TradeBlock }

// SetupRestModel mirrors atlas-data's GET /data/setups/{id} response
// (services/atlas-data/atlas.com/data/setup/rest.go:12,25).
type SetupRestModel struct {
	Id         item.Id `json:"-"`
	TradeBlock bool    `json:"tradeBlock"`
}

func (r SetupRestModel) GetName() string { return "setups" }

func (r SetupRestModel) GetID() string { return strconv.FormatUint(uint64(r.Id), 10) }

func (r *SetupRestModel) SetID(strId string) error { return setItemId(&r.Id, strId) }

func (r *SetupRestModel) SetToOneReferenceID(_ string, _ string) error { return nil }

func (r *SetupRestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }

func (r SetupRestModel) tradeBlock() bool { return r.TradeBlock }

// EtcRestModel mirrors atlas-data's GET /data/etcs/{id} response
// (services/atlas-data/atlas.com/data/etc/rest.go:13,18).
type EtcRestModel struct {
	Id         item.Id `json:"-"`
	TradeBlock bool    `json:"tradeBlock"`
}

func (r EtcRestModel) GetName() string { return "etcs" }

func (r EtcRestModel) GetID() string { return strconv.FormatUint(uint64(r.Id), 10) }

func (r *EtcRestModel) SetID(strId string) error { return setItemId(&r.Id, strId) }

func (r *EtcRestModel) SetToOneReferenceID(_ string, _ string) error { return nil }

func (r *EtcRestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }

func (r EtcRestModel) tradeBlock() bool { return r.TradeBlock }

// CashRestModel mirrors atlas-data's GET /data/cash/items/{id} response
// (services/atlas-data/atlas.com/data/cash/rest.go:47,50).
type CashRestModel struct {
	Id         item.Id `json:"-"`
	TradeBlock bool    `json:"tradeBlock"`
}

func (r CashRestModel) GetName() string { return "cash_items" }

func (r CashRestModel) GetID() string { return strconv.FormatUint(uint64(r.Id), 10) }

func (r *CashRestModel) SetID(strId string) error { return setItemId(&r.Id, strId) }

func (r *CashRestModel) SetToOneReferenceID(_ string, _ string) error { return nil }

func (r *CashRestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }

func (r CashRestModel) tradeBlock() bool { return r.TradeBlock }

func setItemId(target *item.Id, strId string) error {
	id, err := strconv.ParseUint(strId, 10, 32)
	if err != nil {
		return err
	}
	*target = item.Id(id)
	return nil
}

// extractTradeBlock is the shared Extract for every one of the five wire
// models: each carries the same single field this reader cares about.
func extractTradeBlock[R interface{ tradeBlock() bool }](rm R) (bool, error) {
	return rm.tradeBlock(), nil
}
