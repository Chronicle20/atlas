package account

import (
	"atlas-account/database"
	"atlas-account/tenant"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type IdOperator func(tenant.Model, uint32) error

func SetLoggedIn(db *gorm.DB) IdOperator {
	return func(tenant tenant.Model, id uint32) error {
		return ForId(db)(tenant, id, setLoggedIn(db)(tenant))
	}
}

func SetLoggedOut(db *gorm.DB) IdOperator {
	return func(tenant tenant.Model, id uint32) error {
		return ForId(db)(tenant, id, setLoggedOut(db)(tenant))
	}
}

func setLoggedIn(db *gorm.DB) func(tenant tenant.Model) model.Operator[Model] {
	return func(tenant tenant.Model) model.Operator[Model] {
		return func(m Model) error {
			return update(db)(updateState(StateLoggedIn))(tenant, m.Id())
		}
	}
}

func setLoggedOut(db *gorm.DB) func(tenant tenant.Model) model.Operator[Model] {
	return func(tenant tenant.Model) model.Operator[Model] {
		return func(m Model) error {
			return update(db)(updateState(StateNotLoggedIn))(tenant, m.Id())
		}
	}
}

func ForId(db *gorm.DB) func(tenant tenant.Model, id uint32, operator model.Operator[Model]) error {
	return func(tenant tenant.Model, id uint32, operator model.Operator[Model]) error {
		m, err := byIdProvider(db)(tenant, id)()
		if err != nil {
			return err
		}
		return operator(m)
	}
}

func byIdProvider(db *gorm.DB) func(tenant tenant.Model, id uint32) model.Provider[Model] {
	return func(tenant tenant.Model, id uint32) model.Provider[Model] {
		return database.ModelProvider[Model, entity](db)(entityById(tenant, id), modelFromEntity)
	}
}

func byNameProvider(db *gorm.DB) func(tenant tenant.Model, name string) model.SliceProvider[Model] {
	return func(tenant tenant.Model, name string) model.SliceProvider[Model] {
		return database.ModelSliceProvider[Model, entity](db)(entitiesByName(tenant, name), modelFromEntity)
	}
}

func GetById(l logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(id uint32) (Model, error) {
	return func(id uint32) (Model, error) {
		m, err := byIdProvider(db)(tenant, id)()
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve account by id %d.", id)
			return Model{}, err
		}
		return m, nil
	}
}

func GetByName(l logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(name string) (Model, error) {
	return func(name string) (Model, error) {
		m, err := model.First[Model](byNameProvider(db)(tenant, name))
		if err != nil {
			l.WithError(err).Errorf("Unable to locate account with name %s.", name)
			return Model{}, err
		}
		return m, nil
	}
}

func GetOrCreate(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span) func(tenant tenant.Model, name string, password string, automaticRegister bool) (Model, error) {
	return func(tenant tenant.Model, name string, password string, automaticRegister bool) (Model, error) {
		m, err := model.First[Model](byNameProvider(db)(tenant, name))
		if err == nil {
			return m, nil
		}

		if !automaticRegister {
			l.Errorf("Unable to locate account by name %s, and automatic account creation is not enabled.", name)
			return Model{}, errors.New("account not found")
		}

		return Create(l, db, span)(tenant, name, password)
	}
}

func Create(l logrus.FieldLogger, db *gorm.DB, _ opentracing.Span) func(tenant tenant.Model, name string, password string) (Model, error) {
	return func(tenant tenant.Model, name string, password string) (Model, error) {
		l.Debugf("Attempting to create account %s, with password %s.", name, password)
		hashPass, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			l.WithError(err).Errorf("Error generating hash when creating account %s.", name)
			return Model{}, err
		}

		m, err := create(db)(tenant, name, string(hashPass))
		if err != nil {
			l.WithError(err).Errorf("Unable to create account %s.", name)
			return Model{}, err
		}
		return m, nil
	}
}
