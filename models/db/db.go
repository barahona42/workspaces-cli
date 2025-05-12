package db

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"strings"
	"time"
	"workspaces-cli/pkg/workspaces"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

var (
	database *sql.DB = nil
	//go:embed resources/createTable_checkpoints.sql
	createTable_checkpoints string
	//go:embed resources/createTable_workspaces.sql
	createTable_workspaces string
)

func createWorkspacesTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, createTable_workspaces)
	return err
}
func createCheckpointsTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, createTable_checkpoints)
	return err
}

func Open(ctx context.Context, file string) error {
	if database == nil {
		db, err := sql.Open("sqlite3", file)
		if err != nil {
			return err
		}
		database = db
		return errors.Join(createWorkspacesTable(ctx, db), createCheckpointsTable(ctx, db))
	}
	return nil
}

func Close() error {
	if database == nil {
		return nil
	}
	return database.Close()
}

func getWorkspaceId(ctx context.Context, w *workspaces.Workspace) (string, error) {
	q := fmt.Sprintf("select id from workspaces where name == '%s'", w.DirEntry.Name())
	row := database.QueryRowContext(ctx, q)
	var wid string
	err := row.Scan(&wid)
	return wid, err
}

func insertWorkspace(ctx context.Context, w *workspaces.Workspace) (string, error) {
	if w == nil {
		return "", fmt.Errorf("workspace is nil")
	}
	wid := uuid.New().String()
	q := "insert into workspaces (id, name, path) values(?, ?, ?)"
	_, err := database.ExecContext(ctx, q, wid, w.DirEntry.Name(), w.Path())
	if err != nil {
		return "", fmt.Errorf("exec query: %w", err)
	}
	return wid, nil
}

func insertCheckpoint(ctx context.Context, wid string, data []byte) (string, error) {
	if wid == "" {
		return "", fmt.Errorf("empty workspace id")
	}
	cid := uuid.New().String()
	q := "insert into checkpoints (id, workspaceid, value, date) values(?, ?, ?, ?)"
	_, err := database.ExecContext(ctx, q, cid, wid, strings.TrimSpace(string(data)), time.Now().In(time.UTC).Unix())
	if err != nil {
		return "", fmt.Errorf("exec query: %w", err)
	}
	return cid, nil
}
func InsertCheckpoint(ctx context.Context, w workspaces.Workspace, data []byte) error {
	wid, err := getWorkspaceId(ctx, &w)
	if errors.Is(err, sql.ErrNoRows) {
		wwid, wierr := insertWorkspace(ctx, &w)
		if wierr != nil {
			return fmt.Errorf("insert workspace: %w", err)
		}
		wid = wwid
	} else if err != nil {
		return fmt.Errorf("get workspace id: %w", err)
	}
	if wid == "" {
		return fmt.Errorf("workspace id is empty: '%s'", w.DirEntry.Name())
	}
	_, err = insertCheckpoint(ctx, wid, data)
	return err
}
