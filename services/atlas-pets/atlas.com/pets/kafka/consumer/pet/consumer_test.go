package pet_test

import (
	consumer2 "atlas-pets/kafka/consumer/pet"
	"atlas-pets/pet"
	"atlas-pets/pet/exclude"
	"context"
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
	"testing"
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
	consumerCount := 0
	rf := func(config consumer.Config, decorators ...model.Decorator[consumer.Config]) {
		consumerCount++
	}

	consumer2.InitConsumers(l)(rf)("test-consumer-group")
	if consumerCount != 2 {
		t.Fatalf("Expected 2 consumers to be registered, got %d", consumerCount)
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

	consumer2.InitHandlers(l)(db)(rf)
	if handlerCount != 8 {
		t.Fatalf("Expected 8 handlers to be registered, got %d", handlerCount)
	}
}
