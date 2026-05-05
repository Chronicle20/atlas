package monsterbook

// CollectionRestModel is the JSON:API representation of a character's
// monster book collection returned by atlas-monster-book.
type CollectionRestModel struct {
	Id               uint32 `json:"-"`
	BookLevel        uint16 `json:"bookLevel"`
	NormalCount      uint16 `json:"normalCount"`
	SpecialCount     uint16 `json:"specialCount"`
	TotalUniqueCards uint16 `json:"totalUniqueCards"`
	CoverCardId      uint32 `json:"coverCardId"`
	ExpBonusPercent  uint16 `json:"expBonusPercent"`
}

func (r CollectionRestModel) GetName() string {
	return "monster-book"
}

func (r CollectionRestModel) GetID() string {
	return ""
}

func (r *CollectionRestModel) SetID(_ string) error {
	return nil
}

// Extract converts the wire model into the immutable domain Collection.
func Extract(rm CollectionRestModel) (Collection, error) {
	return Collection{
		bookLevel:        rm.BookLevel,
		normalCount:      rm.NormalCount,
		specialCount:     rm.SpecialCount,
		totalUniqueCards: rm.TotalUniqueCards,
		coverCardId:      rm.CoverCardId,
		expBonusPercent:  rm.ExpBonusPercent,
	}, nil
}
