package monsterbook

import (
	"strconv"

	"github.com/jtumidanski/api2go/jsonapi"
)

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

// GetReferences satisfies the jsonapi.MarshalReferences contract; the resource
// has no related resources.
func (r CollectionRestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{}
}

// GetReferencedIDs satisfies the jsonapi.MarshalLinkedRelations contract; the
// resource has no related resources.
func (r CollectionRestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	return nil
}

// GetReferencedStructs satisfies the jsonapi.MarshalIncludedRelations contract;
// the resource has no included resources.
func (r CollectionRestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	return nil
}

// SetToOneReferenceID satisfies the jsonapi.UnmarshalToOneRelations contract.
// Required for api2go decoding even when no relationships are populated — see
// libs/atlas-rest/CLAUDE.md for the rationale.
func (r *CollectionRestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

// SetToManyReferenceIDs satisfies the jsonapi.UnmarshalToManyRelations contract.
// Required for api2go decoding even when no relationships are populated.
func (r *CollectionRestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

// SetReferencedStructs satisfies the jsonapi.UnmarshalIncludedRelations contract.
func (r *CollectionRestModel) SetReferencedStructs(_ map[string]map[string]jsonapi.Data) error {
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
