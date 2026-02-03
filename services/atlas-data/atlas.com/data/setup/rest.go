package setup

import (
	"strconv"
)

type RestModel struct {
	Id             uint32 `json:"-"`
	Price          uint32 `json:"price"`
	SlotMax        uint32 `json:"slotMax"`
	RecoveryHP     uint32 `json:"recoveryHP"`
	TradeBlock     bool   `json:"tradeBlock"`
	NotSale        bool   `json:"notSale"`
	ReqLevel       uint32 `json:"reqLevel"`
	DistanceX      uint32 `json:"distanceX"`
	DistanceY      uint32 `json:"distanceY"`
	MaxDiff        uint32 `json:"maxDiff"`
	Direction      uint32 `json:"direction"`
	TimeLimited    bool   `json:"timeLimited"`
	ReplaceItemId  uint32 `json:"replaceItemId,omitempty"`
	ReplaceMessage string `json:"replaceMessage,omitempty"`
}

func (r RestModel) GetName() string {
	return "setups"
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