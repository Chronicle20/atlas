package model

import "github.com/Chronicle20/atlas-model/model"

//TODO REMOVE THIS ONCE atlas-model updated.

//goland:noinspection GoUnusedExportedFunction
func CollapseProvider[A, T any](f func(A) model.Provider[T]) func(A) (T, error) {
	return func(a A) (T, error) {
		return f(a)()
	}
}

//goland:noinspection GoUnusedExportedFunction
func LiftToProvider[A, T any](f func(A) (T, error)) func(A) model.Provider[T] {
	return func(a A) model.Provider[T] {
		return func() (T, error) {
			return f(a)
		}
	}
}
