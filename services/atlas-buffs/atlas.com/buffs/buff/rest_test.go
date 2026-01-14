package buff

import (
	"atlas-buffs/buff/stat"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRestModel_GetName(t *testing.T) {
	rm := RestModel{}
	assert.Equal(t, "buffs", rm.GetName())
}

func TestRestModel_GetID(t *testing.T) {
	rm := RestModel{Id: "test-id-123"}
	assert.Equal(t, "test-id-123", rm.GetID())
}

func TestRestModel_SetID(t *testing.T) {
	rm := &RestModel{}
	err := rm.SetID("new-id-456")

	assert.NoError(t, err)
	assert.Equal(t, "new-id-456", rm.Id)
}

func TestTransform(t *testing.T) {
	changes := []stat.Model{
		stat.NewStat("STR", 10),
		stat.NewStat("DEX", 5),
	}
	b, err := NewBuff(int32(2001001), int32(60), changes)
	assert.NoError(t, err)

	rm, err := Transform(b)

	assert.NoError(t, err)
	assert.NotEmpty(t, rm.Id)
	assert.Equal(t, int32(2001001), rm.SourceId)
	assert.Equal(t, int32(60), rm.Duration)
	assert.Len(t, rm.Changes, 2)
	assert.NotZero(t, rm.CreatedAt)
	assert.NotZero(t, rm.ExpiresAt)
}

func TestTransform_StatChanges(t *testing.T) {
	changes := []stat.Model{
		stat.NewStat("STR", 10),
		stat.NewStat("DEX", 5),
		stat.NewStat("INT", 15),
	}
	b, err := NewBuff(int32(2001001), int32(60), changes)
	assert.NoError(t, err)

	rm, err := Transform(b)

	assert.NoError(t, err)
	assert.Len(t, rm.Changes, 3)

	// Verify stat changes are transformed correctly
	assert.Equal(t, "STR", rm.Changes[0].Type)
	assert.Equal(t, int32(10), rm.Changes[0].Amount)
	assert.Equal(t, "DEX", rm.Changes[1].Type)
	assert.Equal(t, int32(5), rm.Changes[1].Amount)
	assert.Equal(t, "INT", rm.Changes[2].Type)
	assert.Equal(t, int32(15), rm.Changes[2].Amount)
}
