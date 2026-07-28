package skill

import (
	skill2 "atlas-skills/kafka/message/skill"
	"atlas-skills/skill"
	"atlas-skills/test"
	"os"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	logtest "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
)

func TestMain(m *testing.M) {
	producertest.InstallNoop()
	os.Exit(m.Run())
}

func setupResetConsumerTest(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	skill.InitRegistry(client)
}

// Wrong command type must leave the registry untouched.
func TestHandleCommandResetCooldowns_TypeGuard(t *testing.T) {
	setupResetConsumerTest(t)
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)
	ctx := test.CreateTestContext()
	logger, _ := logtest.NewNullLogger()

	characterId := uint32(100)
	if err := skill.GetRegistry().Apply(ctx, characterId, 1311006, 300); err != nil {
		t.Fatalf("Apply() unexpected error: %v", err)
	}

	c := skill2.Command[skill2.ResetCooldownsBody]{
		TransactionId: uuid.New(),
		WorldId:       world.Id(0),
		CharacterId:   characterId,
		Type:          skill2.CommandTypeSetCooldown, // wrong type
		Body:          skill2.ResetCooldownsBody{ExceptSkillIds: []uint32{5121010}},
	}
	handleCommandResetCooldowns(db)(logger, ctx, c)

	if _, err := skill.GetRegistry().Get(ctx, characterId, 1311006); err != nil {
		t.Fatalf("registry modified by wrong-type command: %v", err)
	}
}

// Happy path: cooldowns cleared except the exclusion list. TestMain installs
// a noop Kafka producer, so the emit step succeeds; this test asserts on
// registry state only, which is all the handler's return value reflects.
func TestHandleCommandResetCooldowns_ClearsRegistry(t *testing.T) {
	setupResetConsumerTest(t)
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)
	ctx := test.CreateTestContext()
	logger, _ := logtest.NewNullLogger()

	characterId := uint32(100)
	for _, id := range []uint32{5121010, 1311006, 5221006} {
		if err := skill.GetRegistry().Apply(ctx, characterId, id, 300); err != nil {
			t.Fatalf("Apply() unexpected error: %v", err)
		}
	}

	c := skill2.Command[skill2.ResetCooldownsBody]{
		TransactionId: uuid.New(),
		WorldId:       world.Id(0),
		CharacterId:   characterId,
		Type:          skill2.CommandTypeResetCooldowns,
		Body: skill2.ResetCooldownsBody{
			ExceptSkillIds: []uint32{5121010},
			SourceSkillId:  5121010,
		},
	}
	handleCommandResetCooldowns(db)(logger, ctx, c)

	if _, err := skill.GetRegistry().Get(ctx, characterId, 5121010); err != nil {
		t.Fatalf("excepted cooldown 5121010 removed: %v", err)
	}
	if _, err := skill.GetRegistry().Get(ctx, characterId, 1311006); err == nil {
		t.Fatalf("cooldown 1311006 not cleared")
	}
	if _, err := skill.GetRegistry().Get(ctx, characterId, 5221006); err == nil {
		t.Fatalf("cooldown 5221006 not cleared")
	}
}
