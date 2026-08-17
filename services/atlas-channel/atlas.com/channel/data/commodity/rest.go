package commodity

import (
	"strconv"
)

// RestModel is the atlas-data cash-shop commodity resource
// (services/atlas-data/atlas.com/data/commodity). Id is the commodity SERIAL
// NUMBER — the value ShopOperationBuy* carry as SerialNumber() and
// GW_CashItemInfo carries as CommodityId. ItemId is the item TEMPLATE id the
// serial number resolves to (mirrors atlas-cashshop's own
// cashshop/commodity.Model.ItemId doc comment: "the cash shop commodity
// serial number ... GW_CashItemInfo carries as CommodityId").
type RestModel struct {
	Id     uint32 `json:"-"`
	ItemId uint32 `json:"itemId"`
}

func (r RestModel) GetName() string {
	return "commodities"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}
