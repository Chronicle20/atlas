package character_test

import (
	"atlas-pets/kafka/consumer/character"
	"atlas-pets/pet"
	"atlas-pets/pet/exclude"
	"context"
	"testing"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func testLogger() logrus.FieldLogger {
	l, _ := test.NewNullLogger()
	return l
}

func testContext() context.Context {
	t, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return tenant.WithContext(context.Background(), t)
}

func testDatabase(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	var migrators []func(db *gorm.DB) error
	migrators = append(migrators, pet.Migration, exclude.Migration)

	for _, migrator := range migrators {
		if err = migrator(db); err != nil {
			t.Fatalf("Failed to migrate database: %v", err)
		}
	}
	return db
}

func TestInitConsumers(t *testing.T) {
	l := testLogger()
	called := false
	rf := func(config consumer.Config, decorators ...model.Decorator[consumer.Config]) {
		called = true
	}

	character.InitConsumers(l)(rf)("test-consumer-group")
	if !called {
		t.Fatalf("Expected consumer registration function to be called")
	}
}

func TestInitHandlers(t *testing.T) {
	l := testLogger()
	db := testDatabase(t)
	handlerCount := 0
	rf := func(topic string, h handler.Handler) (string, error) {
		handlerCount++
		return topic, nil
	}

	character.InitHandlers(l)(db)(rf)
	if handlerCount != 5 {
		t.Fatalf("Expected 5 handlers to be registered, got %d", handlerCount)
	}
}
