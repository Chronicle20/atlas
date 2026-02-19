package database

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type DSNBuilder struct {
	user         string
	password     string
	host         string
	port         uint16
	databaseName string
}

func NewDSNBuilder() *DSNBuilder {
	return &DSNBuilder{}
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
	if user, ok := os.LookupEnv("DB_USER"); ok {
		dsnBuilder = dsnBuilder.SetUser(user)
	}
	if password, ok := os.LookupEnv("DB_PASSWORD"); ok {
		dsnBuilder = dsnBuilder.SetPassword(password)
	}
	if host, ok := os.LookupEnv("DB_HOST"); ok {
		dsnBuilder = dsnBuilder.SetHost(host)
	}
	if portStr, ok := os.LookupEnv("DB_PORT"); ok {
		if port, err := strconv.Atoi(portStr); err == nil {
			dsnBuilder = dsnBuilder.SetPort(uint16(port))
		}
	}
	if databaseName, ok := os.LookupEnv("DB_NAME"); ok {
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
		sqlDB, err := db.DB()
		if err != nil {
			return true, err
		}

		sqlDB.SetMaxOpenConns(getIntEnv("DB_MAX_OPEN_CONNS", 10))
		sqlDB.SetMaxIdleConns(getIntEnv("DB_MAX_IDLE_CONNS", 5))
		sqlDB.SetConnMaxLifetime(getDurationEnv("DB_CONN_MAX_LIFETIME", 5*time.Minute))
		sqlDB.SetConnMaxIdleTime(getDurationEnv("DB_CONN_MAX_IDLE_TIME", 3*time.Minute))

		return false, nil
	}

	err := try(tryToConnect, 10)
	if err != nil {
		l.WithError(err).Fatalf("Failed to connect to database.")
	}

	registerTenantCallbacks(l, db)

	for _, m := range c.migrations {
		err = m(db)
		if err != nil {
			l.WithError(err).Fatalf("Migrating schema.")
		}
	}
	return db
}

func Teardown(l logrus.FieldLogger, db *gorm.DB) func() {
	return func() {
		sqlDB, err := db.DB()
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve sql database.")
			return
		}
		err = sqlDB.Close()
		if err != nil {
			l.WithError(err).Errorf("Unable to close database.")
			return
		}
		l.Infof("Database connection closed.")
	}
}

func getIntEnv(key string, defaultVal int) int {
	if v, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultVal
}

func getDurationEnv(key string, defaultVal time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return defaultVal
}

func try(fn func(attempt int) (retry bool, err error), retries int) error {
	attempt := 1
	for {
		cont, err := fn(attempt)
		if !cont || err == nil {
			return nil
		}
		attempt++
		if attempt > retries {
			return fmt.Errorf("max retry reached: %w", err)
		}
		time.Sleep(1 * time.Second)
	}
}
