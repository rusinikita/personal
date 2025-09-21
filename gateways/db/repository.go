package db

import (
	"context"

	"personal/gateways"
)

// postgres is pgx.Conn methods abstraction
type postgres interface {
	Ping(ctx context.Context) error
}
type repository struct {
	db postgres
}

var _ gateways.DB = (*repository)(nil)
var _ gateways.DBMaintainer = (*repository)(nil)

func NewRepository(db postgres) (gateways.DB, gateways.DBMaintainer) {

	r := &repository{db: db}
	return r, r
}
