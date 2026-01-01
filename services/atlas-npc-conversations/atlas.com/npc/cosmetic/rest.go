package cosmetic

import (
	"github.com/jtumidanski/api2go/jsonapi"
	"strconv"
)

// RestCharacterModel represents the REST API response for character data
// This matches the structure from atlas-query-aggregator's character REST model
type RestCharacterModel struct {
	Id        uint32 `json:"-"`
	Gender    byte   `json:"gender"`
	SkinColor byte   `json:"skinColor"`
	Face      uint32 `json:"face"`
	Hair      uint32 `json:"hair"`
}

func (r RestCharacterModel) GetName() string {
	return "characters"
}

func (r RestCharacterModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestCharacterModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func (r RestCharacterModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{}
}

func (r RestCharacterModel) GetReferencedIDs() []jsonapi.ReferenceID {
	return []jsonapi.ReferenceID{}
}

func (r RestCharacterModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	return []jsonapi.MarshalIdentifier{}
}

func (r *RestCharacterModel) SetToOneReferenceID(name, ID string) error {
	return nil
}

func (r *RestCharacterModel) SetToManyReferenceIDs(name string, IDs []string) error {
	return nil
}

func (r *RestCharacterModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	return nil
}

// ExtractAppearance converts a REST model to a CharacterAppearance
func ExtractAppearance(r RestCharacterModel) (CharacterAppearance, error) {
	return NewCharacterAppearance(r.Id, r.Gender, r.Hair, r.Face, r.SkinColor), nil
}

// CharacterUpdateRequest represents a partial update request for character appearance
// Uses pointers to allow null values for fields that shouldn't be updated
type CharacterUpdateRequest struct {
	Hair      *uint32 `json:"hair,omitempty"`
	Face      *uint32 `json:"face,omitempty"`
	SkinColor *byte   `json:"skinColor,omitempty"`
	Gender    *byte   `json:"gender,omitempty"`
}

// NewHairUpdateRequest creates a request to update hair only
func NewHairUpdateRequest(hair uint32) CharacterUpdateRequest {
	return CharacterUpdateRequest{
		Hair: &hair,
	}
}

// NewFaceUpdateRequest creates a request to update face only
func NewFaceUpdateRequest(face uint32) CharacterUpdateRequest {
	return CharacterUpdateRequest{
		Face: &face,
	}
}

// NewSkinColorUpdateRequest creates a request to update skin color only
func NewSkinColorUpdateRequest(skinColor byte) CharacterUpdateRequest {
	return CharacterUpdateRequest{
		SkinColor: &skinColor,
	}
}
