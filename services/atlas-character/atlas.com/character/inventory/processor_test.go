package inventory_test

import (
	"atlas-character/character"
	"atlas-character/equipable"
	"atlas-character/inventory"
	"atlas-character/inventory/item"
	"atlas-character/kafka/producer"
	"atlas-character/tenant"
	producer2 "github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"
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
	return tenant.Model{
		Id:           uuid.New(),
		Region:       "GMS",
		MajorVersion: 83,
		MinorVersion: 1,
	}
}

func testLogger() logrus.FieldLogger {
	l, _ := test.NewNullLogger()
	return l
}

func testSpan() opentracing.Span {
	return mocktracer.New().StartSpan("test")
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

func TestMove(t *testing.T) {
	l := testLogger()
	db := testDatabase(t)
	span := testSpan()
	tenant := testTenant()

	// Create character
	var createMessages = make([]kafka.Message, 0)
	input := character.NewModelBuilder().SetAccountId(1000).SetWorldId(0).SetName("Atlas").SetLevel(1).SetExperience(0).Build()
	c, err := character.Create(l, db, span, testProducer(&createMessages))(testTenant(), input)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	// Create inventory items
	var createItemMessages = make([]kafka.Message, 0)
	err = inventory.CreateItem(l, db, span, testProducer(&createItemMessages))(tenant, c.Id(), 2, 2000000, 100)
	if err != nil {
		t.Fatalf("Failed to create item: %v", err)
	}
	if len(createItemMessages) != 1 {
		t.Fatalf("Failed to create item: %v", createItemMessages)
	}

	err = inventory.CreateItem(l, db, span, testProducer(&createItemMessages))(tenant, c.Id(), 2, 2000001, 150)
	if err != nil {
		t.Fatalf("Failed to create item: %v", err)
	}
	if len(createItemMessages) != 2 {
		t.Fatalf("Failed to create item: %v", createItemMessages)
	}

	// validate inventory items
	inv, err := inventory.GetInventoryByType(l, db, span, tenant)(c.Id(), 2)()
	if err != nil {
		t.Fatalf("Failed to get inventory: %v", err)
	}
	i1, err := item.GetBySlot(l, db, tenant)(inv.Id(), 1)
	if err != nil {
		t.Fatalf("Failed to get inventory: %v", err)
	}
	if i1.ItemId() != 2000000 {
		t.Fatalf("Inventory item id should be 2000000, got: %d", i1.ItemId())
	}
	if i1.Quantity() != 100 {
		t.Fatalf("Inventory quantity should be 100, got: %d", i1.Quantity())
	}
	i2, err := item.GetBySlot(l, db, tenant)(inv.Id(), 2)
	if err != nil {
		t.Fatalf("Failed to get inventory: %v", err)
	}
	if i2.ItemId() != 2000001 {
		t.Fatalf("Inventory item id should be 2000001, got: %d", i2.ItemId())
	}
	if i2.Quantity() != 150 {
		t.Fatalf("Inventory quantity should be 150, got: %d", i2.Quantity())
	}

	// test move
	var moveItemMessages = make([]kafka.Message, 0)
	err = inventory.Move(l, db, span, testProducer(&moveItemMessages))(tenant, c.Id(), 2, 2, 1)
	if err != nil {
		t.Fatalf("Failed to move item: %v", err)
	}
	if len(moveItemMessages) != 1 {
		t.Fatalf("Failed to move item: %v", moveItemMessages)
	}
	i3, err := item.GetBySlot(l, db, tenant)(inv.Id(), 1)
	if err != nil {
		t.Fatalf("Failed to get inventory: %v", err)
	}
	if i3.ItemId() != 2000001 {
		t.Fatalf("Inventory item id should be 2000001, got: %d", i2.ItemId())
	}
	if i3.Quantity() != 150 {
		t.Fatalf("Inventory quantity should be 150, got: %d", i2.Quantity())
	}
	i4, err := item.GetBySlot(l, db, tenant)(inv.Id(), 2)
	if err != nil {
		t.Fatalf("Failed to get inventory: %v", err)
	}
	if i4.ItemId() != 2000000 {
		t.Fatalf("Inventory item id should be 2000000, got: %d", i1.ItemId())
	}
	if i4.Quantity() != 100 {
		t.Fatalf("Inventory quantity should be 100, got: %d", i1.Quantity())
	}
}
