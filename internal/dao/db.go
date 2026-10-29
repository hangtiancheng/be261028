package dao

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// ErrNotFound is returned by single-row queries that matched nothing. Callers map
// it to nil where the legacy code returned null.
var ErrNotFound = errors.New("dao: no rows found")

// DAO bundles the GORM queries for the be261028 schema.
type DAO struct{}

// New returns a DAO. Queries receive a *gorm.DB so repositories control
// transactionality.
func New() *DAO { return &DAO{} }

// IsDuplicateKey reports whether err is a unique-constraint violation, the Go
// equivalent of Spring's DuplicateKeyException. The connection enables
// gorm.Config.TranslateError, which maps MySQL error 1062 to ErrDuplicatedKey.
func IsDuplicateKey(err error) bool {
	return errors.Is(err, gorm.ErrDuplicatedKey)
}

// recordNotFound normalises GORM's sentinel error to ErrNotFound.
func recordNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}

// checkAffected validates that exactly want rows were affected.
func checkAffected(res *gorm.DB, want int64) error {
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected != want {
		return fmt.Errorf("dao: affected %d rows, want %d", res.RowsAffected, want)
	}
	return nil
}
