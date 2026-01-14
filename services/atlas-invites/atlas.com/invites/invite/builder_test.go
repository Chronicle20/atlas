package invite

import (
	"testing"
	"time"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/stretchr/testify/assert"
)

func TestBuilderBuild(t *testing.T) {
	ten := setupTestTenant(t)
	now := time.Now()

	m, err := NewBuilder().
		SetTenant(ten).
		SetId(1001).
		SetInviteType("BUDDY").
		SetReferenceId(5001).
		SetOriginatorId(2001).
		SetTargetId(3001).
		SetWorldId(1).
		SetAge(now).
		Build()

	assert.NoError(t, err)
	assert.Equal(t, ten, m.Tenant())
	assert.Equal(t, uint32(1001), m.Id())
	assert.Equal(t, "BUDDY", m.Type())
	assert.Equal(t, uint32(5001), m.ReferenceId())
	assert.Equal(t, uint32(2001), m.OriginatorId())
	assert.Equal(t, uint32(3001), m.TargetId())
	assert.Equal(t, byte(1), m.WorldId())
	assert.Equal(t, now, m.Age())
}

func TestBuilderValidationMissingTenant(t *testing.T) {
	_, err := NewBuilder().
		SetId(1001).
		SetInviteType("BUDDY").
		SetOriginatorId(2001).
		SetTargetId(3001).
		Build()

	assert.Error(t, err)
	assert.Equal(t, "tenant is required", err.Error())
}

func TestBuilderValidationNilTenant(t *testing.T) {
	_, err := NewBuilder().
		SetTenant(tenant.Model{}). // Zero value tenant
		SetId(1001).
		SetInviteType("BUDDY").
		SetOriginatorId(2001).
		SetTargetId(3001).
		Build()

	assert.Error(t, err)
	assert.Equal(t, "tenant is required", err.Error())
}

func TestBuilderValidationMissingId(t *testing.T) {
	ten := setupTestTenant(t)

	_, err := NewBuilder().
		SetTenant(ten).
		SetInviteType("BUDDY").
		SetOriginatorId(2001).
		SetTargetId(3001).
		Build()

	assert.Error(t, err)
	assert.Equal(t, "id is required", err.Error())
}

func TestBuilderValidationZeroId(t *testing.T) {
	ten := setupTestTenant(t)

	_, err := NewBuilder().
		SetTenant(ten).
		SetId(0).
		SetInviteType("BUDDY").
		SetOriginatorId(2001).
		SetTargetId(3001).
		Build()

	assert.Error(t, err)
	assert.Equal(t, "id is required", err.Error())
}

func TestBuilderValidationMissingInviteType(t *testing.T) {
	ten := setupTestTenant(t)

	_, err := NewBuilder().
		SetTenant(ten).
		SetId(1001).
		SetOriginatorId(2001).
		SetTargetId(3001).
		Build()

	assert.Error(t, err)
	assert.Equal(t, "inviteType is required", err.Error())
}

func TestBuilderValidationEmptyInviteType(t *testing.T) {
	ten := setupTestTenant(t)

	_, err := NewBuilder().
		SetTenant(ten).
		SetId(1001).
		SetInviteType("").
		SetOriginatorId(2001).
		SetTargetId(3001).
		Build()

	assert.Error(t, err)
	assert.Equal(t, "inviteType is required", err.Error())
}

func TestBuilderValidationMissingOriginatorId(t *testing.T) {
	ten := setupTestTenant(t)

	_, err := NewBuilder().
		SetTenant(ten).
		SetId(1001).
		SetInviteType("BUDDY").
		SetTargetId(3001).
		Build()

	assert.Error(t, err)
	assert.Equal(t, "originatorId is required", err.Error())
}

func TestBuilderValidationZeroOriginatorId(t *testing.T) {
	ten := setupTestTenant(t)

	_, err := NewBuilder().
		SetTenant(ten).
		SetId(1001).
		SetInviteType("BUDDY").
		SetOriginatorId(0).
		SetTargetId(3001).
		Build()

	assert.Error(t, err)
	assert.Equal(t, "originatorId is required", err.Error())
}

func TestBuilderValidationMissingTargetId(t *testing.T) {
	ten := setupTestTenant(t)

	_, err := NewBuilder().
		SetTenant(ten).
		SetId(1001).
		SetInviteType("BUDDY").
		SetOriginatorId(2001).
		Build()

	assert.Error(t, err)
	assert.Equal(t, "targetId is required", err.Error())
}

func TestBuilderValidationZeroTargetId(t *testing.T) {
	ten := setupTestTenant(t)

	_, err := NewBuilder().
		SetTenant(ten).
		SetId(1001).
		SetInviteType("BUDDY").
		SetOriginatorId(2001).
		SetTargetId(0).
		Build()

	assert.Error(t, err)
	assert.Equal(t, "targetId is required", err.Error())
}

func TestBuilderFluentChaining(t *testing.T) {
	ten := setupTestTenant(t)

	builder := NewBuilder()
	result := builder.
		SetTenant(ten).
		SetId(1001).
		SetInviteType("BUDDY").
		SetReferenceId(5001).
		SetOriginatorId(2001).
		SetTargetId(3001).
		SetWorldId(1)

	assert.Same(t, builder, result, "fluent methods should return the same builder instance")
}

func TestBuilderOptionalReferenceId(t *testing.T) {
	ten := setupTestTenant(t)

	// ReferenceId can be zero (optional)
	m, err := NewBuilder().
		SetTenant(ten).
		SetId(1001).
		SetInviteType("BUDDY").
		SetOriginatorId(2001).
		SetTargetId(3001).
		Build()

	assert.NoError(t, err)
	assert.Equal(t, uint32(0), m.ReferenceId())
}

func TestBuilderOptionalWorldId(t *testing.T) {
	ten := setupTestTenant(t)

	// WorldId can be zero (optional)
	m, err := NewBuilder().
		SetTenant(ten).
		SetId(1001).
		SetInviteType("BUDDY").
		SetOriginatorId(2001).
		SetTargetId(3001).
		Build()

	assert.NoError(t, err)
	assert.Equal(t, byte(0), m.WorldId())
}

func TestBuilderDefaultAge(t *testing.T) {
	ten := setupTestTenant(t)
	before := time.Now()

	m, err := NewBuilder().
		SetTenant(ten).
		SetId(1001).
		SetInviteType("BUDDY").
		SetOriginatorId(2001).
		SetTargetId(3001).
		Build()

	after := time.Now()

	assert.NoError(t, err)
	assert.True(t, m.Age().After(before) || m.Age().Equal(before))
	assert.True(t, m.Age().Before(after) || m.Age().Equal(after))
}

func TestBuilderAllInviteTypes(t *testing.T) {
	inviteTypes := []string{"BUDDY", "PARTY", "GUILD", "MESSENGER", "FAMILY", "TRADE"}

	for _, inviteType := range inviteTypes {
		t.Run(inviteType, func(t *testing.T) {
			ten := setupTestTenant(t)

			m, err := NewBuilder().
				SetTenant(ten).
				SetId(1001).
				SetInviteType(inviteType).
				SetOriginatorId(2001).
				SetTargetId(3001).
				Build()

			assert.NoError(t, err)
			assert.Equal(t, inviteType, m.Type())
		})
	}
}

func TestBuilderSettersOverwrite(t *testing.T) {
	ten1 := setupTestTenant(t)
	ten2 := setupTestTenant(t)

	m, err := NewBuilder().
		SetTenant(ten1).
		SetTenant(ten2). // Overwrite
		SetId(1001).
		SetId(2002). // Overwrite
		SetInviteType("BUDDY").
		SetInviteType("PARTY"). // Overwrite
		SetOriginatorId(1000).
		SetOriginatorId(2000). // Overwrite
		SetTargetId(3000).
		SetTargetId(4000). // Overwrite
		Build()

	assert.NoError(t, err)
	assert.Equal(t, ten2, m.Tenant())
	assert.Equal(t, uint32(2002), m.Id())
	assert.Equal(t, "PARTY", m.Type())
	assert.Equal(t, uint32(2000), m.OriginatorId())
	assert.Equal(t, uint32(4000), m.TargetId())
}
