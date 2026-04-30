package mock

import (
	"atlas-character-factory/configuration"
	"atlas-character-factory/configuration/tenant/characters/preset"
	"context"

	"github.com/google/uuid"
)

type FakePresetClient struct {
	Presets map[uuid.UUID]preset.RestModel
	Err     error
}

func (f *FakePresetClient) GetById(_ context.Context, _ uuid.UUID, presetId uuid.UUID) (preset.RestModel, error) {
	if f.Err != nil {
		return preset.RestModel{}, f.Err
	}
	if p, ok := f.Presets[presetId]; ok {
		return p, nil
	}
	return preset.RestModel{}, configuration.ErrPresetNotFound
}
