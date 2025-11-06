package database

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"time"
)

type PostgresConn struct {
	dbConnStr       string
	maxOpenConns    int
	maxIdleConns    int
	maxConnLifetime time.Duration
	maxConnIdleTime time.Duration
}

func NewPostgresConn(dbConnStr string, maxOpenConns int, maxIdleConns int, maxConnLifetime time.Duration, maxConnIdleTime time.Duration) *PostgresConn {
	return &PostgresConn{
		dbConnStr:       dbConnStr,
		maxOpenConns:    maxOpenConns,
		maxIdleConns:    maxIdleConns,
		maxConnLifetime: maxConnLifetime,
		maxConnIdleTime: maxConnIdleTime,
	}
}

func (p *PostgresConn) Connect() (*sqlx.DB, error) {

	db, err := sqlx.Open("postgres", p.dbConnStr)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(p.maxOpenConns)
	db.SetMaxIdleConns(p.maxIdleConns)
	db.SetConnMaxLifetime(p.maxConnLifetime)
	db.SetConnMaxIdleTime(p.maxConnIdleTime)

	if err = db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}
