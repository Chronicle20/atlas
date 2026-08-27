package setup

import (
	"strconv"
)

type RestModel struct {
	Id         uint32 `json:"-"`
	Price      uint32 `json:"price"`
	SlotMax    uint32 `json:"slotMax"`
	RecoveryHP uint32 `json:"recoveryHP"`
	TradeBlock bool   `json:"tradeBlock"`
	NotSale    bool   `json:"notSale"`
	ReqLevel   uint32 `json:"reqLevel"`
	DistanceX  uint32 `json:"distanceX"`
	DistanceY  uint32 `json:"distanceY"`
	MaxDiff    uint32 `json:"maxDiff"`
	Direction  uint32 `json:"direction"`
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

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:         m.id,
		Price:      m.price,
		SlotMax:    m.slotMax,
		RecoveryHP: m.recoveryHP,
		TradeBlock: m.tradeBlock,
		NotSale:    m.notSale,
		ReqLevel:   m.reqLevel,
		DistanceX:  m.distanceX,
		DistanceY:  m.distanceY,
		MaxDiff:    m.maxDiff,
		Direction:  m.direction,
	}, nil
}
