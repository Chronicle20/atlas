package character_test

import (
	"atlas-character/character"
	"atlas-character/equipable"
	"atlas-character/inventory"
	"atlas-character/inventory/item"
	"atlas-character/kafka/producer"
	"context"
	producer2 "github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"testing"
)

func testDatabase(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	var migrators []func(db *gorm.DB) error
	migrators = append(migrators, character.Migration, inventory.Migration, item.Migration, equipable.Migration)

	for _, migrator := range migrators {
		if err := migrator(db); err != nil {
			t.Fatalf("Failed to migrate database: %v", err)
		}
	}
	return db
}

func testTenant() tenant.Model {
	t, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return t
}

func testLogger() logrus.FieldLogger {
	l, _ := test.NewNullLogger()
	return l
}

func testProducer(output *[]kafka.Message) producer.Provider {
	return func(token string) producer2.MessageProducer {
		return func(provider model.Provider[[]kafka.Message]) error {
			res, err := provider()
			if err != nil {
				return err
			}
			for _, r := range res {
				*output = append(*output, r)
			}
			return nil
		}
	}
}

func TestCreateSunny(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())

	input := character.NewModelBuilder().SetAccountId(1000).SetWorldId(0).SetName("Atlas").SetLevel(1).SetExperience(0).Build()

	var outputMessages = make([]kafka.Message, 0)

	c, err := character.Create(testLogger())(testDatabase(t))(tctx)(testProducer(&outputMessages))(input)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}
	if c.AccountId() != 1000 {
		t.Fatalf("Account id should be 1000, was %d", c.AccountId())
	}
	if c.WorldId() != 0 {
		t.Fatalf("World id should be 0, was %d", c.WorldId())
	}
	if c.Name() != "Atlas" {
		t.Fatalf("Name should be Atlas")
	}
	if c.Level() != 1 {
		t.Fatalf("Level should be 1, was %d", c.Level())
	}
	if c.Experience() != 0 {
		t.Fatalf("Experience should be 0, was %d", c.Experience())
	}
	if len(outputMessages) != 1 {
		t.Fatalf("Number of output messages should be 1, was %d", len(outputMessages))
	}
}
