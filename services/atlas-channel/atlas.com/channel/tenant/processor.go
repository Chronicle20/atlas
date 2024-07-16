package tenant

import (
	"atlas-channel/configuration"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"strconv"
)

func IsServiced(id uuid.UUID) bool {
	return getRegistry().Contains(id)
}

func Create(id uuid.UUID, region string, major uint16, minor uint16) Model {
	getRegistry().Add(id)
	return Model{
		Id:           id,
		Region:       region,
		MajorVersion: major,
		MinorVersion: minor,
	}
}

func NewFromConfiguration(l logrus.FieldLogger) func(config configuration.Server) (Model, error) {
	return func(config configuration.Server) (Model, error) {
		majorVersion, err := strconv.Atoi(config.Version.Major)
		if err != nil {
			l.WithError(err).Errorf("Socket service [majorVersion] is configured incorrectly")
			return Model{}, err
		}

		minorVersion, err := strconv.Atoi(config.Version.Minor)
		if err != nil {
			l.WithError(err).Errorf("Socket service [minorVersion] is configured incorrectly")
			return Model{}, err
		}
		return Create(uuid.MustParse(config.Tenant), config.Region, uint16(majorVersion), uint16(minorVersion)), nil
	}
}

func ForAll(operator model.Operator[uuid.UUID]) {
	for _, t := range getRegistry().GetAll() {
		_ = operator(t)
	}
}
