package messenger

import (
	"context"
	"github.com/Chronicle20/atlas-tenant"
)

func CreateMessenger(ctx context.Context) func(characterId uint32) Model {
	return func(characterId uint32) Model {
		t := tenant.MustFromContext(ctx)
		return GetRegistry().Create(t, characterId)
	}
}

func UpdateMessenger(ctx context.Context) func(messengerId uint32, fn func(Model) Model) (Model, error) {
	return func(messengerId uint32, fn func(Model) Model) (Model, error) {
		t := tenant.MustFromContext(ctx)
		return GetRegistry().Update(t, messengerId, fn)
	}
}

func DeleteMessenger(ctx context.Context) func(messengerId uint32) {
	return func(messengerId uint32) {
		t := tenant.MustFromContext(ctx)
		GetRegistry().Remove(t, messengerId)
	}
}
