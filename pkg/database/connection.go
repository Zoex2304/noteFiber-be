package database

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

func ConnectDB(connectionString string) *pgxpool.Pool {
	var err error
	db, err := pgxpool.New(context.Background(), connectionString)
	if err != nil {
		log.Panicf("Unable to connect to DB: %v", err)
	}

	err = db.Ping(context.Background())
	if err != nil {
		log.Panicf("Failed to ping: %v", err)
	}

	return db
}
