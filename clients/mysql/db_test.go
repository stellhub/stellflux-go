package mysql

import (
	"context"
	"testing"

	"github.com/stellhub/stellar/config"
)

func TestNewDBFromConfig(t *testing.T) {
	db, err := NewDBFromConfig(context.Background(), &config.MySQLConfig{
		DSN:             "user:pass@tcp(localhost:3306)/app?parseTime=true",
		MaxOpenConns:    10,
		MaxIdleConns:    2,
		ConnMaxLifetime: "30m",
		ConnMaxIdleTime: "5m",
	}, nil)
	if err != nil {
		t.Fatalf("new db from config: %v", err)
	}
	defer db.Close()

	if db.SQLDB() == nil {
		t.Fatalf("expected sql db")
	}
}

func TestNewDBFromConfigRequiresDSN(t *testing.T) {
	_, err := NewDBFromConfig(context.Background(), &config.MySQLConfig{}, nil)
	if err == nil {
		t.Fatalf("expected missing dsn error")
	}
}

func TestNewDBFromConfigRejectsInvalidPoolDuration(t *testing.T) {
	_, err := NewDBFromConfig(context.Background(), &config.MySQLConfig{
		DSN:             "user:pass@tcp(localhost:3306)/app?parseTime=true",
		ConnMaxLifetime: "later",
	}, nil)
	if err == nil {
		t.Fatalf("expected invalid pool duration error")
	}
}
