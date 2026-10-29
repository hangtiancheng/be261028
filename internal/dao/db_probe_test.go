package dao

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hangtiancheng/be261028/internal/config"
	"github.com/hangtiancheng/be261028/internal/infra/db"
	"gorm.io/gorm"
)

// openProbeDB connects to the local docker MySQL and creates an isolated probe
// table so these tests never touch domain tables. Skipped when MySQL is absent so
// `go test ./...` still works without the docker infra.
func openProbeDB(t *testing.T) (*gorm.DB, context.Context) {
	t.Helper()
	gdb, err := db.New(config.MySQLConfig{
		Host: "127.0.0.1", Port: 3306, User: "root", Password: "pass",
		Database: "be261028", MaxOpenConns: 2, MaxIdleConns: 1, LocName: "Asia/Shanghai",
	})
	if err != nil {
		t.Skipf("mysql unavailable: %v", err)
	}
	t.Cleanup(func() { _ = db.Close(gdb) })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	ddl := `create table if not exists dao_probe (
		id bigint unsigned not null auto_increment,
		k varchar(16) not null,
		v int not null,
		ts datetime not null,
		primary key (id),
		unique key uq_k (k)
	) engine=InnoDB default charset=utf8mb4`
	if err := gdb.WithContext(ctx).Exec(ddl).Error; err != nil {
		t.Skipf("cannot create probe table: %v", err)
	}
	if err := gdb.WithContext(ctx).Exec("delete from dao_probe").Error; err != nil {
		t.Fatalf("clean probe table: %v", err)
	}
	t.Cleanup(func() { _ = gdb.Exec("drop table if exists dao_probe").Error })
	return gdb, ctx
}

// TestClientFoundRowsParity guards the GORM DSN setting that reproduces the legacy
// JDBC behaviour: an UPDATE writing an unchanged value must still report the row.
// Without clientFoundRows MySQL reports 0 changed rows and every repository update
// (e.g. re-enabling an already enabled activity) would be treated as a failure.
func TestClientFoundRowsParity(t *testing.T) {
	gdb, ctx := openProbeDB(t)

	type probeRow struct {
		Id int64     `gorm:"column:id;primaryKey"`
		K  string    `gorm:"column:k"`
		V  int       `gorm:"column:v"`
		Ts time.Time `gorm:"column:ts"`
	}

	now := time.Now().Truncate(time.Second)
	if err := gdb.WithContext(ctx).Exec(
		"insert into dao_probe (k, v, ts) values (?, ?, ?)", "idem", 1, now,
	).Error; err != nil {
		t.Fatalf("insert probe row: %v", err)
	}

	// Same value as stored: must report one matched row.
	res := gdb.WithContext(ctx).Table("dao_probe").
		Where("k = ?", "idem").
		Updates(map[string]any{"v": 1, "ts": now})
	if res.Error != nil {
		t.Fatalf("update unchanged value: %v", res.Error)
	}
	if res.RowsAffected != 1 {
		t.Fatalf("unchanged update RowsAffected = %d, want 1 (clientFoundRows)", res.RowsAffected)
	}

	// Guarded update that matches nothing still reports zero.
	res = gdb.WithContext(ctx).Table("dao_probe").
		Where("k = ? AND v = ?", "idem", 99).
		Updates(map[string]any{"v": 2})
	if res.RowsAffected != 0 {
		t.Fatalf("non-matching update RowsAffected = %d, want 0", res.RowsAffected)
	}

	// DATETIME round-trips in the configured location (Asia/Shanghai).
	var got probeRow
	if err := gdb.WithContext(ctx).Table("dao_probe").Where("k = ?", "idem").Take(&got).Error; err != nil {
		t.Fatalf("read probe row: %v", err)
	}
	if !got.Ts.Equal(now) {
		t.Fatalf("timestamp round-trip = %v, want %v", got.Ts, now)
	}
}

// TestDuplicatedKeyTranslation verifies gorm.Config.TranslateError maps MySQL 1062
// to gorm.ErrDuplicatedKey, which dao.IsDuplicateKey relies on for the idempotent
// crowd-tag insert and the lock-order biz_id guard.
func TestDuplicatedKeyTranslation(t *testing.T) {
	gdb, ctx := openProbeDB(t)

	insert := func() error {
		return gdb.WithContext(ctx).Exec(
			"insert into dao_probe (k, v, ts) values (?, ?, now())", "dup", 1,
		).Error
	}
	if err := insert(); err != nil {
		t.Fatalf("first insert: %v", err)
	}
	err := insert()
	if err == nil {
		t.Fatal("second insert unexpectedly succeeded")
	}
	if !IsDuplicateKey(err) {
		t.Fatalf("IsDuplicateKey(%v) = false, want true", err)
	}
	if !errors.Is(err, gorm.ErrDuplicatedKey) {
		t.Fatal("error is not gorm.ErrDuplicatedKey")
	}
}

// TestRecordNotFoundMapping verifies single-row queries surface dao.ErrNotFound so
// repositories can keep the legacy "null means absent" behaviour.
func TestRecordNotFoundMapping(t *testing.T) {
	gdb, ctx := openProbeDB(t)
	d := New()

	if _, err := d.QueryGroupBuyActivityByActivityId(ctx, gdb, -1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing activity error = %v, want ErrNotFound", err)
	}
	if _, err := d.QueryValidGroupBuyActivityId(ctx, gdb, -1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing valid activity error = %v, want ErrNotFound", err)
	}
	if _, err := d.QueryUnExecutedNotifyTaskByTeamId(ctx, gdb, "no-such-team"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing notify task error = %v, want ErrNotFound", err)
	}
}
