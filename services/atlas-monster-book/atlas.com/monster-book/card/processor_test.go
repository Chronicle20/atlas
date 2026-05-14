package card

import (
	"context"
	"testing"

	"atlas-monster-book/kafka/message"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func tenantCtx(t *testing.T, id uuid.UUID) context.Context {
	t.Helper()
	ten, err := tenant.Create(id, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	return tenant.WithContext(context.Background(), ten)
}

func TestProcessorAddInsertsAtLevel1(t *testing.T) {
	db := newDB(t)
	tid := uuid.New()
	ctx := tenantCtx(t, tid)
	p := NewProcessor(logrus.New(), ctx, db)
	mb := message.NewBuffer()
	res, err := p.Add(mb)(uuid.New(), 1, 2380000)
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if !res.Inserted || res.NewLevel != 1 || res.Duplicate {
		t.Fatalf("got %+v", res)
	}
}

func TestProcessorGetByCharacter(t *testing.T) {
	db := newDB(t)
	tid := uuid.New()
	ctx := tenantCtx(t, tid)
	p := NewProcessor(logrus.New(), ctx, db)
	mb := message.NewBuffer()
	if _, err := p.Add(mb)(uuid.New(), 1, 2380000); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Add(mb)(uuid.New(), 1, 2380001); err != nil {
		t.Fatal(err)
	}
	got, err := p.GetByCharacterId(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 cards, got %d", len(got))
	}
}
