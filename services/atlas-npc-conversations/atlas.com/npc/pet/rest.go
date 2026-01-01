package pet

import (
	"fmt"
	"github.com/jtumidanski/api2go/jsonapi"
	"strconv"
	"time"
)

const (
	Resource = "pets"
)

// RestModel represents the REST model for pets from the pets service
type RestModel struct {
	Id         uint32    `json:"-"`
	CashId     uint64    `json:"cashId"`
	TemplateId uint32    `json:"templateId"`
	Name       string    `json:"name"`
	Level      byte      `json:"level"`
	Closeness  uint16    `json:"closeness"`
	Fullness   byte      `json:"fullness"`
	Expiration time.Time `json:"expiration"`
	OwnerId    uint32    `json:"ownerId"`
	Slot       int8      `json:"slot"`
	X          int16     `json:"x"`
	Y          int16     `json:"y"`
	Stance     byte      `json:"stance"`
	FH         int16     `json:"fh"`
	Flag       uint16    `json:"flag"`
	PurchaseBy uint32    `json:"purchaseBy"`
}

// GetName returns the resource name
func (r RestModel) GetName() string {
	return Resource
}

// GetID returns the resource ID
func (r RestModel) GetID() string {
	return strconv.FormatUint(uint64(r.Id), 10)
}

// SetID sets the resource ID
func (r *RestModel) SetID(idStr string) error {
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid pet ID: %w", err)
	}
	r.Id = uint32(id)
	return nil
}

// GetReferences returns the resource references
func (r RestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{}
}

// GetReferencedIDs returns the referenced IDs
func (r RestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	return []jsonapi.ReferenceID{}
}

// GetReferencedStructs returns the referenced structs
func (r RestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	return []jsonapi.MarshalIdentifier{}
}

// SetToOneReferenceID sets a to-one reference ID
func (r *RestModel) SetToOneReferenceID(name, ID string) error {
	return nil
}

// SetToManyReferenceIDs sets to-many reference IDs
func (r *RestModel) SetToManyReferenceIDs(name string, IDs []string) error {
	return nil
}

// SetReferencedStructs sets referenced structs
func (r *RestModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	return nil
}

// Extract converts a REST model to a domain model
func Extract(rm RestModel) (Model, error) {
	return NewModel(rm.Id, rm.Slot), nil
}
