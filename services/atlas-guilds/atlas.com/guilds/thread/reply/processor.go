package reply

import (
	"context"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func Add(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(threadId uint32, posterId uint32, message string) (Model, error) {
	return func(ctx context.Context) func(db *gorm.DB) func(threadId uint32, posterId uint32, message string) (Model, error) {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(threadId uint32, posterId uint32, message string) (Model, error) {
			return func(threadId uint32, posterId uint32, message string) (Model, error) {
				return create(db, t.Id(), threadId, posterId, message)
			}
		}
	}
}

func Delete(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(threadId uint32, replyId uint32) error {
	return func(ctx context.Context) func(db *gorm.DB) func(threadId uint32, replyId uint32) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(threadId uint32, replyId uint32) error {
			return func(threadId uint32, replyId uint32) error {
				return remove(db, t.Id(), threadId, replyId)
			}
		}
	}
}
