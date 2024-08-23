package tenant

import (
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
)

//goland:noinspection GoUnusedExportedFunction
func Creator(id uuid.UUID, region string, majorVersion uint16, minorVersion uint16) model.Provider[Model] {
	t := Model{
		id:           id,
		region:       region,
		majorVersion: majorVersion,
		minorVersion: minorVersion,
	}
	getRegistry().Add(t)
	return model.FixedProvider(t)
}

//goland:noinspection GoUnusedExportedFunction
func Create(id uuid.UUID, region string, majorVersion uint16, minorVersion uint16) (Model, error) {
	return Creator(id, region, majorVersion, minorVersion)()
}

//goland:noinspection GoUnusedExportedFunction
func AllProvider() model.Provider[[]Model] {
	return model.FixedProvider(getRegistry().GetAll())
}

//goland:noinspection GoUnusedExportedFunction
func ForAll(operator model.Operator[Model]) error {
	return model.ForEachSlice(AllProvider(), operator)
}
