package baseline

import (
	"fmt"

	"github.com/google/uuid"
)

// PublishInputModel is the JSON:API input for POST /api/data/baseline/publish.
//
// The handler ignores the id (operator picks region+version), but the type
// must implement MarshalIdentifier so api2go can decode the inbound document.
type PublishInputModel struct {
	Id           string `json:"-"`
	Region       string `json:"region"`
	MajorVersion int    `json:"majorVersion"`
	MinorVersion int    `json:"minorVersion"`
}

func (PublishInputModel) GetName() string                                       { return "baselinePublishes" }
func (m PublishInputModel) GetID() string                                       { return m.Id }
func (m *PublishInputModel) SetID(id string) error                              { m.Id = id; return nil }
func (m *PublishInputModel) SetToOneReferenceID(_, _ string) error              { return nil }
func (m *PublishInputModel) SetToManyReferenceIDs(_ string, _ []string) error   { return nil }

// PublishOutputModel is what gets returned on success as a JSON:API document.
type PublishOutputModel struct {
	Id     string `json:"-"`
	Sha256 string `json:"sha256"`
}

func (PublishOutputModel) GetName() string                                       { return "baselinePublishes" }
func (m PublishOutputModel) GetID() string                                       { return m.Id }
func (m *PublishOutputModel) SetID(id string) error                              { m.Id = id; return nil }
func (m *PublishOutputModel) SetToOneReferenceID(_, _ string) error              { return nil }
func (m *PublishOutputModel) SetToManyReferenceIDs(_ string, _ []string) error   { return nil }

// PublishOutputId composes the canonical id used in the JSON:API response.
func PublishOutputId(region string, major, minor int) string {
	return fmt.Sprintf("%s/%d.%d", region, major, minor)
}

// RestoreInputModel is the JSON:API input for POST /api/data/baseline/restore.
type RestoreInputModel struct {
	Id           string    `json:"-"`
	Region       string    `json:"region"`
	MajorVersion int       `json:"majorVersion"`
	MinorVersion int       `json:"minorVersion"`
	TenantID     uuid.UUID `json:"tenantId"`
}

func (RestoreInputModel) GetName() string                                       { return "baselineRestores" }
func (m RestoreInputModel) GetID() string                                       { return m.Id }
func (m *RestoreInputModel) SetID(id string) error                              { m.Id = id; return nil }
func (m *RestoreInputModel) SetToOneReferenceID(_, _ string) error              { return nil }
func (m *RestoreInputModel) SetToManyReferenceIDs(_ string, _ []string) error   { return nil }
