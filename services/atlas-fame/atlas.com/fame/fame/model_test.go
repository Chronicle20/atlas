package fame

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func createTestModel() Model {
	return Model{
		tenantId:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		id:          uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		characterId: 1000,
		targetId:    2000,
		amount:      1,
		createdAt:   time.Date(2026, 1, 13, 12, 0, 0, 0, time.UTC),
	}
}

func TestModel_TenantId(t *testing.T) {
	m := createTestModel()

	expected := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	assert.Equal(t, expected, m.TenantId())
}

func TestModel_Id(t *testing.T) {
	m := createTestModel()

	expected := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	assert.Equal(t, expected, m.Id())
}

func TestModel_CharacterId(t *testing.T) {
	m := createTestModel()

	assert.Equal(t, uint32(1000), m.CharacterId())
}

func TestModel_TargetId(t *testing.T) {
	m := createTestModel()

	assert.Equal(t, uint32(2000), m.TargetId())
}

func TestModel_Amount(t *testing.T) {
	m := createTestModel()

	assert.Equal(t, int8(1), m.Amount())
}

func TestModel_CreatedAt(t *testing.T) {
	m := createTestModel()

	expected := time.Date(2026, 1, 13, 12, 0, 0, 0, time.UTC)
	assert.Equal(t, expected, m.CreatedAt())
}

func TestModel_AllAccessors(t *testing.T) {
	tenantId := uuid.New()
	id := uuid.New()
	createdAt := time.Now()

	m := Model{
		tenantId:    tenantId,
		id:          id,
		characterId: 3000,
		targetId:    4000,
		amount:      -1,
		createdAt:   createdAt,
	}

	assert.Equal(t, tenantId, m.TenantId())
	assert.Equal(t, id, m.Id())
	assert.Equal(t, uint32(3000), m.CharacterId())
	assert.Equal(t, uint32(4000), m.TargetId())
	assert.Equal(t, int8(-1), m.Amount())
	assert.Equal(t, createdAt, m.CreatedAt())
}
