package character

import (
	"atlas-character/character"
	character2 "atlas-character/kafka/message/character"
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// newConsumerTestDB mirrors character/processor_test.go's testDatabase: a
// uniquely-named shared-cache in-memory sqlite database, migrated with just
// what handleCreateCharacter's CreateAndEmit path touches (the character
// table and the outbox every emission lands in).
func newConsumerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	l, _ := test.NewNullLogger()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", uuid.NewString())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	if sqlDB, dbErr := db.DB(); dbErr == nil {
		sqlDB.SetMaxIdleConns(1)
		sqlDB.SetConnMaxLifetime(0)
		sqlDB.SetConnMaxIdleTime(0)
	}

	database.RegisterTenantCallbacks(l, db)

	for _, migrator := range []func(db *gorm.DB) error{character.Migration, outbox.Migration} {
		if err := migrator(db); err != nil {
			t.Fatalf("Failed to migrate database: %v", err)
		}
	}
	return db
}

func consumerTestLogger() logrus.FieldLogger {
	l, _ := test.NewNullLogger()
	return l
}

func consumerTestContext() context.Context {
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		panic(err)
	}
	return tenant.WithContext(context.Background(), tm)
}

// task-18: proves the wire fields land on the persisted model, mirroring
// Task 17's AP/SP additions to CreateCharacterCommandBody on the producer
// side.
func TestHandleCreateCharacterForwardsApAndSp(t *testing.T) {
	db := newConsumerTestDB(t)
	l, ctx := consumerTestLogger(), consumerTestContext()

	cmd := character2.Command[character2.CreateCharacterCommandBody]{
		TransactionId: uuid.New(),
		WorldId:       world.Id(0),
		Type:          character2.CommandCreateCharacter,
		Body: character2.CreateCharacterCommandBody{
			AccountId: 1000,
			WorldId:   world.Id(0),
			Name:      "ApSpForward",
			Level:     30,
			JobId:     job.Id(0),
			MapId:     _map.Id(0),
			AP:        12,
			SP:        "3,0,0,0,0,0,0,0,0,0",
		},
	}

	handleCreateCharacter(db)(l, ctx, cmd)

	results, err := character.NewProcessor(l, ctx, db).GetForName()("ApSpForward")
	if err != nil {
		t.Fatalf("GetForName: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected exactly 1 character named ApSpForward, got %d", len(results))
	}
	c := results[0]
	if c.AP() != 12 {
		t.Fatalf("AP should be 12, was %d", c.AP())
	}
	if c.SPString() != "3,0,0,0,0,0,0,0,0,0" {
		t.Fatalf("SP should be 3,0,0,0,0,0,0,0,0,0, was %s", c.SPString())
	}
}
