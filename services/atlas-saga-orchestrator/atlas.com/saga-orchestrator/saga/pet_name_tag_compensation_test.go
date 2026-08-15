//go:build test

package saga

import (
	petmock "atlas-saga-orchestrator/pet/mock"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	petNameTagCharId  = uint32(77002)
	petNameTagItemId  = uint32(5170000)
	petNameTagPetId   = uint32(4242)
	petNameTagNewName = "Renamed"
	petNameTagOldName = "Original"
)

func newPetNameTagSaga(t *testing.T, tx uuid.UUID, renameStatus Status) Saga {
	t.Helper()
	s, err := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(PetNameTagUse).
		SetInitiatedBy("pet-name-tag-compensation-test").
		AddStep("rename_pet", renameStatus, RenamePet, RenamePetPayload{
			CharacterId:  petNameTagCharId,
			PetId:        petNameTagPetId,
			Name:         petNameTagNewName,
			PreviousName: petNameTagOldName,
		}).
		AddStep("consume_pet_name_tag", Failed, DestroyAsset, DestroyAssetPayload{
			CharacterId: petNameTagCharId,
			TemplateId:  petNameTagItemId,
			Quantity:    1,
			RemoveAll:   false,
		}).
		Build()
	require.NoError(t, err)
	return s
}

type renameCall struct {
	PetId       uint32
	CharacterId uint32
	Name        string
}

func recordingPetMock(calls *[]renameCall) *petmock.ProcessorMock {
	return &petmock.ProcessorMock{
		RenameAndEmitFunc: func(_ uuid.UUID, petId uint32, characterId uint32, name string) error {
			*calls = append(*calls, renameCall{petId, characterId, name})
			return nil
		},
	}
}

// A failed consume_pet_name_tag must revert the pet's name to the PreviousName
// captured at saga-build time — exactly once (PRD FR-7.4). Without this the
// player is told the rename failed while the pet keeps the new name.
func TestPetNameTagCompensationRevertsName(t *testing.T) {
	logger, _ := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	var calls []renameCall
	s := newPetNameTagSaga(t, uuid.New(), Completed)
	NewCompensator(logger, testTenantContext()).
		WithPetProcessor(recordingPetMock(&calls)).
		DispatchPetNameTagRollbacks(s)

	require.Len(t, calls, 1, "completed rename must be reverted exactly once")
	assert.Equal(t, petNameTagPetId, calls[0].PetId)
	assert.Equal(t, petNameTagCharId, calls[0].CharacterId)
	assert.Equal(t, petNameTagOldName, calls[0].Name, "revert must carry PreviousName")
}

// A rename step that never completed applied nothing and has no inverse. Issuing
// a revert here would overwrite a name the saga never set.
func TestPetNameTagCompensationSkipsUncompletedRename(t *testing.T) {
	logger, _ := test.NewNullLogger()

	var calls []renameCall
	s := newPetNameTagSaga(t, uuid.New(), Failed)
	NewCompensator(logger, testTenantContext()).
		WithPetProcessor(recordingPetMock(&calls)).
		DispatchPetNameTagRollbacks(s)

	assert.Empty(t, calls, "an uncompleted rename must not be reverted")
}
