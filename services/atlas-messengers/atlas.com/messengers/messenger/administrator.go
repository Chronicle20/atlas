package messenger

import (
	"context"
)

func CreateMessenger(ctx context.Context) func(characterId uint32) Model {
	return func(characterId uint32) Model {
		return GetRegistry().Create(ctx, characterId)
	}
}

func UpdateMessenger(ctx context.Context) func(messengerId uint32, fn func(Model) Model) (Model, error) {
	return func(messengerId uint32, fn func(Model) Model) (Model, error) {
		return GetRegistry().Update(ctx, messengerId, fn)
	}
}

func DeleteMessenger(ctx context.Context) func(messengerId uint32) {
	return func(messengerId uint32) {
		GetRegistry().Remove(ctx, messengerId)
	}
}
