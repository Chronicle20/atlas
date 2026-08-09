package opening

import (
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// postgresDuplicateKeySQLState is Postgres SQLSTATE 23505 (unique_violation) —
// https://www.postgresql.org/docs/current/errcodes-appendix.html.
const postgresDuplicateKeySQLState = "23505"

// sqliteDuplicateKeyExtendedCodes mirrors the mapping gorm.io/driver/sqlite's
// own error_translator.go uses for gorm.ErrDuplicatedKey: SQLITE_CONSTRAINT_PRIMARYKEY
// (1555) and SQLITE_CONSTRAINT_UNIQUE (2067) — https://www.sqlite.org/rescode.html.
var sqliteDuplicateKeyExtendedCodes = map[int]bool{
	1555: true,
	2067: true,
}

// sqliteErrMessage matches the exported fields of mattn/go-sqlite3's Error
// type (Code, ExtendedCode, SystemErrno as plain int-kind fields with no
// custom MarshalJSON), so json.Marshal/Unmarshal round-trips it without a
// direct type-assert. Detecting it this way — instead of importing
// mattn/go-sqlite3 and type-asserting *sqlite3.Error — avoids requiring
// CGO_ENABLED for this package, exactly as gorm's own sqlite translator does.
type sqliteErrMessage struct {
	Code         int `json:"Code"`
	ExtendedCode int `json:"ExtendedCode"`
	SystemErrno  int `json:"SystemErrno"`
}

// isDuplicateKeyError reports whether err is a primary-key/unique-constraint
// violation from either driver this service runs against: Postgres in
// production, sqlite in tests. This detection is local to the opening
// package (the shared database connector does not enable gorm's
// TranslateError) rather than a repo-wide change, so it deliberately
// duplicates — not depends on — the two drivers' own translation logic.
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == postgresDuplicateKeySQLState
	}

	parsed, marshalErr := json.Marshal(err)
	if marshalErr != nil {
		return false
	}
	var msg sqliteErrMessage
	if json.Unmarshal(parsed, &msg) != nil {
		return false
	}
	return sqliteDuplicateKeyExtendedCodes[msg.ExtendedCode]
}
