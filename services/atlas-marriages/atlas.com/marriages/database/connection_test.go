package database

import (
	"testing"
	"gorm.io/gorm"
)

func TestNewDSNBuilder(t *testing.T) {
	builder := NewDSNBuilder()
	
	if builder == nil {
		t.Error("NewDSNBuilder() returned nil")
	}
	
	if builder.user != "" {
		t.Errorf("Expected user to be empty, got %s", builder.user)
	}
	
	if builder.password != "" {
		t.Errorf("Expected password to be empty, got %s", builder.password)
	}
	
	if builder.host != "" {
		t.Errorf("Expected host to be empty, got %s", builder.host)
	}
	
	if builder.port != 0 {
		t.Errorf("Expected port to be 0, got %d", builder.port)
	}
	
	if builder.databaseName != "" {
		t.Errorf("Expected databaseName to be empty, got %s", builder.databaseName)
	}
}

func TestDSNBuilder_Setters(t *testing.T) {
	builder := NewDSNBuilder()
	
	// Test SetUser
	builder = builder.SetUser("testuser")
	if builder.user != "testuser" {
		t.Errorf("Expected user to be 'testuser', got %s", builder.user)
	}
	
	// Test SetPassword
	builder = builder.SetPassword("testpass")
	if builder.password != "testpass" {
		t.Errorf("Expected password to be 'testpass', got %s", builder.password)
	}
	
	// Test SetHost
	builder = builder.SetHost("localhost")
	if builder.host != "localhost" {
		t.Errorf("Expected host to be 'localhost', got %s", builder.host)
	}
	
	// Test SetPort
	builder = builder.SetPort(5432)
	if builder.port != 5432 {
		t.Errorf("Expected port to be 5432, got %d", builder.port)
	}
	
	// Test SetDatabaseName
	builder = builder.SetDatabaseName("testdb")
	if builder.databaseName != "testdb" {
		t.Errorf("Expected databaseName to be 'testdb', got %s", builder.databaseName)
	}
}

func TestDSNBuilder_Build(t *testing.T) {
	builder := NewDSNBuilder().
		SetUser("testuser").
		SetPassword("testpass").
		SetHost("localhost").
		SetPort(5432).
		SetDatabaseName("testdb")
	
	dsn := builder.Build()
	expected := "host=localhost user=testuser password=testpass dbname=testdb port=5432 sslmode=disable TimeZone=UTC"
	
	if dsn != expected {
		t.Errorf("Expected DSN to be '%s', got '%s'", expected, dsn)
	}
}

func TestDSNBuilder_BuildWithDefaults(t *testing.T) {
	builder := NewDSNBuilder()
	
	dsn := builder.Build()
	expected := "host= user= password= dbname= port=0 sslmode=disable TimeZone=UTC"
	
	if dsn != expected {
		t.Errorf("Expected DSN to be '%s', got '%s'", expected, dsn)
	}
}

func TestSetMigrations(t *testing.T) {
	mockMigrator := func(db *gorm.DB) error {
		return nil
	}
	
	config := &Configuration{
		dsn:        "test",
		migrations: make([]Migrator, 0),
	}
	
	configurator := SetMigrations(mockMigrator)
	configurator(config)
	
	if len(config.migrations) != 1 {
		t.Errorf("Expected 1 migration, got %d", len(config.migrations))
	}
}

func TestSetMigrations_Multiple(t *testing.T) {
	mockMigrator1 := func(db *gorm.DB) error {
		return nil
	}
	mockMigrator2 := func(db *gorm.DB) error {
		return nil
	}
	
	config := &Configuration{
		dsn:        "test",
		migrations: make([]Migrator, 0),
	}
	
	configurator := SetMigrations(mockMigrator1, mockMigrator2)
	configurator(config)
	
	if len(config.migrations) != 2 {
		t.Errorf("Expected 2 migrations, got %d", len(config.migrations))
	}
}