// Package tradeability is atlas-inventory's read client for the two atlas-data
// item properties the karma gates need: WZ info/tradeBlock (is the item
// untradeable by data?) and WZ info/tradeAvailable (the item's applicable karma
// type). atlas-data exposes them on FIVE separate resources, one per inventory
// compartment, at five paths with five JSON:API type names — there is no
// unified item resource — so this reader carries one RestModel per resource and
// dispatches on the compartment the asset came from. Modelled on
// services/atlas-trades/atlas.com/trades/data/item.
package tradeability

import (
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

type Model struct {
	tradeBlock     bool
	tradeAvailable int32
}

func (m Model) TradeBlock() bool      { return m.tradeBlock }
func (m Model) TradeAvailable() int32 { return m.tradeAvailable }

// NewModel builds a Model from outside the package. It exists for test
// callers (Task 7, Task 10) that need to construct a non-zero tradeability
// Model without going through a live atlas-data request.
func NewModel(tradeBlock bool, tradeAvailable int32) Model {
	return Model{tradeBlock: tradeBlock, tradeAvailable: tradeAvailable}
}

type EquipmentRestModel struct {
	Id             item.Id `json:"-"`
	TradeBlock     bool    `json:"tradeBlock"`
	TradeAvailable int32   `json:"tradeAvailable"`
}

func (r EquipmentRestModel) GetName() string { return "statistics" }

func (r EquipmentRestModel) GetID() string { return strconv.FormatUint(uint64(r.Id), 10) }

func (r *EquipmentRestModel) SetID(s string) error                             { return setItemId(&r.Id, s) }
func (r *EquipmentRestModel) SetToOneReferenceID(_ string, _ string) error     { return nil }
func (r *EquipmentRestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }
func (r EquipmentRestModel) fields() (bool, int32)                             { return r.TradeBlock, r.TradeAvailable }

func (r *EquipmentRestModel) setFields(tb bool, ta int32) { r.TradeBlock, r.TradeAvailable = tb, ta }

type ConsumableRestModel struct {
	Id             item.Id `json:"-"`
	TradeBlock     bool    `json:"tradeBlock"`
	TradeAvailable int32   `json:"tradeAvailable"`
}

func (r ConsumableRestModel) GetName() string { return "consumables" }

func (r ConsumableRestModel) GetID() string { return strconv.FormatUint(uint64(r.Id), 10) }

func (r *ConsumableRestModel) SetID(s string) error                             { return setItemId(&r.Id, s) }
func (r *ConsumableRestModel) SetToOneReferenceID(_ string, _ string) error     { return nil }
func (r *ConsumableRestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }
func (r ConsumableRestModel) fields() (bool, int32)                             { return r.TradeBlock, r.TradeAvailable }

func (r *ConsumableRestModel) setFields(tb bool, ta int32) { r.TradeBlock, r.TradeAvailable = tb, ta }

type SetupRestModel struct {
	Id             item.Id `json:"-"`
	TradeBlock     bool    `json:"tradeBlock"`
	TradeAvailable int32   `json:"tradeAvailable"`
}

func (r SetupRestModel) GetName() string { return "setups" }
func (r SetupRestModel) GetID() string   { return strconv.FormatUint(uint64(r.Id), 10) }

func (r *SetupRestModel) SetID(s string) error                             { return setItemId(&r.Id, s) }
func (r *SetupRestModel) SetToOneReferenceID(_ string, _ string) error     { return nil }
func (r *SetupRestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }
func (r SetupRestModel) fields() (bool, int32)                             { return r.TradeBlock, r.TradeAvailable }

func (r *SetupRestModel) setFields(tb bool, ta int32) { r.TradeBlock, r.TradeAvailable = tb, ta }

type EtcRestModel struct {
	Id             item.Id `json:"-"`
	TradeBlock     bool    `json:"tradeBlock"`
	TradeAvailable int32   `json:"tradeAvailable"`
}

func (r EtcRestModel) GetName() string { return "etcs" }
func (r EtcRestModel) GetID() string   { return strconv.FormatUint(uint64(r.Id), 10) }

func (r *EtcRestModel) SetID(s string) error                             { return setItemId(&r.Id, s) }
func (r *EtcRestModel) SetToOneReferenceID(_ string, _ string) error     { return nil }
func (r *EtcRestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }
func (r EtcRestModel) fields() (bool, int32)                             { return r.TradeBlock, r.TradeAvailable }

func (r *EtcRestModel) setFields(tb bool, ta int32) { r.TradeBlock, r.TradeAvailable = tb, ta }

type CashRestModel struct {
	Id             item.Id `json:"-"`
	TradeBlock     bool    `json:"tradeBlock"`
	TradeAvailable int32   `json:"tradeAvailable"`
}

func (r CashRestModel) GetName() string { return "cash_items" }
func (r CashRestModel) GetID() string   { return strconv.FormatUint(uint64(r.Id), 10) }

func (r *CashRestModel) SetID(s string) error                             { return setItemId(&r.Id, s) }
func (r *CashRestModel) SetToOneReferenceID(_ string, _ string) error     { return nil }
func (r *CashRestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }
func (r CashRestModel) fields() (bool, int32)                             { return r.TradeBlock, r.TradeAvailable }

func (r *CashRestModel) setFields(tb bool, ta int32) { r.TradeBlock, r.TradeAvailable = tb, ta }

func setItemId(target *item.Id, strId string) error {
	id, err := strconv.ParseUint(strId, 10, 32)
	if err != nil {
		return err
	}
	*target = item.Id(id)
	return nil
}

// extract is the shared Extract for all five wire models.
func extract[R interface{ fields() (bool, int32) }](rm R) (Model, error) {
	tb, ta := rm.fields()
	return Model{tradeBlock: tb, tradeAvailable: ta}, nil
}

// transform is the shared Transform for all five wire models, the inverse of
// extract. PR is the pointer type of R so transform can populate a zero-value
// R without a per-type composite literal.
func transform[R any, PR interface {
	*R
	setFields(bool, int32)
}](m Model) R {
	var r R
	PR(&r).setFields(m.tradeBlock, m.tradeAvailable)
	return r
}

func TransformEquipment(m Model) (EquipmentRestModel, error) {
	return transform[EquipmentRestModel](m), nil
}

func TransformConsumable(m Model) (ConsumableRestModel, error) {
	return transform[ConsumableRestModel](m), nil
}

func TransformSetup(m Model) (SetupRestModel, error) {
	return transform[SetupRestModel](m), nil
}

func TransformEtc(m Model) (EtcRestModel, error) {
	return transform[EtcRestModel](m), nil
}

func TransformCash(m Model) (CashRestModel, error) {
	return transform[CashRestModel](m), nil
}
