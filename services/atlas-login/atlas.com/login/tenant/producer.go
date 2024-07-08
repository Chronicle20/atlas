package tenant

import (
	"atlas-login/configuration"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"strconv"
)

func New(l logrus.FieldLogger) func(config configuration.Server) (Model, error) {
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

		return Model{
			Id:           uuid.MustParse(config.Tenant),
			Region:       config.Region,
			MajorVersion: uint16(majorVersion),
			MinorVersion: uint16(minorVersion),
		}, nil
	}
}
