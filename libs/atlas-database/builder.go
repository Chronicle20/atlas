package database

import (
	"fmt"
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
