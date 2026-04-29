package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const AffectedAreaRemovedWriter = "AffectedAreaRemoved"

// AffectedAreaRemoved is the v83 clientbound packet that despawns an
// affected-area (mist) from the field. The wire body is just the uint32 mist
// key derived from the UUID (see mistKey in affected_area_created.go).
type AffectedAreaRemoved struct {
	mistId  uuid.UUID
	ownerId uint32
}

func NewAffectedAreaRemoved(mistId uuid.UUID, ownerId uint32) AffectedAreaRemoved {
	return AffectedAreaRemoved{mistId: mistId, ownerId: ownerId}
}

func (m AffectedAreaRemoved) MistId() uuid.UUID { return m.mistId }
func (m AffectedAreaRemoved) OwnerId() uint32   { return m.ownerId }
func (m AffectedAreaRemoved) Operation() string { return AffectedAreaRemovedWriter }
func (m AffectedAreaRemoved) String() string {
	return fmt.Sprintf("mistId [%s], ownerId [%d]", m.mistId, m.ownerId)
}

func (m AffectedAreaRemoved) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(mistKey(m.mistId))
		return w.Bytes()
	}
}
