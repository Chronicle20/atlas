package monsterbook

import "strconv"

// CollectionRestModel mirrors the wire shape returned by atlas-monster-book at
// GET /characters/{characterId}/monster-book. The query-aggregator only needs
// totalUniqueCards for the monsterBookCount validation condition, but the full
// shape is captured so future conditions can read additional fields without
// redefining the wire model.
type CollectionRestModel struct {
	Id               uint32 `json:"-"`
	BookLevel        uint16 `json:"bookLevel"`
	NormalCount      uint16 `json:"normalCount"`
	SpecialCount     uint16 `json:"specialCount"`
	TotalUniqueCards uint16 `json:"totalUniqueCards"`
	CoverCardId      uint32 `json:"coverCardId"`
	ExpBonusPercent  uint16 `json:"expBonusPercent"`
}

// GetName returns the JSON:API resource name used by atlas-monster-book.
func (r CollectionRestModel) GetName() string { return "monster-book" }

// GetID returns the resource ID. atlas-monster-book always returns the
// character ID here, so we surface it for clients that care.
func (r CollectionRestModel) GetID() string {
	return strconv.FormatUint(uint64(r.Id), 10)
}

// SetID parses the JSON:API id field back into the character id.
func (r *CollectionRestModel) SetID(id string) error {
	if id == "" {
		return nil
	}
	v, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return err
	}
	r.Id = uint32(v)
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
