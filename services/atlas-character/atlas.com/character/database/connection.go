package database

import (
	"atlas-character/retry"
	"fmt"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"os"
	"strconv"
)

type DSNBuilder struct {
	user         string
	password     string
	host         string
	port         uint16
	databaseName string
}

func NewDSNBuilder() *DSNBuilder {
	return &DSNBuilder{
		user:         "",
		password:     "",
		host:         "",
		port:         0,
		databaseName: "",
	}
}

func (d *DSNBuilder) SetUser(value string) *DSNBuilder {
	d.user = value
	return d
}

func (d *DSNBuilder) SetPassword(value string) *DSNBuilder {
	d.password = value
	return d
}

func (d *DSNBuilder) SetHost(value string) *DSNBuilder {
	d.host = value
	return d
}

func (d *DSNBuilder) SetPort(port uint16) *DSNBuilder {
	d.port = port
	return d
}

func (d *DSNBuilder) SetDatabaseName(value string) *DSNBuilder {
	d.databaseName = value
	return d
}

func (d *DSNBuilder) Build() string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=UTC", d.host, d.user, d.password, d.databaseName, d.port)
}

type Configuration struct {
	dsn        string
	migrations []Migrator
}

type Configurator func(c *Configuration)

func SetMigrations(migrations ...Migrator) Configurator {
	return func(c *Configuration) {
		c.migrations = migrations
	}
}

type Migrator func(db *gorm.DB) error

func Connect(l logrus.FieldLogger, configurators ...Configurator) *gorm.DB {
	dsnBuilder := NewDSNBuilder()
	user, ok := os.LookupEnv("DB_USER")
	if ok {
		dsnBuilder = dsnBuilder.SetUser(user)
	}

	password, ok := os.LookupEnv("DB_PASSWORD")
	if ok {
		dsnBuilder = dsnBuilder.SetPassword(password)
	}

	host, ok := os.LookupEnv("DB_HOST")
	if ok {
		dsnBuilder = dsnBuilder.SetHost(host)
	}

	portStr, ok := os.LookupEnv("DB_PORT")
	if ok {
		port, err := strconv.Atoi(portStr)
		if err == nil {
			dsnBuilder = dsnBuilder.SetPort(uint16(port))
		}
	}

	databaseName, ok := os.LookupEnv("DB_NAME")
	if ok {
		dsnBuilder = dsnBuilder.SetDatabaseName(databaseName)
	}

	c := &Configuration{
		dsn:        dsnBuilder.Build(),
		migrations: make([]Migrator, 0),
	}
	for _, configurator := range configurators {
		configurator(c)
	}

	var db *gorm.DB
	tryToConnect := func(attempt int) (bool, error) {
		var err error
		db, err = gorm.Open(postgres.Open(dsnBuilder.Build()), &gorm.Config{})
		if err != nil {
			return true, err
		}
		return false, err
	}

	err := retry.Try(tryToConnect, 10)
	if err != nil {
		l.WithError(err).Fatalf("Failed to connect to database.")
	}

	// Migrate the schema
	for _, m := range c.migrations {
		err = m(db)
		if err != nil {
			l.WithError(err).Fatalf("Migrating schema.")
		}
	}
	return db
}
