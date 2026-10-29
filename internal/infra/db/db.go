package db

import (
	"fmt"
	"net/url"
	"time"

	"github.com/hangtiancheng/be261028/internal/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// New opens a MySQL connection pool through GORM. The connection location is
// pinned to the configured timezone so DATETIME values round-trip consistently
// with the legacy Asia/Shanghai setup.
func New(cfg config.MySQLConfig) (*gorm.DB, error) {
	loc, err := time.LoadLocation(cfg.LocName)
	if err != nil {
		return nil, fmt.Errorf("load mysql location %q: %w", cfg.LocName, err)
	}

	// clientFoundRows makes UPDATE report matched rows instead of changed rows,
	// matching the legacy JDBC default (useAffectedRows=false). Without it an
	// update that writes identical values (e.g. a same-second timestamp) reports
	// 0 rows and the repository would wrongly treat it as a failure.
	params := url.Values{}
	params.Set("charset", "utf8mb4")
	params.Set("parseTime", "true")
	params.Set("loc", cfg.LocName)
	params.Set("clientFoundRows", "true")
	params.Set("timeout", "30s")
	params.Set("readTimeout", "60s")
	params.Set("writeTimeout", "60s")
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database, params.Encode())

	gdb, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		// Translate driver errors so callers can test gorm.ErrDuplicatedKey
		// instead of matching MySQL error numbers.
		TranslateError: true,
		// SQL logging stays off to mirror the previous database/sql setup.
		Logger:  logger.Default.LogMode(logger.Silent),
		NowFunc: func() time.Time { return time.Now().In(loc) },
	})
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}

	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, fmt.Errorf("unwrap mysql pool: %w", err)
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(10 * time.Minute)
	sqlDB.SetConnMaxIdleTime(time.Minute)

	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping mysql: %w", err)
	}
	return gdb, nil
}

// Close releases the underlying connection pool.
func Close(gdb *gorm.DB) error {
	sqlDB, err := gdb.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
