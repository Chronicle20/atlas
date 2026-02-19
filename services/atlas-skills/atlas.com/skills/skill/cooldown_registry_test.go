package skill

import (
	"context"
	"testing"
	"time"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func setupCooldownRegistryTest(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(client)
}

func setupCooldownTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	assert.NoError(t, err)
	return ten
}

func cooldownTestCtx(t tenant.Model) context.Context {
	return tenant.WithContext(context.Background(), t)
}

func TestCooldownRegistry_Apply_And_Get(t *testing.T) {
	setupCooldownRegistryTest(t)
	ten := setupCooldownTestTenant(t)
	ctx := cooldownTestCtx(ten)

	err := GetRegistry().Apply(ctx, 1000, 2001001, 30)
	assert.NoError(t, err)

	expiresAt, err := GetRegistry().Get(ctx, 1000, 2001001)
	assert.NoError(t, err)
	assert.True(t, expiresAt.After(time.Now()))
	assert.True(t, expiresAt.Before(time.Now().Add(31*time.Second)))
}

func TestCooldownRegistry_Get_NotFound(t *testing.T) {
	setupCooldownRegistryTest(t)
	ten := setupCooldownTestTenant(t)
	ctx := cooldownTestCtx(ten)

	_, err := GetRegistry().Get(ctx, 1000, 999)
	assert.Error(t, err)
}

func TestCooldownRegistry_Clear(t *testing.T) {
	setupCooldownRegistryTest(t)
	ten := setupCooldownTestTenant(t)
	ctx := cooldownTestCtx(ten)

	_ = GetRegistry().Apply(ctx, 1000, 2001001, 30)

	err := GetRegistry().Clear(ctx, 1000, 2001001)
	assert.NoError(t, err)

	_, err = GetRegistry().Get(ctx, 1000, 2001001)
	assert.Error(t, err)
}

func TestCooldownRegistry_ClearAll(t *testing.T) {
	setupCooldownRegistryTest(t)
	ten := setupCooldownTestTenant(t)
	ctx := cooldownTestCtx(ten)

	_ = GetRegistry().Apply(ctx, 1000, 2001001, 30)
	_ = GetRegistry().Apply(ctx, 1000, 2001002, 60)
	_ = GetRegistry().Apply(ctx, 1000, 2001003, 90)

	err := GetRegistry().ClearAll(ctx, 1000)
	assert.NoError(t, err)

	_, err = GetRegistry().Get(ctx, 1000, 2001001)
	assert.Error(t, err)
	_, err = GetRegistry().Get(ctx, 1000, 2001002)
	assert.Error(t, err)
	_, err = GetRegistry().Get(ctx, 1000, 2001003)
	assert.Error(t, err)
}

func TestCooldownRegistry_ClearAll_DoesNotAffectOtherCharacters(t *testing.T) {
	setupCooldownRegistryTest(t)
	ten := setupCooldownTestTenant(t)
	ctx := cooldownTestCtx(ten)

	_ = GetRegistry().Apply(ctx, 1000, 2001001, 30)
	_ = GetRegistry().Apply(ctx, 2000, 2001001, 30)

	_ = GetRegistry().ClearAll(ctx, 1000)

	_, err := GetRegistry().Get(ctx, 2000, 2001001)
	assert.NoError(t, err)
}

func TestCooldownRegistry_ClearAll_NonExistent(t *testing.T) {
	setupCooldownRegistryTest(t)
	ten := setupCooldownTestTenant(t)
	ctx := cooldownTestCtx(ten)

	err := GetRegistry().ClearAll(ctx, 9999)
	assert.NoError(t, err)
}

func TestCooldownRegistry_GetAll(t *testing.T) {
	setupCooldownRegistryTest(t)
	ten := setupCooldownTestTenant(t)
	ctx := cooldownTestCtx(ten)

	_ = GetRegistry().Apply(ctx, 1000, 2001001, 30)
	_ = GetRegistry().Apply(ctx, 1000, 2001002, 60)
	_ = GetRegistry().Apply(ctx, 2000, 3001001, 90)

	all := GetRegistry().GetAll(context.Background())
	assert.Len(t, all, 3)

	charSkills := make(map[uint32][]uint32)
	for _, h := range all {
		charSkills[h.CharacterId()] = append(charSkills[h.CharacterId()], h.SkillId())
		ht := h.Tenant()
		assert.Equal(t, ten.Id(), ht.Id())
	}
	assert.Len(t, charSkills[1000], 2)
	assert.Len(t, charSkills[2000], 1)
}

func TestCooldownRegistry_GetAll_CrossTenant(t *testing.T) {
	setupCooldownRegistryTest(t)

	ten1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ten2, _ := tenant.Create(uuid.New(), "EMS", 83, 1)

	ctx1 := cooldownTestCtx(ten1)
	ctx2 := cooldownTestCtx(ten2)

	_ = GetRegistry().Apply(ctx1, 1000, 2001001, 30)
	_ = GetRegistry().Apply(ctx2, 2000, 3001001, 60)

	all := GetRegistry().GetAll(context.Background())
	assert.Len(t, all, 2)

	tenantIds := make(map[uuid.UUID]bool)
	for _, h := range all {
		ht := h.Tenant()
		tenantIds[ht.Id()] = true
	}
	assert.True(t, tenantIds[ten1.Id()])
	assert.True(t, tenantIds[ten2.Id()])
}

func TestCooldownRegistry_TenantIsolation(t *testing.T) {
	setupCooldownRegistryTest(t)

	ten1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ten2, _ := tenant.Create(uuid.New(), "EMS", 83, 1)

	ctx1 := cooldownTestCtx(ten1)
	ctx2 := cooldownTestCtx(ten2)

	_ = GetRegistry().Apply(ctx1, 1000, 2001001, 30)

	_, err := GetRegistry().Get(ctx1, 1000, 2001001)
	assert.NoError(t, err)

	_, err = GetRegistry().Get(ctx2, 1000, 2001001)
	assert.Error(t, err)
}

func TestCooldownRegistry_MultipleSkillsSameCharacter(t *testing.T) {
	setupCooldownRegistryTest(t)
	ten := setupCooldownTestTenant(t)
	ctx := cooldownTestCtx(ten)

	_ = GetRegistry().Apply(ctx, 1000, 2001001, 30)
	_ = GetRegistry().Apply(ctx, 1000, 2001002, 60)

	exp1, err := GetRegistry().Get(ctx, 1000, 2001001)
	assert.NoError(t, err)

	exp2, err := GetRegistry().Get(ctx, 1000, 2001002)
	assert.NoError(t, err)

	assert.True(t, exp2.After(exp1))
}
