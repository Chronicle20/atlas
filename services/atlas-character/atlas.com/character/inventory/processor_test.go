package inventory_test

import (
	"atlas-character/asset"
	"atlas-character/character"
	"atlas-character/equipable"
	statistics2 "atlas-character/equipable/statistics"
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
	"gorm.io/gorm/logger"
	"log"
	"os"
	"testing"
	"time"
)

func testDatabase(t *testing.T) *gorm.DB {
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second, // Slow SQL threshold
			LogLevel:                  logger.Info, // Log level
			IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound error for logger
			//ParameterizedQueries:      true,        // Don't include params in the SQL log
			//Colorful:                  false,       // Disable color
		},
	)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: newLogger})
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

func TestAdjustingEquipment(t *testing.T) {
	l := testLogger()
	db := testDatabase(t)
	span := testSpan()
	tenant := testTenant()
	t.Logf("TenantId [%s]", tenant.Id.String())

	// Create character
	var createMessages = make([]kafka.Message, 0)
	input := character.NewModelBuilder().SetAccountId(1000).SetWorldId(0).SetName("Atlas").SetLevel(1).SetExperience(0).Build()
	c, err := character.Create(l, db, span, testProducer(&createMessages))(tenant, input)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	// Create inventory items
	top := createAndVerifyMockEquip(t, l, db, span)(tenant)(c.Id())(1040010)
	bottom := createAndVerifyMockEquip(t, l, db, span)(tenant)(c.Id())(1060002)
	overall := createAndVerifyMockEquip(t, l, db, span)(tenant)(c.Id())(1050018)
	t.Logf("Top [%d], Bottom [%d], Overall [%d].", top.Slot(), bottom.Slot(), overall.Slot())

	//inventory.EquipItemForCharacter()
}

func createAndVerifyMockEquip(t *testing.T, l logrus.FieldLogger, db *gorm.DB, span opentracing.Span) func(tenant tenant.Model) func(characterId uint32) func(itemId uint32) equipable.Model {
	return func(tenant tenant.Model) func(characterId uint32) func(itemId uint32) equipable.Model {
		return func(characterId uint32) func(itemId uint32) equipable.Model {
			return func(itemId uint32) equipable.Model {
				var wipSlot int16
				iap := func(quantity uint32, slot int16) model.Provider[[]kafka.Message] {
					return func() ([]kafka.Message, error) {
						wipSlot = slot
						return make([]kafka.Message, 0), nil
					}
				}
				err := createMockEquipAsset(l, db, span, iap)(tenant)(characterId)(int8(inventory.TypeValueEquip))(itemId)
				if err != nil {
					t.Fatalf("Failed to create item: %v", err)
				}

				wipE, err := equipable.GetBySlot(db, tenant)(characterId, wipSlot)
				if err != nil {
					t.Fatalf("Failed to retreive created item.")
				}
				if wipE.ItemId() != itemId {
					t.Fatalf("Newly created item is not valid.")
				}
				return wipE
			}
		}
	}
}

func createMockEquipAsset(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, iap inventory.ItemAddProvider) func(tenant tenant.Model) func(characterId uint32) func(inventoryType int8) func(itemId uint32) error {
	return func(tenant tenant.Model) func(characterId uint32) func(inventoryType int8) func(itemId uint32) error {
		return func(characterId uint32) func(inventoryType int8) func(itemId uint32) error {
			return func(inventoryType int8) func(itemId uint32) error {
				return func(itemId uint32) error {
					inv, err := inventory.GetInventoryByType(l, db, span, tenant)(characterId, inventory.Type(inventoryType))()
					if err != nil {
						l.WithError(err).Errorf("Unable to locate inventory [%d] for character [%d].", inventoryType, characterId)
						return err
					}
					iup := func(quantity uint32, slot int16) model.Provider[[]kafka.Message] {
						return func() ([]kafka.Message, error) {
							return make([]kafka.Message, 0), nil
						}
					}
					eap := asset.NoOpSliceProvider
					smp := inventory.OfOneSlotMaxProvider
					var esc statistics2.Creator = func(itemId uint32) model.Provider[statistics2.Model] {
						return func() (statistics2.Model, error) {
							return statistics2.Model{}, nil
						}
					}
					nac := equipable.CreateItem(l, db, span, tenant, esc)(characterId)(inv.Id(), inventoryType)(itemId)
					aqu := asset.NoOpQuantityUpdater

					_, err = inventory.CreateAsset(l)(eap, smp, nac, aqu, iap, iup, 1)()
					if err != nil {
						l.WithError(err).Errorf("Unable to create [%d] equipable [%d] for character [%d].", 1, itemId, characterId)
					}
					return err
				}
			}
		}
	}
}

func createMockItemAsset(l logrus.FieldLogger) func(db *gorm.DB) func(span opentracing.Span) func(tenant tenant.Model) func(characterId uint32) func(inventoryType int8) func(itemId uint32) func(quantity uint32) error {
	return func(db *gorm.DB) func(span opentracing.Span) func(tenant tenant.Model) func(characterId uint32) func(inventoryType int8) func(itemId uint32) func(quantity uint32) error {
		return func(span opentracing.Span) func(tenant tenant.Model) func(characterId uint32) func(inventoryType int8) func(itemId uint32) func(quantity uint32) error {
			return func(tenant tenant.Model) func(characterId uint32) func(inventoryType int8) func(itemId uint32) func(quantity uint32) error {
				return func(characterId uint32) func(inventoryType int8) func(itemId uint32) func(quantity uint32) error {
					return func(inventoryType int8) func(itemId uint32) func(quantity uint32) error {
						return func(itemId uint32) func(quantity uint32) error {
							return func(quantity uint32) error {
								inv, err := inventory.GetInventoryByType(l, db, span, tenant)(characterId, inventory.Type(inventoryType))()
								if err != nil {
									l.WithError(err).Errorf("Unable to locate inventory [%d] for character [%d].", inventoryType, characterId)
									return err
								}

								iap := func(quantity uint32, slot int16) model.Provider[[]kafka.Message] {
									return func() ([]kafka.Message, error) {
										return make([]kafka.Message, 0), nil
									}
								}
								iup := func(quantity uint32, slot int16) model.Provider[[]kafka.Message] {
									return func() ([]kafka.Message, error) {
										return make([]kafka.Message, 0), nil
									}
								}
								eap := model.SliceMap(item.ByItemIdProvider(db)(tenant)(inv.Id())(itemId), item.ToAsset)
								smp := func() (uint32, error) {
									// TODO properly look this up.
									return 200, nil
								}
								nac := item.CreateItem(db, tenant)(characterId)(inv.Id(), inventoryType)(itemId)
								aqu := item.UpdateQuantity(db, tenant)

								_, err = inventory.CreateAsset(l)(eap, smp, nac, aqu, iap, iup, quantity)()
								if err != nil {
									l.WithError(err).Errorf("Unable to create [%d] equipable [%d] for character [%d].", quantity, itemId, characterId)
								}
								return err
							}
						}
					}
				}
			}
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
	err = createMockItemAsset(l)(db)(span)(tenant)(c.Id())(2)(2000000)(100)
	if err != nil {
		t.Fatalf("Failed to create item: %v", err)
	}

	err = createMockItemAsset(l)(db)(span)(tenant)(c.Id())(2)(2000001)(150)
	if err != nil {
		t.Fatalf("Failed to create item: %v", err)
	}

	// validate inventory items
	inv, err := inventory.GetInventoryByType(l, db, span, tenant)(c.Id(), 2)()
	if err != nil {
		t.Fatalf("Failed to get inventory: %v", err)
	}
	i1, err := item.GetBySlot(db, tenant)(inv.Id(), 1)
	if err != nil {
		t.Fatalf("Failed to get inventory: %v", err)
	}
	if i1.ItemId() != 2000000 {
		t.Fatalf("Inventory item id should be 2000000, got: %d", i1.ItemId())
	}
	if i1.Quantity() != 100 {
		t.Fatalf("Inventory quantity should be 100, got: %d", i1.Quantity())
	}
	i2, err := item.GetBySlot(db, tenant)(inv.Id(), 2)
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
	i3, err := item.GetBySlot(db, tenant)(inv.Id(), 1)
	if err != nil {
		t.Fatalf("Failed to get inventory: %v", err)
	}
	if i3.ItemId() != 2000001 {
		t.Fatalf("Inventory item id should be 2000001, got: %d", i2.ItemId())
	}
	if i3.Quantity() != 150 {
		t.Fatalf("Inventory quantity should be 150, got: %d", i2.Quantity())
	}
	i4, err := item.GetBySlot(db, tenant)(inv.Id(), 2)
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
