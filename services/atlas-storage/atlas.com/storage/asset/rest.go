package asset

import (
	"strconv"
	"time"
)

// RestModel represents an asset in REST response
type RestModel struct {
	Id            string    `json:"-"`
	StorageId     string    `json:"storage_id"`
	Slot          int16     `json:"slot"`
	TemplateId    uint32    `json:"template_id"`
	Expiration    time.Time `json:"expiration"`
	ReferenceId   uint32    `json:"reference_id"`
	ReferenceType string    `json:"reference_type"`
	// Stackable data (only present for stackable items)
	Quantity *uint32 `json:"quantity,omitempty"`
	OwnerId  *uint32 `json:"owner_id,omitempty"`
	Flag     *uint16 `json:"flag,omitempty"`
}

func (r RestModel) GetName() string {
	return "storage_assets"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// Transform converts a Model to a RestModel
func Transform(m Model[any]) RestModel {
	return RestModel{
		Id:            strconv.Itoa(int(m.Id())),
		StorageId:     m.StorageId().String(),
		Slot:          m.Slot(),
		TemplateId:    m.TemplateId(),
		Expiration:    m.Expiration(),
		ReferenceId:   m.ReferenceId(),
		ReferenceType: string(m.ReferenceType()),
	}
}

// TransformWithStackable converts a Model to a RestModel with stackable data
func TransformWithStackable(m Model[any], quantity uint32, ownerId uint32, flag uint16) RestModel {
	rm := Transform(m)
	rm.Quantity = &quantity
	rm.OwnerId = &ownerId
	rm.Flag = &flag
	return rm
}

// TransformAll converts multiple Models to RestModels
func TransformAll(models []Model[any]) []RestModel {
	result := make([]RestModel, 0, len(models))
	for _, m := range models {
		result = append(result, Transform(m))
	}
	return result
}
