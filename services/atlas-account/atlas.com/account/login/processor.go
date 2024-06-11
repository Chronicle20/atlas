package login

import (
	"atlas-account/account"
	"atlas-account/configuration"
	"atlas-account/tenant"
	"errors"
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	SystemError       = "SYSTEM_ERROR"
	NotRegistered     = "NOT_REGISTERED"
	DeletedOrBlocked  = "DELETED_OR_BLOCKED"
	AlreadyLoggedIn   = "ALREADY_LOGGED_IN"
	IncorrectPassword = "INCORRECT_PASSWORD"
)

func AttemptLogin(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(sessionId uuid.UUID, name string, password string) error {
	return func(sessionId uuid.UUID, name string, password string) error {
		if checkLoginAttempts(sessionId) > 4 {
			return errors.New("TOO_MANY_ATTEMPTS")
		}

		c, err := configuration.Get()
		if err != nil {
			l.WithError(err).Errorf("Error reading needed configuration.")
			return errors.New(SystemError)
		}

		a, err := account.GetOrCreate(l, db, span)(tenant, name, password, c.AutomaticRegister)
		if err != nil && !c.AutomaticRegister {
			return errors.New(NotRegistered)
		}
		if err != nil {
			return errors.New(SystemError)
		}

		if a.Banned() {
			return errors.New(DeletedOrBlocked)
		}

		// TODO implement ip, mac, and temporary banning practices

		if a.State() != account.StateNotLoggedIn {
			return errors.New(AlreadyLoggedIn)
		} else if a.Password()[0] == uint8('$') && a.Password()[1] == uint8('2') && bcrypt.CompareHashAndPassword([]byte(a.Password()), []byte(password)) == nil {
			// TODO implement tos tracking
		} else {
			return errors.New(IncorrectPassword)
		}

		err = account.SetLoggedIn(db)(tenant, a.Id())
		if err != nil {
			l.WithError(err).Errorf("Error trying to update logged in state for %s.", name)
			return errors.New(SystemError)
		}

		return nil
	}
}

func checkLoginAttempts(sessionId uuid.UUID) byte {
	return 0
}
