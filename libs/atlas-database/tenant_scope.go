package database

import (
	"context"
	"reflect"

	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type contextKey string

const skipTenantFilterKey contextKey = "skip_tenant_filter"

// WithoutTenantFilter returns a context that disables automatic tenant filtering.
// Use this for cross-tenant queries such as startup recovery or admin operations.
func WithoutTenantFilter(ctx context.Context) context.Context {
	return context.WithValue(ctx, skipTenantFilterKey, true)
}

func shouldSkipTenantFilter(ctx context.Context) bool {
	v, ok := ctx.Value(skipTenantFilterKey).(bool)
	return ok && v
}

func hasTenantColumn(db *gorm.DB) bool {
	if db.Statement.Schema == nil {
		return false
	}
	_, ok := db.Statement.Schema.FieldsByDBName["tenant_id"]
	return ok
}

// RegisterTenantCallbacks registers GORM callbacks that automatically inject tenant
// filtering from context. This is called internally by Connect, but is exported for
// use in test setups where the DB is created directly (e.g., SQLite in-memory).
func RegisterTenantCallbacks(l logrus.FieldLogger, db *gorm.DB) {
	registerTenantCallbacks(l, db)
}

func registerTenantCallbacks(l logrus.FieldLogger, db *gorm.DB) {
	db.Callback().Query().Before("gorm:query").Register("tenant:query", tenantQueryCallback(l))
	db.Callback().Row().Before("gorm:row").Register("tenant:row", tenantQueryCallback(l))
	db.Callback().Create().Before("gorm:create").Register("tenant:create", tenantCreateCallback(l))
	db.Callback().Update().Before("gorm:update").Register("tenant:update", tenantQueryCallback(l))
	db.Callback().Delete().Before("gorm:delete").Register("tenant:delete", tenantQueryCallback(l))
}

func tenantQueryCallback(l logrus.FieldLogger) func(db *gorm.DB) {
	return func(db *gorm.DB) {
		if db.Error != nil {
			return
		}

		ctx := db.Statement.Context
		if shouldSkipTenantFilter(ctx) {
			return
		}

		if !hasTenantColumn(db) {
			return
		}

		t, err := tenant.FromContext(ctx)()
		if err != nil {
			l.Debugf("No tenant in context for query on %s, skipping tenant filter.", db.Statement.Schema.Table)
			return
		}

		db.Statement.AddClause(clause.Where{
			Exprs: []clause.Expression{
				clause.Eq{Column: clause.Column{Table: clause.CurrentTable, Name: "tenant_id"}, Value: t.Id()},
			},
		})
	}
}

func tenantCreateCallback(l logrus.FieldLogger) func(db *gorm.DB) {
	return func(db *gorm.DB) {
		if db.Error != nil {
			return
		}

		if !hasTenantColumn(db) {
			return
		}

		ctx := db.Statement.Context
		if shouldSkipTenantFilter(ctx) {
			return
		}

		t, err := tenant.FromContext(ctx)()
		if err != nil {
			return
		}

		field := db.Statement.Schema.FieldsByDBName["tenant_id"]
		if db.Statement.ReflectValue.IsValid() {
			rv := db.Statement.ReflectValue
			switch rv.Kind() {
			case reflect.Struct:
				val, _ := field.ValueOf(ctx, rv)
				if val == nil || val == "" {
					l.Warnf("Creating entity in %s with empty tenant_id while tenant %s is in context.", db.Statement.Schema.Table, t.Id().String())
				}
			case reflect.Slice, reflect.Array:
				for i := 0; i < rv.Len(); i++ {
					val, _ := field.ValueOf(ctx, rv.Index(i))
					if val == nil || val == "" {
						l.Warnf("Creating entity in %s with empty tenant_id while tenant %s is in context.", db.Statement.Schema.Table, t.Id().String())
						break
					}
				}
			}
		}
	}
}
