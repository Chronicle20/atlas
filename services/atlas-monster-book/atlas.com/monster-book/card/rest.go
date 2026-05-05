package card

import (
	"strconv"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

type RestModel struct {
	CardId          item.Id   `json:"-"`
	Level           uint8     `json:"level"`
	IsSpecial       bool      `json:"isSpecial"`
	FirstAcquiredAt time.Time `json:"firstAcquiredAt"`
}

func (r RestModel) GetName() string { return "monster-book-card" }
func (r RestModel) GetID() string   { return strconv.FormatUint(uint64(r.CardId), 10) }
func (r *RestModel) SetID(id string) error {
	v, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return err
	}
	r.CardId = item.Id(v)
	return nil
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		CardId:          m.CardId(),
		Level:           m.Level(),
		IsSpecial:       m.IsSpecial(),
		FirstAcquiredAt: m.FirstAcquiredAt(),
	}, nil
}
