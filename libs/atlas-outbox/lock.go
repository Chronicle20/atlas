package outbox

import (
	"context"
	"database/sql"

	"gorm.io/gorm"
)

const advisoryLockKey int64 = 0x4f7574626f78 // 'Outbox'

func tryAdvisoryLock(ctx context.Context, db *gorm.DB) (locker, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return locker{}, err
	}
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		return locker{}, err
	}
	var got bool
	if err := conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", advisoryLockKey).Scan(&got); err != nil {
		_ = conn.Close()
		return locker{}, err
	}
	if !got {
		_ = conn.Close()
		return locker{}, nil
	}
	return locker{conn: conn, held: true}, nil
}

type locker struct {
	conn *sql.Conn
	held bool
}

func (l locker) Held() bool { return l.held }

func (l locker) Release(ctx context.Context) {
	if l.conn == nil {
		return
	}
	_, _ = l.conn.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", advisoryLockKey)
	_ = l.conn.Close()
}
