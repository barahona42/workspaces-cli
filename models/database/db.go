package db

import (
	"context"
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

var (
	database *sql.DB = nil
)

func createWorkspacesTable(ctx context.Context, db *sql.DB) error {
	q := "CREATE TABLE IF NOT EXISTS WORKSPACES(id unique,name,path)"
	_, err := db.ExecContext(ctx, q)
	return err
}

func Open(ctx context.Context, file string) error {
	if database == nil {
		db, err := sql.Open("sqlite3", file)
		if err != nil {
			return err
		}
		database = db
		return createWorkspacesTable(ctx, db)
	}
	return nil
}

func Close() error {
	if database == nil {
		return nil
	}
	return database.Close()
}
