package monsterbook

import (
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/jtumidanski/api2go/jsonapi"
)

// CollectionRestModel is the JSON:API representation of a character's
// monster book collection returned by atlas-monster-book.
type CollectionRestModel struct {
	Id               uint32  `json:"-"`
	BookLevel        uint16  `json:"bookLevel"`
	NormalCount      uint16  `json:"normalCount"`
	SpecialCount     uint16  `json:"specialCount"`
	TotalUniqueCards uint16  `json:"totalUniqueCards"`
	CoverCardId      item.Id `json:"coverCardId"`
	ExpBonusPercent  uint16  `json:"expBonusPercent"`
}

func (r CollectionRestModel) GetName() string {
	return "monster-book"
}

func (r CollectionRestModel) GetID() string {
	if r.Id == 0 {
		return ""
	}
	return strconv.FormatUint(uint64(r.Id), 10)
}

func (r *CollectionRestModel) SetID(strId string) error {
	if strId == "" {
		r.Id = 0
		return nil
	}
	id, err := strconv.ParseUint(strId, 10, 32)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

// GetReferences implements jsonapi.MarshalReferences. The monster-book
// resource has no relationships; an empty list satisfies the interface.
func (r CollectionRestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{}
}

// GetReferencedIDs implements jsonapi.MarshalLinkedRelations.
func (r CollectionRestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	return []jsonapi.ReferenceID{}
}

// SetToOneReferenceID implements jsonapi.UnmarshalToOneRelations.
func (r *CollectionRestModel) SetToOneReferenceID(_ string, _ string) error {
	return nil
}

// SetToManyReferenceIDs implements jsonapi.UnmarshalToManyRelations.
func (r *CollectionRestModel) SetToManyReferenceIDs(_ string, _ []string) error {
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
