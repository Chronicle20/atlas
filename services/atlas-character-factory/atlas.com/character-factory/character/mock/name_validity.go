package mock

import (
	"atlas-character-factory/character"
	"context"
)

type FakeNameValidityClient struct {
	Result character.NameValidityResult
	Err    error
}

func (f *FakeNameValidityClient) Check(_ context.Context, _ string, _ byte) (character.NameValidityResult, error) {
	if f.Err != nil {
		return character.NameValidityResult{}, f.Err
	}
	return f.Result, nil
}
