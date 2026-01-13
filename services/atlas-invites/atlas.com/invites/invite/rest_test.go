package invite

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTransform(t *testing.T) {
	ten := setupTestTenant(t)
	now := time.Now()

	// Create a model via the registry to get a properly constructed instance
	m := GetRegistry().Create(ten, 1001, 1, 2001, "BUDDY", 5001)

	rm, err := Transform(m)

	assert.NoError(t, err)
	assert.Equal(t, m.Id(), rm.Id)
	assert.Equal(t, m.Type(), rm.Type)
	assert.Equal(t, m.ReferenceId(), rm.ReferenceId)
	assert.Equal(t, m.OriginatorId(), rm.OriginatorId)
	assert.Equal(t, m.TargetId(), rm.TargetId)
	assert.WithinDuration(t, now, rm.Age, time.Second)
}

func TestTransform_AllInviteTypes(t *testing.T) {
	inviteTypes := []string{"BUDDY", "PARTY", "GUILD", "MESSENGER", "FAMILY", "TRADE"}

	for _, inviteType := range inviteTypes {
		t.Run(inviteType, func(t *testing.T) {
			ten := setupTestTenant(t)
			m := GetRegistry().Create(ten, 1001, 1, 2001, inviteType, 5001)

			rm, err := Transform(m)

			assert.NoError(t, err)
			assert.Equal(t, inviteType, rm.Type)
		})
	}
}

func TestRestModel_GetName(t *testing.T) {
	rm := RestModel{}
	assert.Equal(t, "invites", rm.GetName())
}

func TestRestModel_GetID(t *testing.T) {
	rm := RestModel{Id: 12345}
	assert.Equal(t, "12345", rm.GetID())
}

func TestRestModel_SetID(t *testing.T) {
	rm := RestModel{}

	err := rm.SetID("12345")

	assert.NoError(t, err)
	assert.Equal(t, uint32(12345), rm.Id)
}

func TestRestModel_SetID_InvalidInput(t *testing.T) {
	rm := RestModel{}

	err := rm.SetID("not-a-number")

	assert.Error(t, err)
}

func TestRestModel_SetID_EmptyString(t *testing.T) {
	rm := RestModel{}

	err := rm.SetID("")

	assert.Error(t, err)
}
