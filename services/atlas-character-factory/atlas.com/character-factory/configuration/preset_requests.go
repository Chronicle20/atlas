package configuration

import (
	"atlas-character-factory/configuration/tenant/characters/preset"
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

var ErrPresetNotFound = errors.New("preset not found")

type PresetClient interface {
	GetById(ctx context.Context, tenantId uuid.UUID, presetId uuid.UUID) (preset.RestModel, error)
}

type PresetClientImpl struct {
	l logrus.FieldLogger
}

func NewPresetClient(l logrus.FieldLogger) *PresetClientImpl {
	return &PresetClientImpl{l: l}
}

func (c *PresetClientImpl) GetById(ctx context.Context, tenantId uuid.UUID, presetId uuid.UUID) (preset.RestModel, error) {
	tc, err := GetTenantConfig(tenantId)
	if err != nil {
		return preset.RestModel{}, err
	}
	target := presetId.String()
	for _, p := range tc.Characters.Presets {
		if p.Id == target {
			return p, nil
		}
	}
	return preset.RestModel{}, ErrPresetNotFound
}
