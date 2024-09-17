package tenant

import (
	"context"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
)

const (
	ID           = "TENANT_ID"
	Region       = "REGION"
	MajorVersion = "MAJOR_VERSION"
	MinorVersion = "MINOR_VERSION"
)

//goland:noinspection GoUnusedExportedFunction
func Creator(id uuid.UUID, region string, majorVersion uint16, minorVersion uint16) model.Provider[Model] {
	t := Model{
		id:           id,
		region:       region,
		majorVersion: majorVersion,
		minorVersion: minorVersion,
	}
	return model.FixedProvider(t)
}

//goland:noinspection GoUnusedExportedFunction
func Create(id uuid.UUID, region string, majorVersion uint16, minorVersion uint16) (Model, error) {
	return Creator(id, region, majorVersion, minorVersion)()
}

//goland:noinspection GoUnusedExportedFunction
func Register(id uuid.UUID, region string, majorVersion uint16, minorVersion uint16) (Model, error) {
	t, err := Create(id, region, majorVersion, minorVersion)
	if err != nil {
		return Model{}, err
	}
	getRegistry().Add(t)
	return t, nil
}

//goland:noinspection GoUnusedExportedFunction
func AllProvider() model.Provider[[]Model] {
	return model.FixedProvider(getRegistry().GetAll())
}

//goland:noinspection GoUnusedExportedFunction
func ForAll(operator model.Operator[Model]) error {
	return model.ForEachSlice(AllProvider(), operator)
}

//goland:noinspection GoUnusedExportedFunction
func FromContext(ctx context.Context) model.Provider[Model] {
	var ok bool
	var id uuid.UUID
	var region string
	var majorVersion uint16
	var minorVersion uint16

	if id, ok = ctx.Value(ID).(uuid.UUID); !ok {
		return model.ErrorProvider[Model](errors.New("unable to retrieve id from context"))
	}
	if region, ok = ctx.Value(Region).(string); !ok {
		return model.ErrorProvider[Model](errors.New("unable to retrieve region from context"))
	}
	if majorVersion, ok = ctx.Value(MajorVersion).(uint16); !ok {
		return model.ErrorProvider[Model](errors.New("unable to retrieve majorVersion from context"))
	}
	if minorVersion, ok = ctx.Value(MinorVersion).(uint16); !ok {
		return model.ErrorProvider[Model](errors.New("unable to retrieve minorVersion from context"))
	}
	return func() (Model, error) {
		return Model{id: id, region: region, majorVersion: majorVersion, minorVersion: minorVersion}, nil
	}
}

//goland:noinspection GoUnusedExportedFunction
func MustFromContext(ctx context.Context) Model {
	t, err := FromContext(ctx)()
	if err != nil {
		panic("ctx parse err: " + err.Error())
	}
	return t
}

//goland:noinspection GoUnusedExportedFunction
func WithContext(ctx context.Context, tenant Model) context.Context {
	var wctx = ctx
	wctx = context.WithValue(wctx, ID, tenant.Id())
	wctx = context.WithValue(wctx, Region, tenant.Region())
	wctx = context.WithValue(wctx, MajorVersion, tenant.MajorVersion())
	wctx = context.WithValue(wctx, MinorVersion, tenant.MinorVersion())
	return wctx
}
