package mount

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func taskTestLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)
	return l
}

// tickCall records a single invocation of the applyTick seam so the test can
// assert the loop ticked the right (tenant, worldId, characterId) once per
// active entry.
type tickCall struct {
	tenantId    uuid.UUID
	worldId     world.Id
	characterId uint32
}

func TestTirednessTask_TicksEachActiveMountOnce(t *testing.T) {
	t1, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	assert.NoError(t, err)
	t2, err := tenant.Create(uuid.New(), "JMS", 83, 1)
	assert.NoError(t, err)

	active := []ActiveEntry{
		{Tenant: t1, CharacterId: 100, Ctx: MountRideContext{WorldId: world.Id(0), SkillId: 80001000, VehicleId: 1902000}},
		{Tenant: t2, CharacterId: 200, Ctx: MountRideContext{WorldId: world.Id(5), SkillId: 80001001, VehicleId: 1902001}},
	}

	origGetActive := getActive
	origApplyTick := applyTick
	t.Cleanup(func() {
		getActive = origGetActive
		applyTick = origApplyTick
	})

	getActive = func(_ context.Context) ([]ActiveEntry, error) {
		return active, nil
	}

	var calls []tickCall
	applyTick = func(_ logrus.FieldLogger, ctx context.Context, _ *gorm.DB, worldId world.Id, characterId uint32) error {
		// The ctx handed to the tick path MUST carry that entry's tenant so the
		// processor scopes to the correct tenant's row.
		tn := tenant.MustFromContext(ctx)
		calls = append(calls, tickCall{tenantId: tn.Id(), worldId: worldId, characterId: characterId})
		return nil
	}

	task := NewTirednessTask(taskTestLogger(), nil, time.Minute)
	task.Run()

	assert.Len(t, calls, 2, "Run must tick once per active entry")

	assert.Equal(t, t1.Id(), calls[0].tenantId, "first entry must carry tenant 1")
	assert.Equal(t, world.Id(0), calls[0].worldId)
	assert.Equal(t, uint32(100), calls[0].characterId)

	assert.Equal(t, t2.Id(), calls[1].tenantId, "second entry must carry tenant 2")
	assert.Equal(t, world.Id(5), calls[1].worldId)
	assert.Equal(t, uint32(200), calls[1].characterId)
}

func TestTirednessTask_SleepTime(t *testing.T) {
	task := NewTirednessTask(taskTestLogger(), nil, time.Minute)
	assert.Equal(t, time.Minute, task.SleepTime())
}
