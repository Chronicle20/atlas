package drop_test

import (
	"atlas-inventory/drop"
	"atlas-inventory/kafka/message"
	dropMsg "atlas-inventory/kafka/message/drop"
	"context"
	"testing"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

func testLogger() logrus.FieldLogger {
	l, _ := test.NewNullLogger()
	return l
}

func testFieldModel() field.Model {
	return field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
}

func TestCreateForEquipment(t *testing.T) {
	l := testLogger()
	ctx := context.Background()
	p := drop.NewProcessor(l, ctx)
	buf := message.NewBuffer()

	m := testFieldModel()
	itemId := uint32(1000000)
	equipmentId := uint32(123)
	dropType := byte(1)
	x := int16(100)
	y := int16(200)
	ownerId := uint32(456)

	err := p.CreateForEquipment(buf)(m, itemId, equipmentId, dropType, x, y, ownerId)
	if err != nil {
		t.Fatalf("CreateForEquipment failed: %v", err)
	}

	messages := buf.GetAll()
	if len(messages) == 0 {
		t.Fatal("Expected messages in buffer, got none")
	}

	topicMessages, ok := messages[dropMsg.EnvCommandTopic]
	if !ok {
		t.Fatalf("Expected messages in topic %s", dropMsg.EnvCommandTopic)
	}

	if len(topicMessages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(topicMessages))
	}
}

func TestCreateForItem(t *testing.T) {
	l := testLogger()
	ctx := context.Background()
	p := drop.NewProcessor(l, ctx)
	buf := message.NewBuffer()

	m := testFieldModel()
	itemId := uint32(2000000)
	quantity := uint32(10)
	dropType := byte(0)
	x := int16(150)
	y := int16(250)
	ownerId := uint32(789)

	err := p.CreateForItem(buf)(m, itemId, quantity, dropType, x, y, ownerId)
	if err != nil {
		t.Fatalf("CreateForItem failed: %v", err)
	}

	messages := buf.GetAll()
	if len(messages) == 0 {
		t.Fatal("Expected messages in buffer, got none")
	}

	topicMessages, ok := messages[dropMsg.EnvCommandTopic]
	if !ok {
		t.Fatalf("Expected messages in topic %s", dropMsg.EnvCommandTopic)
	}

	if len(topicMessages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(topicMessages))
	}
}

func TestCancelReservation(t *testing.T) {
	l := testLogger()
	ctx := context.Background()
	p := drop.NewProcessor(l, ctx)
	buf := message.NewBuffer()

	m := testFieldModel()
	dropId := uint32(999)
	characterId := uint32(123)

	err := p.CancelReservation(buf)(m, dropId, characterId)
	if err != nil {
		t.Fatalf("CancelReservation failed: %v", err)
	}

	messages := buf.GetAll()
	if len(messages) == 0 {
		t.Fatal("Expected messages in buffer, got none")
	}

	topicMessages, ok := messages[dropMsg.EnvCommandTopic]
	if !ok {
		t.Fatalf("Expected messages in topic %s", dropMsg.EnvCommandTopic)
	}

	if len(topicMessages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(topicMessages))
	}
}

func TestRequestPickUp(t *testing.T) {
	l := testLogger()
	ctx := context.Background()
	p := drop.NewProcessor(l, ctx)
	buf := message.NewBuffer()

	m := testFieldModel()
	dropId := uint32(888)
	characterId := uint32(456)

	err := p.RequestPickUp(buf)(m, dropId, characterId)
	if err != nil {
		t.Fatalf("RequestPickUp failed: %v", err)
	}

	messages := buf.GetAll()
	if len(messages) == 0 {
		t.Fatal("Expected messages in buffer, got none")
	}

	topicMessages, ok := messages[dropMsg.EnvCommandTopic]
	if !ok {
		t.Fatalf("Expected messages in topic %s", dropMsg.EnvCommandTopic)
	}

	if len(topicMessages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(topicMessages))
	}
}

func TestMultipleOperations(t *testing.T) {
	l := testLogger()
	ctx := context.Background()
	p := drop.NewProcessor(l, ctx)
	buf := message.NewBuffer()

	m := testFieldModel()

	// Perform multiple operations on the same buffer
	err := p.CreateForItem(buf)(m, 1000000, 5, 0, 100, 200, 123)
	if err != nil {
		t.Fatalf("First CreateForItem failed: %v", err)
	}

	err = p.CreateForItem(buf)(m, 2000000, 10, 0, 150, 250, 456)
	if err != nil {
		t.Fatalf("Second CreateForItem failed: %v", err)
	}

	err = p.RequestPickUp(buf)(m, 999, 789)
	if err != nil {
		t.Fatalf("RequestPickUp failed: %v", err)
	}

	messages := buf.GetAll()
	topicMessages, ok := messages[dropMsg.EnvCommandTopic]
	if !ok {
		t.Fatalf("Expected messages in topic %s", dropMsg.EnvCommandTopic)
	}

	if len(topicMessages) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(topicMessages))
	}
}
