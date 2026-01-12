package storage

import (
	"context"
	"database/sql"
	"fmt"
	"gofast/service-core/config"
	"net"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

type Storage struct {
	Conn *sql.DB
}

func NewStorage(cfg *config.Config) (*Storage, func(), error) {
	host := net.JoinHostPort(cfg.PostgresHost, cfg.PostgresPort)
	url := fmt.Sprintf(
		"postgres://%s:%s@%s/%s?sslmode=disable",
		cfg.PostgresUser,
		cfg.PostgresPassword,
		host,
		cfg.PostgresDB,
	)

	dbpool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		return nil, nil, fmt.Errorf("error opening connection: %w", err)
	}

	db := stdlib.OpenDBFromPool(dbpool)

	err = db.PingContext(context.Background())
	if err != nil {
		dbpool.Close()
		return nil, nil, fmt.Errorf("error pinging database: %w", err)
	}

	var clean = func() {
		dbpool.Close()
	}

	return &Storage{Conn: db}, clean, nil
}
